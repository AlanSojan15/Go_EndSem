package portfolio

import (
	customerrors "crypto-portfolio-tracker/errors"
	"crypto-portfolio-tracker/models"
	"errors"
	"sync"
	"testing"
	"time"
)

type mockAPI struct {
	prices map[string]float64
	err    error
}

func (m *mockAPI) FetchPrice(coinID string) (float64, error) {
	if m.err != nil {
		return 0, m.err
	}
	p, ok := m.prices[coinID]
	if !ok {
		return 0, customerrors.NewAPIError("simple/price", 0, customerrors.ErrPriceNotAvailable)
	}
	return p, nil
}

func (m *mockAPI) FetchMultiplePrices(coinIDs ...string) (map[string]float64, error) {
	if m.err != nil {
		return nil, m.err
	}
	result := make(map[string]float64, len(coinIDs))
	for _, id := range coinIDs {
		if p, ok := m.prices[id]; ok {
			result[id] = p
		}
	}
	return result, nil
}

func (m *mockAPI) GetSupportedCoins() (map[string]string, error) {
	return map[string]string{"bitcoin": "Bitcoin", "ethereum": "Ethereum"}, nil
}

type slowMockAPI struct {
	mockAPI
	delay time.Duration
}

func (s *slowMockAPI) FetchPrice(coinID string) (float64, error) {
	time.Sleep(s.delay)
	return s.mockAPI.FetchPrice(coinID)
}

func (s *slowMockAPI) FetchMultiplePrices(coinIDs ...string) (map[string]float64, error) {
	time.Sleep(s.delay)
	return s.mockAPI.FetchMultiplePrices(coinIDs...)
}

func makePortfolio(holdings ...models.Holding) *models.Portfolio {
	return &models.Portfolio{
		UserEmail: "test@example.com",
		Holdings:  holdings,
		UpdatedAt: time.Now(),
	}
}

func holding(coinID, coinName string, qty, buyPrice float64) models.Holding {
	return models.Holding{
		CoinID:   coinID,
		CoinName: coinName,
		Quantity: qty,
		BuyPrice: buyPrice,
		AddedAt:  time.Now(),
	}
}

func TestCalculateTotalValue_SingleHolding(t *testing.T) {
	p := makePortfolio(holding("bitcoin", "Bitcoin", 2, 30000))
	api := &mockAPI{prices: map[string]float64{"bitcoin": 50000}}

	total, err := CalculateTotalValue(p, api)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 100000 {
		t.Errorf("total: got %.2f, want 100000.00", total)
	}
}

func TestCalculateTotalValue_MultipleHoldings(t *testing.T) {
	p := makePortfolio(
		holding("bitcoin", "Bitcoin", 1, 40000),
		holding("ethereum", "Ethereum", 10, 2000),
	)
	api := &mockAPI{prices: map[string]float64{
		"bitcoin":  60000,
		"ethereum": 3000,
	}}

	total, err := CalculateTotalValue(p, api)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 90000 {
		t.Errorf("total: got %.2f, want 90000.00", total)
	}
}

func TestCalculateTotalValue_EmptyPortfolio(t *testing.T) {
	p := makePortfolio()
	api := &mockAPI{prices: map[string]float64{}}

	total, err := CalculateTotalValue(p, api)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 for empty portfolio, got %.2f", total)
	}
}

func TestCalculateTotalValue_APIError(t *testing.T) {
	p := makePortfolio(holding("bitcoin", "Bitcoin", 1, 30000))
	api := &mockAPI{err: customerrors.ErrRateLimitExceeded}

	_, err := CalculateTotalValue(p, api)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, customerrors.ErrRateLimitExceeded) {
		t.Errorf("error chain should contain ErrRateLimitExceeded, got: %v", err)
	}
}

