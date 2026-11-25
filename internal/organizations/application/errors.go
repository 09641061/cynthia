package application

import "errors"

var (
	// ErrNotFound indicates the organization record does not exist.
	ErrNotFound = errors.New("organization not found")
	// ErrPreconditionFailed indicates an update used a stale resource version.
	ErrPreconditionFailed = errors.New("organization version precondition failed")
)
