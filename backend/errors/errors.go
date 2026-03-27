package errors

import (
	"errors"
	"fmt"
)

var (
	ErrEmptyHoldings      = errors.New("no holdings provided")
	ErrEmptyPortfolio     = errors.New("portfolio is empty")
	ErrCoinNotFound       = errors.New("coin not found in portfolio")
	ErrPriceNotAvailable  = errors.New("price not available")
	ErrInvalidQuantity    = errors.New("invalid quantity")
	ErrInvalidPrice       = errors.New("invalid price")
	ErrDatabaseConnection = errors.New("database connection failed")
	ErrRateLimitExceeded  = errors.New("API rate limit exceeded")
	ErrAuthFailed         = errors.New("authentication failed")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidOTP         = errors.New("invalid OTP")
)

type PortfolioError struct {
	Operation string
	CoinID    string
	Err       error
}

func (e *PortfolioError) Error() string {
	if e.CoinID != "" {
		return fmt.Sprintf("portfolio %s failed for coin %s: %v", e.Operation, e.CoinID, e.Err)
	}
	return fmt.Sprintf("portfolio %s failed: %v", e.Operation, e.Err)
}

func (e *PortfolioError) Unwrap() error {
	return e.Err
}

func NewPortfolioError(operation, coinID string, err error) error {
	return &PortfolioError{
		Operation: operation,
		CoinID:    coinID,
		Err:       err,
	}
}

type APIError struct {
	Endpoint   string
	StatusCode int
	Err        error
}

func (e *APIError) Error() string {
	if e.StatusCode != 0 {
		return fmt.Sprintf("API request to %s failed with status %d: %v", e.Endpoint, e.StatusCode, e.Err)
	}
	return fmt.Sprintf("API request to %s failed: %v", e.Endpoint, e.Err)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

func NewAPIError(endpoint string, statusCode int, err error) error {
	return &APIError{
		Endpoint:   endpoint,
		StatusCode: statusCode,
		Err:        err,
	}
}

type DatabaseError struct {
	Operation  string
	Collection string
	Err        error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database %s on collection '%s' failed: %v", e.Operation, e.Collection, e.Err)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

func NewDatabaseError(operation, collection string, err error) error {
	return &DatabaseError{
		Operation:  operation,
		Collection: collection,
		Err:        err,
	}
}

type ValidationError struct {
	Field string
	Value interface{}
	Err   error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s' with value '%v': %v", e.Field, e.Value, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

func NewValidationError(field string, value interface{}, err error) error {
	return &ValidationError{
		Field: field,
		Value: value,
		Err:   err,
	}
}
