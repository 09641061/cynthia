package application

import "errors"

var (
	// ErrNotFound indicates a category or service does not exist.
	ErrNotFound = errors.New("catalog resource not found")
	// ErrConflict indicates a persistence constraint rejected a catalog change.
	ErrConflict = errors.New("catalog state conflict")
	// ErrPreconditionFailed indicates a mutation used a stale resource version.
	ErrPreconditionFailed = errors.New("catalog version precondition failed")
)
