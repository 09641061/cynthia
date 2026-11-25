package postgres

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/crm/application"
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

func buildConversationFilter(filter application.ConversationFilter) (string, []any) {
	clauses := make([]string, 0, 3)
	args := make([]any, 0, 4)
	if filter.Status != "" {
		args = append(args, filter.Status)
		clauses = append(clauses, "c.status = $"+strconv.Itoa(len(args)))
	}
	if filter.CustomerID != "" {
		args = append(args, filter.CustomerID)
		clauses = append(clauses, "c.customer_id = $"+strconv.Itoa(len(args)))
	}
	if filter.Query != "" {
		args = append(args, filter.Query)
		qIdx := strconv.Itoa(len(args))
		digits := extractDigits(filter.Query)
		if digits != "" {
			args = append(args, digits)
			dIdx := strconv.Itoa(len(args))
			clauses = append(clauses, "(u.name ILIKE '%' || $"+qIdx+" || '%' OR regexp_replace(u.phone, '[^0-9]', '', 'g') LIKE '%' || $"+dIdx+" || '%')")
		} else {
			clauses = append(clauses, "u.name ILIKE '%' || $"+qIdx+" || '%'")
		}
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
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