func TestCalculateTotalValue_MissingPrice(t *testing.T) {
	p := makePortfolio(holding("solana", "Solana", 5, 100))
	api := &mockAPI{prices: map[string]float64{"bitcoin": 60000}}

	_, err := CalculateTotalValue(p, api)
	if err == nil {
		t.Fatal("expected error for missing price, got nil")
	}
	if !errors.Is(err, customerrors.ErrPriceNotAvailable) {
		t.Errorf("expected ErrPriceNotAvailable in chain, got: %v", err)
	}
}

func TestCalculateProfitLoss_Profit(t *testing.T) {
	p := makePortfolio(holding("bitcoin", "Bitcoin", 1, 30000))
	api := &mockAPI{prices: map[string]float64{"bitcoin": 60000}}

	pl, err := CalculateProfitLoss(p, api)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pl["bitcoin"] != 30000 {
		t.Errorf("bitcoin P/L: got %.2f, want 30000.00", pl["bitcoin"])
	}
}

func TestCalculateProfitLoss_Loss(t *testing.T) {
	p := makePortfolio(holding("ethereum", "Ethereum", 2, 3000))
	api := &mockAPI{prices: map[string]float64{"ethereum": 1500}}

	pl, err := CalculateProfitLoss(p, api)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pl["ethereum"] != -3000 {
		t.Errorf("ethereum P/L: got %.2f, want -3000.00", pl["ethereum"])
	}
}

func TestCalculateProfitLoss_MultipleCoins(t *testing.T) {
	p := makePortfolio(
		holding("bitcoin", "Bitcoin", 1, 40000),
		holding("ethereum", "Ethereum", 5, 3000),
	)
	api := &mockAPI{prices: map[string]float64{
		"bitcoin":  60000,
		"ethereum": 2000,
	}}

	pl, err := CalculateProfitLoss(p, api)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pl["bitcoin"] != 20000 {
		t.Errorf("bitcoin: got %.2f, want 20000", pl["bitcoin"])
	}
	if pl["ethereum"] != -5000 {
		t.Errorf("ethereum: got %.2f, want -5000", pl["ethereum"])
	}
}

func TestCalculateProfitLoss_SpecificCoins(t *testing.T) {
	p := makePortfolio(
		holding("bitcoin", "Bitcoin", 1, 30000),
		holding("ethereum", "Ethereum", 2, 2000),
	)
	api := &mockAPI{prices: map[string]float64{
		"bitcoin":  50000,
		"ethereum": 3000,
	}}

	pl, err := CalculateProfitLoss(p, api, "bitcoin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := pl["ethereum"]; exists {
		t.Error("ethereum should not be in result when not requested")
	}
	if pl["bitcoin"] != 20000 {
		t.Errorf("bitcoin: got %.2f, want 20000", pl["bitcoin"])
	}
}

func TestCalculateProfitLoss_EmptyPortfolio(t *testing.T) {
	p := makePortfolio()
	api := &mockAPI{prices: map[string]float64{}}

	_, err := CalculateProfitLoss(p, api)
	if err == nil {
		t.Fatal("expected ErrEmptyPortfolio, got nil")
	}
	if !errors.Is(err, customerrors.ErrEmptyPortfolio) {
		t.Errorf("expected ErrEmptyPortfolio, got: %v", err)
	}
}

func TestCalculateProfitLoss_APIError(t *testing.T) {
	p := makePortfolio(holding("bitcoin", "Bitcoin", 1, 30000))
	api := &mockAPI{err: customerrors.ErrRateLimitExceeded}

	_, err := CalculateProfitLoss(p, api)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, customerrors.ErrRateLimitExceeded) {
		t.Errorf("expected ErrRateLimitExceeded in chain, got: %v", err)
	}
}

func TestCalculateProfitLoss_CoinNotInPortfolio(t *testing.T) {
	p := makePortfolio(holding("bitcoin", "Bitcoin", 1, 30000))
	api := &mockAPI{prices: map[string]float64{
		"bitcoin": 50000,
		"solana":  100,
	}}

	pl, err := CalculateProfitLoss(p, api, "bitcoin", "solana")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, exists := pl["solana"]; exists {
		t.Error("solana should not appear since it is not in the portfolio")
	}
	if pl["bitcoin"] != 20000 {
		t.Errorf("bitcoin: got %.2f, want 20000", pl["bitcoin"])
	}
}

