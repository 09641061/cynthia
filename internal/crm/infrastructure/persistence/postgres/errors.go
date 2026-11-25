package postgres

import (
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/crm/application"
)

const (
	pgErrUniqueViolation     = "23505"
	pgErrForeignKeyViolation = "23503"
	maxSearchQueryRunes      = 200
	maxSearchTokens          = 8
)

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(s string) string { return likeEscaper.Replace(s) }

func truncateRunes(s string, maxRunes int) string {
	n := 0
	for i := range s {
		if n == maxRunes {
			return s[:i]
		}
		n++
	}
	return s
}

func isPGCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}

func buildCustomerFilter(query string) (string, []any) {
	tokens := strings.Fields(truncateRunes(query, maxSearchQueryRunes))
	if len(tokens) == 0 {
		return "", nil
	}
	if len(tokens) > maxSearchTokens {
		tokens = tokens[:maxSearchTokens]
	}
	clauses := make([]string, 0, len(tokens))
	args := make([]any, 0, len(tokens)*2)
	for _, token := range tokens {
		if token == "+" {
			args = append(args, "+")
			qIdx := strconv.Itoa(len(args))
			clauses = append(clauses, `(name ILIKE '%' || $`+qIdx+` || '%' ESCAPE '\' OR phone LIKE '+%')`)
			continue
		}
		args = append(args, escapeLike(token))
		qIdx := strconv.Itoa(len(args))
		if digits := extractDigits(token); digits != "" {
			args = append(args, digits)
			dIdx := strconv.Itoa(len(args))
			clauses = append(clauses, `(name ILIKE '%' || $`+qIdx+` || '%' ESCAPE '\' OR regexp_replace(phone, '[^0-9]', '', 'g') LIKE '%' || $`+dIdx+` || '%' ESCAPE '\')`)
			continue
		}
		clauses = append(clauses, `name ILIKE '%' || $`+qIdx+` || '%' ESCAPE '\'`)
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
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
		if '0' <= r && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
