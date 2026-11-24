package domain

import (
	"errors"
	"strings"
	"time"
)

// ErrInvalid indicates a catalog value or transition violates catalog rules.
var ErrInvalid = errors.New("invalid catalog data")

// InvalidFieldError identifies a catalog field that violates a business constraint.
type InvalidFieldError struct {
	Field  string
	Reason string
}

// Error returns a safe field-specific validation message.
func (e InvalidFieldError) Error() string { return e.Field + ": " + e.Reason }

// Unwrap identifies the general catalog validation failure.
func (e InvalidFieldError) Unwrap() error { return ErrInvalid }

// Category is an ordered group of services in the catalog.
type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sort_order"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Services  []Service `json:"services"`
}

// Service is a bookable offering assigned to one category.
type Service struct {
	ID             string    `json:"id"`
	CategoryID     string    `json:"category_id"`
	Name           string    `json:"name"`
	DurationMinute int       `json:"duration_minutes"`
	PriceMinor     int64     `json:"price_minor_units"`
	CurrencyCode   string    `json:"currency_code"`
	SortOrder      int       `json:"sort_order"`
	Version        int64     `json:"version"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ValidateName checks the shared catalog name constraints.
func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return InvalidFieldError{Field: "name", Reason: "must not be blank"}
	}
	if len([]rune(name)) > 120 {
		return InvalidFieldError{Field: "name", Reason: "must not exceed 120 characters"}
	}
	return nil
}

// ValidateService checks the catalog service fields and supported ranges.
func ValidateService(name string, duration int, price int64, currency string) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if duration < 1 || duration > 1440 {
		return InvalidFieldError{Field: "duration_minutes", Reason: "must be between 1 and 1440"}
	}
	if price < 0 {
		return InvalidFieldError{Field: "price_minor_units", Reason: "must not be negative"}
	}
	if len(currency) != 3 {
		return InvalidFieldError{Field: "currency_code", Reason: "must be three uppercase letters"}
	}
	for _, char := range currency {
		if char < 'A' || char > 'Z' {
			return InvalidFieldError{Field: "currency_code", Reason: "must be three uppercase letters"}
		}
	}
	return nil
}
