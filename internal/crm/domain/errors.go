package domain

import "errors"

// ErrInvalid indicates invalid CRM domain input.
var ErrInvalid = errors.New("invalid crm input")

// InvalidFieldError identifies a domain validation failure on a specific field.
type InvalidFieldError struct {
	Field  string
	Reason string
}

func (e InvalidFieldError) Error() string { return e.Field + ": " + e.Reason }

func (e InvalidFieldError) Unwrap() error { return ErrInvalid }
