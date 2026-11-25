package application

import (
	"errors"

	"go-service/internal/crm/domain"
)

// Application sentinel errors mapped to HTTP status codes.
var (
	ErrNotFound   = errors.New("crm resource not found")
	ErrConflict   = errors.New("crm state conflict")
	ErrBadGateway = errors.New("upstream whatsapp delivery failed")
)

// DuplicatePhoneError indicates a customer with the same normalized phone already exists.
type DuplicatePhoneError struct {
	Existing domain.Customer
}

func (e DuplicatePhoneError) Error() string { return "customer phone already exists" }

func (e DuplicatePhoneError) Unwrap() error { return ErrConflict }

// DeliveryError wraps an upstream WhatsApp delivery failure alongside the persisted message.
type DeliveryError struct {
	Message domain.Message
	Reason  string
}

func (e DeliveryError) Error() string { return e.Reason }

func (e DeliveryError) Unwrap() error { return ErrBadGateway }
