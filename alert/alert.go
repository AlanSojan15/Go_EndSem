package alert

import (
	"context"
	"crypto-portfolio-tracker/api"
	"crypto-portfolio-tracker/db"
	emailpkg "crypto-portfolio-tracker/email"
	customerrors "crypto-portfolio-tracker/errors"
	"crypto-portfolio-tracker/models"
	"crypto-portfolio-tracker/portfolio"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ValidateCoinExists(coinID string, userEmail string, apiClient api.CryptoApi) error {
	coinID = strings.ToLower(strings.TrimSpace(coinID))

	p, err := portfolio.GetPortfolio(userEmail)
	if err != nil {
		return fmt.Errorf("could not load portfolio: %w", err)
	}

	for _, h := range p.Holdings {
		if h.CoinID == coinID {
			return nil
		}
	}

	return customerrors.NewPortfolioError(
		"validate coin",
		coinID,
		fmt.Errorf("coin %q is not in your portfolio — only portfolio coins can have alerts", coinID),
	)
}

func CreateAlert(userEmail, coinID, coinName string, alertType models.AlertType, threshold float64, apiClient api.CryptoApi) error {
	coinID = strings.ToLower(strings.TrimSpace(coinID))

	if threshold <= 0 {
		return customerrors.NewValidationError("threshold_price", threshold, customerrors.ErrInvalidPrice)
	}

	if err := ValidateCoinExists(coinID, userEmail, apiClient); err != nil {
		return err
	}

	database, err := db.ConnectDatabase()
	if err != nil {
		return customerrors.NewDatabaseError("connect", "alerts", err)
	}

	alert := models.Alert{
		ID:             primitive.NewObjectID().Hex(),
		UserEmail:      userEmail,
		CoinID:         coinID,
		CoinName:       coinName,
		AlertType:      alertType,
		ThresholdPrice: threshold,
		Triggered:      false,
		CreatedAt:      time.Now(),
	}

	_, err = database.Collection("alerts").InsertOne(context.TODO(), alert)
	if err != nil {
		return customerrors.NewDatabaseError("insert", "alerts", err)
	}

	return nil
}

func GetAlerts(userEmail string) ([]models.Alert, error) {
	database, err := db.ConnectDatabase()
	if err != nil {
		return nil, customerrors.NewDatabaseError("connect", "alerts", err)
	}

	cursor, err := database.Collection("alerts").Find(
		context.TODO(),
		bson.M{"user_email": userEmail, "triggered": false},
	)
	if err != nil {
		return nil, customerrors.NewDatabaseError("find", "alerts", err)
	}
	defer cursor.Close(context.TODO())

	var alerts []models.Alert
	if err := cursor.All(context.TODO(), &alerts); err != nil {
		return nil, customerrors.NewDatabaseError("decode", "alerts", err)
	}

	return alerts, nil
}

func CheckAndTriggerAlerts(userEmail string, apiClient api.CryptoApi) error {
	alerts, err := GetAlerts(userEmail)
	if err != nil {
		return err
	}

	if len(alerts) == 0 {
		fmt.Println("No active alerts to check.")
		return nil
	}

	seen := make(map[string]bool)
	var coinIDs []string
	for _, a := range alerts {
		if !seen[a.CoinID] {
			coinIDs = append(coinIDs, a.CoinID)
			seen[a.CoinID] = true
		}
	}

	prices, err := apiClient.FetchMultiplePrices(coinIDs...)
	if err != nil {
		return fmt.Errorf("could not fetch prices for alert check: %w", err)
	}

	database, err := db.ConnectDatabase()
	if err != nil {
		return customerrors.NewDatabaseError("connect", "alerts", err)
	}

	triggeredCount := 0
	for _, a := range alerts {
		currentPrice, ok := prices[a.CoinID]
		if !ok {
			fmt.Printf("  Warning: price not available for %s, skipping.\n", a.CoinID)
			continue
		}

		triggered := false
		switch a.AlertType {
		case models.AlertTypeBuy:
			triggered = currentPrice <= a.ThresholdPrice
		case models.AlertTypeSell:
			triggered = currentPrice >= a.ThresholdPrice
		}

		if triggered {
			now := time.Now()
			_, err := database.Collection("alerts").UpdateOne(
				context.TODO(),
				bson.M{"_id": a.ID},
				bson.M{"$set": bson.M{
					"triggered":    true,
					"triggered_at": now,
				}},
			)
			if err != nil {
				fmt.Printf("  Warning: could not mark alert as triggered for %s: %v\n", a.CoinID, err)
				continue
			}

			sendAlertEmail(userEmail, a, currentPrice)
			triggeredCount++
		}
	}

	if triggeredCount == 0 {
		fmt.Printf("  Checked %d alert(s) — none triggered yet.\n", len(alerts))
	} else {
		fmt.Printf("  %d alert(s) triggered and email(s) sent.\n", triggeredCount)
	}

	return nil
}

func sendAlertEmail(userEmail string, a models.Alert, currentPrice float64) {
	var action, direction string
	if a.AlertType == models.AlertTypeBuy {
		action = "BUY"
		direction = "dropped to"
	} else {
		action = "SELL"
		direction = "risen to"
	}

	subject := fmt.Sprintf("Crypto Alert: %s %s threshold reached!", a.CoinName, action)
	body := fmt.Sprintf(
		"Hello,\n\n"+
			"Your %s alert for %s (%s) has been triggered.\n\n"+
			"  Threshold Price : $%.2f\n"+
			"  Current Price   : $%.2f\n"+
			"  Status          : Price has %s your threshold.\n\n"+
			"Consider this a signal to %s.\n\n"+
			"— Crypto Portfolio Tracker",
		action, a.CoinName, a.CoinID,
		a.ThresholdPrice,
		currentPrice,
		direction,
		strings.ToLower(action),
	)

	emailpkg.SendAlert(userEmail, subject, body)
}

func DisplayAlerts(userEmail string) error {
	alerts, err := GetAlerts(userEmail)
	if err != nil {
		return err
	}

	if len(alerts) == 0 {
		fmt.Println("You have no active alerts.")
		return nil
	}

	fmt.Println("\n========== YOUR ALERTS ==========")
	for i, a := range alerts {
		fmt.Printf("\n[%d] %s (%s)\n", i+1, a.CoinName, a.CoinID)
		fmt.Printf("    Type      : %s\n", strings.ToUpper(string(a.AlertType)))
		fmt.Printf("    Threshold : $%.2f\n", a.ThresholdPrice)
		fmt.Printf("    Created   : %s\n", a.CreatedAt.Format("02 Jan 2006, 15:04 UTC"))
	}
	fmt.Println("\n=================================")

	return nil
}

func DeleteAlert(userEmail, alertID string) error {
	database, err := db.ConnectDatabase()
	if err != nil {
		return customerrors.NewDatabaseError("connect", "alerts", err)
	}

	result, err := database.Collection("alerts").DeleteOne(
		context.TODO(),
		bson.M{"_id": alertID, "user_email": userEmail},
	)
	if err != nil {
		return customerrors.NewDatabaseError("delete", "alerts", err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("alert not found or does not belong to your account")
	}

	return nil
}