func TestPortfolioError_Wrapping(t *testing.T) {
	wrapped := customerrors.NewPortfolioError("fetch", "bitcoin", customerrors.ErrPriceNotAvailable)
	if !errors.Is(wrapped, customerrors.ErrPriceNotAvailable) {
		t.Error("errors.Is should find ErrPriceNotAvailable through PortfolioError")
	}
}

func TestAPIError_Wrapping(t *testing.T) {
	wrapped := customerrors.NewAPIError("simple/price", 429, customerrors.ErrRateLimitExceeded)
	if !errors.Is(wrapped, customerrors.ErrRateLimitExceeded) {
		t.Error("errors.Is should find ErrRateLimitExceeded through APIError")
	}

	var apiErr *customerrors.APIError
	if !errors.As(wrapped, &apiErr) {
		t.Fatal("errors.As should return *APIError")
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("StatusCode: got %d, want 429", apiErr.StatusCode)
	}
}

func TestDatabaseError_Wrapping(t *testing.T) {
	wrapped := customerrors.NewDatabaseError("insert", "portfolios", customerrors.ErrDatabaseConnection)
	if !errors.Is(wrapped, customerrors.ErrDatabaseConnection) {
		t.Error("errors.Is should find ErrDatabaseConnection through DatabaseError")
	}

	var dbErr *customerrors.DatabaseError
	if !errors.As(wrapped, &dbErr) {
		t.Fatal("errors.As should return *DatabaseError")
	}
	if dbErr.Collection != "portfolios" {
		t.Errorf("Collection: got %q, want %q", dbErr.Collection, "portfolios")
	}
}

func TestValidationError_Wrapping(t *testing.T) {
	wrapped := customerrors.NewValidationError("quantity", -1.0, customerrors.ErrInvalidQuantity)
	if !errors.Is(wrapped, customerrors.ErrInvalidQuantity) {
		t.Error("errors.Is should find ErrInvalidQuantity through ValidationError")
	}

	var valErr *customerrors.ValidationError
	if !errors.As(wrapped, &valErr) {
		t.Fatal("errors.As should return *ValidationError")
	}
	if valErr.Field != "quantity" {
		t.Errorf("Field: got %q, want %q", valErr.Field, "quantity")
	}
}

