package domain

import "errors"

var (
	// ErrNotFound indicates the requested appointment does not exist.
	ErrNotFound = errors.New("appointment not found")
	// ErrPersonNotFound indicates the referenced bookable person does not exist.
	ErrPersonNotFound = errors.New("person not found")
	// ErrCustomerNotFound indicates the referenced customer does not exist.
	ErrCustomerNotFound = errors.New("customer not found")
	// ErrServiceNotFound indicates the referenced service does not exist.
	ErrServiceNotFound = errors.New("service not found")
	// ErrTimeConflict indicates the requested time range overlaps an active booking for the person.
	ErrTimeConflict = errors.New("appointment time conflicts with an existing booking")
	// ErrInvalid indicates an appointment value or transition violates calendar rules.
	ErrInvalid = errors.New("invalid appointment data")
)

// InvalidFieldError identifies a field that violates a calendar domain constraint.
type InvalidFieldError struct {
	Field  string
	Reason string
}

// Error returns a safe field-specific validation message.
func (e InvalidFieldError) Error() string { return e.Field + ": " + e.Reason }

// Unwrap identifies the general calendar validation failure.
func (e InvalidFieldError) Unwrap() error { return ErrInvalid }
