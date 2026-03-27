package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type Holding struct {
	CoinID   string    `bson:"coin_id"   json:"coin_id"`
	CoinName string    `bson:"coin_name" json:"coin_name"`
	Quantity float64   `bson:"quantity"  json:"quantity"`
	BuyPrice float64   `bson:"buy_price" json:"buy_price"`
	AddedAt  time.Time `bson:"added_at"  json:"added_at"`
}

type holdingJSON struct {
	CoinID   string  `json:"coin_id"`
	CoinName string  `json:"coin_name"`
	Quantity float64 `json:"quantity"`
	BuyPrice float64 `json:"buy_price"`
	AddedAt  string  `json:"added_at"`
}

func (h Holding) MarshalJSON() ([]byte, error) {
	return json.Marshal(holdingJSON{
		CoinID:   h.CoinID,
		CoinName: h.CoinName,
		Quantity: h.Quantity,
		BuyPrice: h.BuyPrice,
		AddedAt:  h.AddedAt.UTC().Format(time.RFC3339),
	})
}

func (h *Holding) UnmarshalJSON(data []byte) error {
	var raw holdingJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("holding: json unmarshal: %w", err)
	}

	t, err := time.Parse(time.RFC3339, raw.AddedAt)
	if err != nil {
		return fmt.Errorf("holding: parse added_at %q: %w", raw.AddedAt, err)
	}

	h.CoinID = raw.CoinID
	h.CoinName = raw.CoinName
	h.Quantity = raw.Quantity
	h.BuyPrice = raw.BuyPrice
	h.AddedAt = t.UTC()
	return nil
}

type Portfolio struct {
	UserEmail string    `bson:"user_email" json:"user_email"`
	Holdings  []Holding `bson:"holdings"   json:"holdings"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

type portfolioJSON struct {
	UserEmail string    `json:"user_email"`
	Holdings  []Holding `json:"holdings"`
	UpdatedAt string    `json:"updated_at"`
}

func (p Portfolio) MarshalJSON() ([]byte, error) {
	return json.Marshal(portfolioJSON{
		UserEmail: p.UserEmail,
		Holdings:  p.Holdings,
		UpdatedAt: p.UpdatedAt.UTC().Format(time.RFC3339),
	})
}

func (p *Portfolio) UnmarshalJSON(data []byte) error {
	var raw portfolioJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("portfolio: json unmarshal: %w", err)
	}

	t, err := time.Parse(time.RFC3339, raw.UpdatedAt)
	if err != nil {
		return fmt.Errorf("portfolio: parse updated_at %q: %w", raw.UpdatedAt, err)
	}

	p.UserEmail = raw.UserEmail
	p.Holdings = raw.Holdings
	p.UpdatedAt = t.UTC()
	return nil
}
