package auth

import (
	"bufio"
	"context"
	"crypto-portfolio-tracker/db"
	"crypto-portfolio-tracker/email"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Email    string `bson:"email"`
	Password string `bson:"password"`
	Verified bool   `bson:"verified"`
	OTP      string `bson:"otp"`
}

func generateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func hashPassword(password string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func checkPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

func Signup(userEmail, password string, reader *bufio.Reader) bool {
	database, err := db.ConnectDatabase()
	if err != nil {
		fmt.Println("Database connection failed:", err)
		return false
	}

	collection := database.Collection("users")

	var existingUser User
	err = collection.FindOne(
		context.TODO(),
		bson.M{"email": userEmail},
	).Decode(&existingUser)

	if err == nil {
		fmt.Println("Account already exists with this email.")
		return false
	}

	if err != mongo.ErrNoDocuments {
		fmt.Println("Database error:", err)
		return false
	}

	otp := generateOTP()
	email.SendOTP(userEmail, otp)

	fmt.Print("Enter OTP: ")
	inputOTP, err := reader.ReadString('\n')
	inputOTP = strings.TrimSpace(inputOTP)
	if err != nil {
		fmt.Println("Input error:", err)
		return false
	}
	if inputOTP != otp {
		fmt.Println("Invalid OTP. Signup failed.")
		return false
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		fmt.Println("Password hashing failed:", err)
		return false
	}

	user := User{
		Email:    userEmail,
		Password: hashedPassword,
		Verified: true,
		OTP:      "",
	}

	_, err = collection.InsertOne(context.TODO(), user)
	if err != nil {
		fmt.Println("Signup failed:", err)
		return false
	}

	fmt.Println("Signup successful!")
	return true
}

// SignupNoOTP creates a user account without sending or requiring an OTP.
// This is intended for API-driven clients (e.g., React frontend).
func SignupNoOTP(userEmail, password string) error {
	database, err := db.ConnectDatabase()
	if err != nil {
		return fmt.Errorf("database connection failed: %w", err)
	}

	collection := database.Collection("users")

	var existing User
	err = collection.FindOne(
		context.TODO(),
		bson.M{"email": userEmail},
	).Decode(&existing)

	if err == nil {
		return fmt.Errorf("account already exists")
	}
	if err != mongo.ErrNoDocuments {
		return fmt.Errorf("database error: %w", err)
	}

	hashedPassword, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("password hashing failed: %w", err)
	}

	user := User{
		Email:    userEmail,
		Password: hashedPassword,
		Verified: true,
		OTP:      "",
	}

	_, err = collection.InsertOne(context.TODO(), user)
	if err != nil {
		return fmt.Errorf("signup failed: %w", err)
	}

	return nil
}

func Login(email, password string) bool {
	database, err := db.ConnectDatabase()
	if err != nil {
		fmt.Println("Database connection failed:", err)
		return false
	}
	var u User

	database.Collection("users").FindOne(
		context.TODO(),
		map[string]string{"email": email},
	).Decode(&u)

	if !u.Verified {
		fmt.Println("Email not verified")
		return false
	}

	if checkPassword(u.Password, password) {
		fmt.Println("Login successful!")
		return true
	}
	return false
}
