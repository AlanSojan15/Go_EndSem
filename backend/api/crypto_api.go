package api

type CryptoApi interface {
	FetchPrice(coinID string) (float64, error)
	FetchMultiplePrices(coinIDs ...string) (map[string]float64, error)
	GetSupportedCoins() (map[string]string, error)
}
