package migrations

import "embed"

// FS contains the versioned SQL migrations owned by bounded contexts.
//
//go:embed *.sql
var FS embed.FS
