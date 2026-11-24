package postgres

import (
	"errors"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pgErrUniqueViolation     = "23505"
	pgErrForeignKeyViolation = "23503"
)

func isPGCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

func buildCustomerFilter(query string) (string, []any) {
	if query == "" {
		return "", nil
	}
	digits := extractDigits(query)
	if digits != "" {
		return `WHERE name ILIKE '%' || $1 || '%' OR regexp_replace(phone, '[^0-9]', '', 'g') LIKE '%' || $2 || '%'`, []any{query, digits}
	}
	return `WHERE name ILIKE '%' || $1 || '%'`, []any{query}
}

func extractDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
