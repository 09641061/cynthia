package application

import "errors"

// ErrNotFound indicates the organization record does not exist.
var ErrNotFound = errors.New("organization not found")
