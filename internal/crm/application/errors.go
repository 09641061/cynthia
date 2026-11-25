package application

import (
	"errors"

	"go-service/internal/crm/domain"
)

// Application sentinel errors mapped to HTTP status codes.
var (
	ErrNotFound           = errors.New("crm resource not found")
	ErrConflict           = errors.New("crm state conflict")
	ErrPreconditionFailed = errors.New("crm precondition failed")
)

// DuplicatePhoneError indicates a customer with the same normalized phone already exists.
type DuplicatePhoneError struct {
	Existing domain.Customer
}

func (e DuplicatePhoneError) Error() string { return "customer phone already exists" }

func (e DuplicatePhoneError) Unwrap() error { return ErrConflict }
