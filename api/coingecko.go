package api

import (
	customerrors "crypto-portfolio-tracker/errors"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type CoinGecko struct {
	BaseURL         string
	Client          *http.Client
	mu              sync.Mutex
	lastRequestTime time.Time
	minDelay        time.Duration
}

func NewCoinGecko() (*CoinGecko, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}
	url := os.Getenv("URL")
	if url == "" {
		return nil, fmt.Errorf("URL not set in environment")
	}

	return &CoinGecko{
		BaseURL: url,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

func (cg *CoinGecko) waitForRateLimit() {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	if !cg.lastRequestTime.IsZero() {
		if elapsed := time.Since(cg.lastRequestTime); elapsed < cg.minDelay {
			time.Sleep(cg.minDelay - elapsed)
		}
	}
	cg.lastRequestTime = time.Now()
}

func (cg *CoinGecko) FetchMultiplePrices(coinIDs ...string) (map[string]float64, error) {
	if len(coinIDs) == 0 {
		return nil, customerrors.NewValidationError("coinIDs", coinIDs, customerrors.ErrEmptyHoldings)
	}

	cg.waitForRateLimit()

	coinList := strings.Join(coinIDs, ",")
	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd", cg.BaseURL, coinList)

	resp, err := cg.Client.Get(url)
	if err != nil {
		return nil, customerrors.NewAPIError("simple/price", 0, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, customerrors.NewAPIError("simple/price", 429, customerrors.ErrRateLimitExceeded)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, customerrors.NewAPIError("simple/price", resp.StatusCode, errors.New("failed to fetch prices"))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, customerrors.NewAPIError("simple/price", 0, fmt.Errorf("failed to read response: %w", err))
	}

	var raw map[string]map[string]float64
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, customerrors.NewAPIError("simple/price", 0, fmt.Errorf("failed to parse JSON: %w", err))
	}

	prices := make(map[string]float64, len(coinIDs))
	var missing []string
	for _, id := range coinIDs {
		if data, ok := raw[id]; ok {
			prices[id] = data["usd"]
		} else {
			missing = append(missing, id)
		}
	}

	// Fan-out: fetch any missing coins concurrently.
	if len(missing) > 0 {
		type result struct {
			coinID string
			price  float64
			err    error
		}

		ch := make(chan result, len(missing))
		var wg sync.WaitGroup

		for _, id := range missing {
			id := id
			wg.Add(1)
			go func() {
				defer wg.Done()
				p, err := cg.FetchPrice(id)
				ch <- result{coinID: id, price: p, err: err}
			}()
		}

		wg.Wait()
		close(ch)

		var mu sync.Mutex
		for r := range ch {
			if r.err != nil {
				// Non-fatal: record what we have and continue.
				continue
			}
			mu.Lock()
			prices[r.coinID] = r.price
			mu.Unlock()
		}
	}

	return prices, nil
}

func (cg *CoinGecko) FetchPrice(coinID string) (float64, error) {
	cg.waitForRateLimit()

	url := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=usd", cg.BaseURL, coinID)

	resp, err := cg.Client.Get(url)
	if err != nil {
		return 0, customerrors.NewAPIError("simple/price", 0, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return 0, customerrors.NewAPIError("simple/price", 429, customerrors.ErrRateLimitExceeded)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, customerrors.NewAPIError("simple/price", resp.StatusCode, fmt.Errorf("coin %s not found", coinID))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, customerrors.NewAPIError("simple/price", 0, fmt.Errorf("failed to read response: %w", err))
	}

	var result map[string]map[string]float64
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, customerrors.NewAPIError("simple/price", 0, fmt.Errorf("failed to parse JSON: %w", err))
	}

	price, ok := result[coinID]["usd"]
	if !ok {
		return 0, customerrors.NewAPIError("simple/price", 0, customerrors.ErrPriceNotAvailable)
	}

	return price, nil
}

func (cg *CoinGecko) GetSupportedCoins() (map[string]string, error) {
	url := fmt.Sprintf("%s/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=100&page=1", cg.BaseURL)

	resp, err := cg.Client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch supported coins: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var coinList []struct {
		ID     string `json:"id"`
		Symbol string `json:"symbol"`
		Name   string `json:"name"`
	}

	if err := json.Unmarshal(body, &coinList); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	coins := make(map[string]string, len(coinList))
	for _, coin := range coinList {
		coins[coin.ID] = coin.Name
	}

	return coins, nil
}