func TestAPIError_ErrorString_WithStatusCode(t *testing.T) {
	err := customerrors.NewAPIError("simple/price", 404, errors.New("not found"))
	got := err.Error()
	want := "API request to simple/price failed with status 404: not found"
	if got != want {
		t.Errorf("error string:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestAPIError_ErrorString_NoStatusCode(t *testing.T) {
	err := customerrors.NewAPIError("simple/price", 0, errors.New("timeout"))
	got := err.Error()
	want := "API request to simple/price failed: timeout"
	if got != want {
		t.Errorf("error string:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestPortfolioError_ErrorString_WithCoin(t *testing.T) {
	err := customerrors.NewPortfolioError("update", "ethereum", errors.New("write failed"))
	got := err.Error()
	want := "portfolio update failed for coin ethereum: write failed"
	if got != want {
		t.Errorf("error string:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestPortfolioError_ErrorString_NoCoin(t *testing.T) {
	err := customerrors.NewPortfolioError("fetch", "", errors.New("db down"))
	got := err.Error()
	want := "portfolio fetch failed: db down"
	if got != want {
		t.Errorf("error string:\n  got:  %q\n  want: %q", got, want)
	}
}

func TestPipeline_StreamHoldings(t *testing.T) {
	holdings := []models.Holding{
		holding("bitcoin", "Bitcoin", 1, 30000),
		holding("ethereum", "Ethereum", 2, 2000),
		holding("solana", "Solana", 5, 100),
	}

	jobsCh := make(chan holdingJob, len(holdings))
	done := make(chan struct{})

	go streamHoldings(holdings, jobsCh, done)

	var received []string
	for job := range jobsCh {
		received = append(received, job.holding.CoinID)
	}

	if len(received) != len(holdings) {
		t.Errorf("got %d jobs, want %d", len(received), len(holdings))
	}
	for i, id := range []string{"bitcoin", "ethereum", "solana"} {
		if received[i] != id {
			t.Errorf("job[%d]: got %q, want %q", i, received[i], id)
		}
	}
}

func TestPipeline_StreamHoldings_Done(t *testing.T) {
	holdings := make([]models.Holding, 100)
	for i := range holdings {
		holdings[i] = holding("bitcoin", "Bitcoin", 1, 30000)
	}

	jobsCh := make(chan holdingJob, 1)
	done := make(chan struct{})

	go streamHoldings(holdings, jobsCh, done)

	<-jobsCh
	close(done)

	count := 1
	for range jobsCh {
		count++
	}
	if count >= 100 {
		t.Error("done channel did not cancel streamHoldings early")
	}
}

func TestPipeline_PriceWorker(t *testing.T) {
	prices := map[string]float64{
		"bitcoin":  60000,
		"ethereum": 3000,
	}
	api := &mockAPI{prices: prices}

	jobsCh := make(chan holdingJob, 2)
	resultCh := make(chan priceResult, 2)
	done := make(chan struct{})

	jobsCh <- holdingJob{holding: holding("bitcoin", "Bitcoin", 1, 30000)}
	jobsCh <- holdingJob{holding: holding("ethereum", "Ethereum", 2, 2000)}
	close(jobsCh)

	var wg sync.WaitGroup
	wg.Add(1)
	go priceWorker(api, prices, jobsCh, resultCh, done, &wg)
	wg.Wait()
	close(resultCh)

	results := make(map[string]priceResult)
	for r := range resultCh {
		results[r.coinID] = r
	}

	if r, ok := results["bitcoin"]; !ok || r.price != 60000 {
		t.Errorf("bitcoin: got price %.2f, want 60000", results["bitcoin"].price)
	}
	if r, ok := results["ethereum"]; !ok || r.price != 3000 {
		t.Errorf("ethereum: got price %.2f, want 3000", results["ethereum"].price)
	}
}

func TestPipeline_PriceWorker_MissingPrice(t *testing.T) {
	prices := map[string]float64{"bitcoin": 60000}
	api := &mockAPI{prices: prices}

	jobsCh := make(chan holdingJob, 1)
	resultCh := make(chan priceResult, 1)
	done := make(chan struct{})

	jobsCh <- holdingJob{holding: holding("solana", "Solana", 5, 100)}
	close(jobsCh)

	var wg sync.WaitGroup
	wg.Add(1)
	go priceWorker(api, prices, jobsCh, resultCh, done, &wg)
	wg.Wait()
	close(resultCh)

	r := <-resultCh
	if r.err == nil {
		t.Fatal("expected error for missing price, got nil")
	}
	if !errors.Is(r.err, customerrors.ErrPriceNotAvailable) {
		t.Errorf("expected ErrPriceNotAvailable, got: %v", r.err)
	}
}

func TestPipeline_RunPricePipeline_CorrectResults(t *testing.T) {
	holdings := []models.Holding{
		holding("bitcoin", "Bitcoin", 1, 30000),
		holding("ethereum", "Ethereum", 5, 2000),
		holding("solana", "Solana", 10, 50),
	}
	prices := map[string]float64{
		"bitcoin":  60000,
		"ethereum": 3000,
		"solana":   100,
	}
	api := &mockAPI{prices: prices}

	resultCh, cancel := runPricePipeline(holdings, prices, api, 3)
	defer cancel()

	results := make(map[string]priceResult)
	for r := range resultCh {
		if r.err != nil {
			t.Errorf("unexpected error for %s: %v", r.coinID, r.err)
		}
		results[r.coinID] = r
	}

	if len(results) != 3 {
		t.Errorf("got %d results, want 3", len(results))
	}
	if results["bitcoin"].price != 60000 {
		t.Errorf("bitcoin price: got %.2f, want 60000", results["bitcoin"].price)
	}
	if results["ethereum"].price != 3000 {
		t.Errorf("ethereum price: got %.2f, want 3000", results["ethereum"].price)
	}
	if results["solana"].price != 100 {
		t.Errorf("solana price: got %.2f, want 100", results["solana"].price)
	}
}

func TestPipeline_CancelStopsPipeline(t *testing.T) {
	holdings := make([]models.Holding, 50)
	for i := range holdings {
		holdings[i] = holding("bitcoin", "Bitcoin", 1, 30000)
	}
	prices := map[string]float64{"bitcoin": 60000}
	api := &mockAPI{prices: prices}

	resultCh, cancel := runPricePipeline(holdings, prices, api, 2)

	cancel()

	for range resultCh {
	}
}

func TestCalculateTotalValue_Concurrent(t *testing.T) {
	p := makePortfolio(
		holding("bitcoin", "Bitcoin", 1, 30000),
		holding("ethereum", "Ethereum", 5, 2000),
		holding("solana", "Solana", 20, 50),
	)
	api := &slowMockAPI{
		mockAPI: mockAPI{prices: map[string]float64{
			"bitcoin":  60000,
			"ethereum": 3000,
			"solana":   100,
		}},
		delay: 5 * time.Millisecond,
	}

	const want = 77000.0
	const goroutines = 10

	errs := make(chan error, goroutines)
	totals := make(chan float64, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			total, err := CalculateTotalValue(p, api)
			if err != nil {
				errs <- err
				return
			}
			totals <- total
		}()
	}
	wg.Wait()
	close(errs)
	close(totals)

	for err := range errs {
		t.Errorf("unexpected error: %v", err)
	}
	for total := range totals {
		if total != want {
			t.Errorf("total: got %.2f, want %.2f", total, want)
		}
	}
}

func TestCalculateProfitLoss_Concurrent(t *testing.T) {
	p := makePortfolio(
		holding("bitcoin", "Bitcoin", 2, 30000),
		holding("ethereum", "Ethereum", 10, 1500),
	)
	api := &slowMockAPI{
		mockAPI: mockAPI{prices: map[string]float64{
			"bitcoin":  50000,
			"ethereum": 2000,
		}},
		delay: 5 * time.Millisecond,
	}

	const goroutines = 10

	type plResult struct {
		pl  map[string]float64
		err error
	}
	ch := make(chan plResult, goroutines)

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pl, err := CalculateProfitLoss(p, api)
			ch <- plResult{pl, err}
		}()
	}
	wg.Wait()
	close(ch)

	for r := range ch {
		if r.err != nil {
			t.Errorf("unexpected error: %v", r.err)
			continue
		}
		if r.pl["bitcoin"] != 40000 {
			t.Errorf("bitcoin P/L: got %.2f, want 40000", r.pl["bitcoin"])
		}
		if r.pl["ethereum"] != 5000 {
			t.Errorf("ethereum P/L: got %.2f, want 5000", r.pl["ethereum"])
		}
	}
}

func TestCalculateTotalValue_ConcurrentWithError(t *testing.T) {
	p := makePortfolio(holding("bitcoin", "Bitcoin", 1, 30000))
	api := &slowMockAPI{
		mockAPI: mockAPI{err: customerrors.ErrRateLimitExceeded},
		delay:   2 * time.Millisecond,
	}

	var wg sync.WaitGroup
	errs := make(chan error, 5)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := CalculateTotalValue(p, api)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err == nil {
			t.Error("expected error, got nil")
			continue
		}
		if !errors.Is(err, customerrors.ErrRateLimitExceeded) {
			t.Errorf("expected ErrRateLimitExceeded, got: %v", err)
		}
	}
}
