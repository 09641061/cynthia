package domain

import (
	"strings"
	"time"
)

// Customer represents a CRM customer profile.
type Customer struct {
	ID        string
	Name      string
	Phone     string
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ValidateName trims and validates a customer name.
func ValidateName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", InvalidFieldError{Field: "name", Reason: "must not be empty"}
	}
	return name, nil
}

// NewCustomer constructs a validated Customer with version 1.
func NewCustomer(id, name, phone string) (Customer, error) {
	validName, err := ValidateName(name)
	if err != nil {
		return Customer{}, err
	}
	validPhone, err := NormalizePhone(phone)
	if err != nil {
		return Customer{}, err
	}
	return Customer{
		ID:      id,
		Name:    validName,
		Phone:   validPhone,
		Version: 1,
	}, nil
}
