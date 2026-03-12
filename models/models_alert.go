package models

import "time"

type AlertType string

const (
	AlertTypeBuy  AlertType = "buy"
	AlertTypeSell AlertType = "sell"
)

type Alert struct {
	ID             string    `bson:"_id,omitempty"  json:"id"`
	UserEmail      string    `bson:"user_email"     json:"user_email"`
	CoinID         string    `bson:"coin_id"        json:"coin_id"`
	CoinName       string    `bson:"coin_name"      json:"coin_name"`
	AlertType      AlertType `bson:"alert_type"     json:"alert_type"`
	ThresholdPrice float64   `bson:"threshold_price" json:"threshold_price"`
	Triggered      bool      `bson:"triggered"      json:"triggered"`
	CreatedAt      time.Time `bson:"created_at"     json:"created_at"`
	TriggeredAt    time.Time `bson:"triggered_at,omitempty" json:"triggered_at,omitempty"`
}
