package postgres

import (
	"errors"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
)

const (
	pgErrExclusionViolation  = "23P01"
	pgErrForeignKeyViolation = "23503"
)

func mapPGError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}
	if pgErr.Code == pgErrExclusionViolation {
		return domain.ErrTimeConflict
	}
	if pgErr.Code == pgErrForeignKeyViolation {
		switch {
		case strings.Contains(pgErr.ConstraintName, "person"):
			return domain.ErrPersonNotFound
		case strings.Contains(pgErr.ConstraintName, "customer"):
			return domain.ErrCustomerNotFound
		case strings.Contains(pgErr.ConstraintName, "service"):
			return domain.ErrServiceNotFound
		}
		return domain.ErrInvalid
	}
	return err
}

func buildAppointmentFilter(filter application.AppointmentFilter) (string, []any) {
	clauses := make([]string, 0, 5)
	args := make([]any, 0, 5)
	if filter.From != nil {
		args = append(args, filter.From.UTC())
		clauses = append(clauses, "ends_at > $"+strconv.Itoa(len(args)))
	}
	if filter.To != nil {
		args = append(args, filter.To.UTC())
		clauses = append(clauses, "starts_at < $"+strconv.Itoa(len(args)))
	}
	if filter.PersonID != "" {
		args = append(args, filter.PersonID)
		clauses = append(clauses, "person_id = $"+strconv.Itoa(len(args)))
	}
	if filter.CustomerID != "" {
		args = append(args, filter.CustomerID)
		clauses = append(clauses, "customer_id = $"+strconv.Itoa(len(args)))
	}
	if filter.Status != "" {
		args = append(args, filter.Status)
		clauses = append(clauses, "status = $"+strconv.Itoa(len(args)))
	}
	if len(clauses) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}
