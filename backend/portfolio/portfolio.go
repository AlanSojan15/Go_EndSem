package portfolio

import (
	"context"
	"crypto-portfolio-tracker/api"
	"crypto-portfolio-tracker/db"
	customerrors "crypto-portfolio-tracker/errors"
	"crypto-portfolio-tracker/models"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type priceResult struct {
	coinID   string
	coinName string
	price    float64
	quantity float64
	buyPrice float64
	err      error
}

type holdingJob struct {
	holding models.Holding
}

func AddMultipleHoldings(userEmail string, holdings ...models.Holding) error {
	if len(holdings) == 0 {
		return customerrors.ErrEmptyHoldings
	}

	database, err := db.ConnectDatabase()
	if err != nil {
		return customerrors.NewDatabaseError("connect", "portfolios", err)
	}

	collection := database.Collection("portfolios")

	for i := range holdings {
		if holdings[i].Quantity <= 0 {
			return customerrors.NewValidationError("quantity", holdings[i].Quantity, customerrors.ErrInvalidQuantity)
		}
		if holdings[i].BuyPrice <= 0 {
			return customerrors.NewValidationError("buy_price", holdings[i].BuyPrice, customerrors.ErrInvalidPrice)
		}
		// CoinGecko IDs are always lowercase (e.g. "bitcoin", not "Bitcoin").
		holdings[i].CoinID = strings.ToLower(strings.TrimSpace(holdings[i].CoinID))
	}

	for _, h := range holdings {
		h.AddedAt = time.Now()

		// Try to increment quantity if the coin already exists in the portfolio.
		filter := bson.M{"user_email": userEmail, "holdings.coin_id": h.CoinID}
		update := bson.M{
			"$inc": bson.M{"holdings.$.quantity": h.Quantity},
			"$set": bson.M{"updated_at": time.Now()},
		}

		result, err := collection.UpdateOne(context.TODO(), filter, update)
		if err != nil {
			return customerrors.NewDatabaseError("update", "portfolios", err)
		}

		if result.MatchedCount == 0 {
			// Coin not in portfolio yet — push a new entry.
			filter = bson.M{"user_email": userEmail}
			update = bson.M{
				"$push": bson.M{"holdings": h},
				"$set":  bson.M{"updated_at": time.Now()},
			}
			_, err = collection.UpdateOne(
				context.TODO(),
				filter,
				update,
				options.Update().SetUpsert(true),
			)
			if err != nil {
				return customerrors.NewPortfolioError("add holding", h.CoinID, err)
			}
		}
	}

	return nil
}

func GetPortfolio(userEmail string) (*models.Portfolio, error) {
	database, err := db.ConnectDatabase()
	if err != nil {
		return nil, customerrors.NewDatabaseError("connect", "portfolios", err)
	}

	var portfolio models.Portfolio
	collection := database.Collection("portfolios")

	err = collection.FindOne(
		context.TODO(),
		bson.M{"user_email": userEmail},
	).Decode(&portfolio)

	if err == mongo.ErrNoDocuments {
		return &models.Portfolio{
			UserEmail: userEmail,
			Holdings:  []models.Holding{},
			UpdatedAt: time.Now(),
		}, nil
	}

	if err != nil {
		return nil, customerrors.NewDatabaseError("fetch", "portfolios", err)
	}

	return &portfolio, nil
}

func streamHoldings(holdings []models.Holding, jobsCh chan<- holdingJob, done <-chan struct{}) {
	defer close(jobsCh)
	for _, h := range holdings {
		select {
		case jobsCh <- holdingJob{holding: h}:
		case <-done:
			return
		}
	}
}
func priceWorker(
	apiClient api.CryptoApi,
	prices map[string]float64,
	jobsCh <-chan holdingJob,
	resultCh chan<- priceResult,
	done <-chan struct{},
	wg *sync.WaitGroup,
) {
	defer wg.Done()
	for job := range jobsCh {
		h := job.holding
		price, ok := prices[h.CoinID]

		var r priceResult
		if !ok {
			r = priceResult{
				coinID: h.CoinID,
				err:    customerrors.NewPortfolioError("price lookup", h.CoinID, customerrors.ErrPriceNotAvailable),
			}
		} else {
			r = priceResult{
				coinID:   h.CoinID,
				coinName: h.CoinName,
				price:    price,
				quantity: h.Quantity,
				buyPrice: h.BuyPrice,
			}
		}

		select {
		case resultCh <- r:
		case <-done:
			return
		}
	}
}

func runPricePipeline(
	holdings []models.Holding,
	prices map[string]float64,
	apiClient api.CryptoApi,
	numWorkers int,
) (<-chan priceResult, func()) {
	jobsCh := make(chan holdingJob, len(holdings))
	resultCh := make(chan priceResult, len(holdings))
	done := make(chan struct{})

	cancel := func() {
		select {
		case <-done:
		default:
			close(done)
		}
	}

	go streamHoldings(holdings, jobsCh, done)

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go priceWorker(apiClient, prices, jobsCh, resultCh, done, &wg)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	return resultCh, cancel
}

func CalculateTotalValue(portfolio *models.Portfolio, apiClient api.CryptoApi) (float64, error) {
	if len(portfolio.Holdings) == 0 {
		return 0, nil
	}

	coinIDs := make([]string, len(portfolio.Holdings))
	for i, h := range portfolio.Holdings {
		coinIDs[i] = h.CoinID
	}

	prices, err := apiClient.FetchMultiplePrices(coinIDs...)
	if err != nil {
		return 0, customerrors.NewPortfolioError("calculate total value", "", err)
	}

	numWorkers := len(portfolio.Holdings)
	resultCh, cancel := runPricePipeline(portfolio.Holdings, prices, apiClient, numWorkers)
	defer cancel()

	var total float64
	for r := range resultCh {
		if r.err != nil {
			cancel()
			return 0, r.err
		}
		total += r.price * r.quantity
	}

	return total, nil
}

func CalculateProfitLoss(portfolio *models.Portfolio, apiClient api.CryptoApi, coinIDs ...string) (map[string]float64, error) {
	if len(coinIDs) == 0 {
		for _, h := range portfolio.Holdings {
			coinIDs = append(coinIDs, h.CoinID)
		}
	}

	if len(coinIDs) == 0 {
		return map[string]float64{}, customerrors.ErrEmptyPortfolio
	}

	holdingsMap := make(map[string]models.Holding, len(portfolio.Holdings))
	for _, h := range portfolio.Holdings {
		holdingsMap[h.CoinID] = h
	}

	requested := make([]models.Holding, 0, len(coinIDs))
	for _, id := range coinIDs {
		if h, ok := holdingsMap[id]; ok {
			requested = append(requested, h)
		}
	}

	prices, err := apiClient.FetchMultiplePrices(coinIDs...)
	if err != nil {
		return nil, customerrors.NewPortfolioError("calculate profit/loss", "", err)
	}

	numWorkers := len(requested)
	if numWorkers == 0 {
		return map[string]float64{}, nil
	}

	resultCh, cancel := runPricePipeline(requested, prices, apiClient, numWorkers)
	defer cancel()

	profitLoss := make(map[string]float64, len(requested))
	for r := range resultCh {
		if r.err != nil {
			cancel()
			return nil, r.err
		}
		invested := r.buyPrice * r.quantity
		current := r.price * r.quantity
		profitLoss[r.coinID] = current - invested
	}

	return profitLoss, nil
}

func DisplayPortfolio(userEmail string, apiClient api.CryptoApi) error {
	portfolio, err := GetPortfolio(userEmail)
	if err != nil {
		return customerrors.NewPortfolioError("display portfolio", "", err)
	}

	if len(portfolio.Holdings) == 0 {
		fmt.Println("Your portfolio is empty.")
		return nil
	}

	coinIDs := make([]string, len(portfolio.Holdings))
	for i, h := range portfolio.Holdings {
		coinIDs[i] = h.CoinID
	}
	prices, err := apiClient.FetchMultiplePrices(coinIDs...)
	if err != nil {
		return customerrors.NewPortfolioError("display portfolio", "", err)
	}

	numWorkers := len(portfolio.Holdings)
	resultCh, cancel := runPricePipeline(portfolio.Holdings, prices, apiClient, numWorkers)
	defer cancel()

	resultMap := make(map[string]priceResult, len(portfolio.Holdings))
	for r := range resultCh {
		if r.err != nil {
			fmt.Printf("Warning: Could not fetch price for %s\n", r.coinID)
			continue
		}
		resultMap[r.coinID] = r
	}

	fmt.Println("\n========== YOUR PORTFOLIO ==========")

	var totalValue float64
	for _, h := range portfolio.Holdings {
		r, ok := resultMap[h.CoinID]
		if !ok {
			fmt.Printf("Warning: No result for %s\n", h.CoinName)
			continue
		}

		currentValue := r.price * r.quantity
		invested := r.buyPrice * r.quantity
		profitLoss := currentValue - invested
		profitLossPercent := (profitLoss / invested) * 100
		totalValue += currentValue

		fmt.Printf("\nCoin: %s (%s)\n", h.CoinName, h.CoinID)
		fmt.Printf("  Quantity      : %.4f\n", h.Quantity)
		fmt.Printf("  Buy Price     : $%.2f\n", h.BuyPrice)
		fmt.Printf("  Current Price : $%.2f\n", r.price)
		fmt.Printf("  Current Value : $%.2f\n", currentValue)
		fmt.Printf("  Profit/Loss   : $%.2f (%.2f%%)\n", profitLoss, profitLossPercent)
	}

	fmt.Printf("\n====================================\n")
	fmt.Printf("Total Portfolio Value: $%.2f\n", totalValue)
	fmt.Printf("====================================\n\n")

	return nil
}
