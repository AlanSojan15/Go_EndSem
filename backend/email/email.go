package email

import (
	"fmt"
	"net/smtp"
	"os"

	"github.com/joho/godotenv"
)

func loadCredentials() (from, password string, err error) {
	if err = godotenv.Load(); err != nil {
		return "", "", fmt.Errorf("error loading .env file: %w", err)
	}
	from = os.Getenv("EMAIL")
	password = os.Getenv("PASSWORD")
	if from == "" || password == "" {
		return "", "", fmt.Errorf("EMAIL or PASSWORD not set in .env file")
	}
	return from, password, nil
}

func sendMail(to, subject, body string) error {
	from, password, err := loadCredentials()
	if err != nil {
		return err
	}

	msg := "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" +
		body

	auth := smtp.PlainAuth("", from, password, "smtp.gmail.com")
	return smtp.SendMail("smtp.gmail.com:587", auth, from, []string{to}, []byte(msg))
}

func SendOTP(toEmail, otp string) {
	subject := "Crypto Tracker OTP Verification"
	body := "Your OTP is: " + otp

	if err := sendMail(toEmail, subject, body); err != nil {
		fmt.Println("Error sending OTP email:", err)
		return
	}
	fmt.Println("OTP sent to", toEmail)
}

func SendAlert(toEmail, subject, body string) {
	if err := sendMail(toEmail, subject, body); err != nil {
		fmt.Println("Error sending alert email:", err)
		return
	}
	fmt.Println("Alert email sent to", toEmail)
}
