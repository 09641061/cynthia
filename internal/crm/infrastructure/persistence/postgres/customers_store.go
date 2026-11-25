package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

// Store implements CRM persistence against PostgreSQL.
type Store struct{ db *sql.DB }

// NewStore creates a PostgreSQL CRM store.
func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// ListCustomers returns paginated customers matching optional query.
func (s *Store) ListCustomers(ctx context.Context, query string, page, pageSize int) ([]domain.Customer, int, error) {
	where, args := buildCustomerFilter(query)
	var total int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM crm_customers `+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	limitIdx, offsetIdx := len(args)-1, len(args)
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, phone, version, created_at, updated_at FROM crm_customers `+where+
			` ORDER BY created_at DESC, id DESC LIMIT $`+strconv.Itoa(limitIdx)+` OFFSET $`+strconv.Itoa(offsetIdx), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]domain.Customer, 0)
	for rows.Next() {
		var c domain.Customer
		if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.Version, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}

// GetCustomer fetches a customer by ID.
func (s *Store) GetCustomer(ctx context.Context, id string) (domain.Customer, error) {
	var c domain.Customer
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, phone, version, created_at, updated_at FROM crm_customers WHERE id = $1`, id,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Customer{}, application.ErrNotFound
	}
	return c, err
}

// GetCustomerByPhone fetches a customer by normalized phone.
func (s *Store) GetCustomerByPhone(ctx context.Context, phone string) (domain.Customer, error) {
	var c domain.Customer
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, phone, version, created_at, updated_at FROM crm_customers WHERE phone = $1`, phone,
	).Scan(&c.ID, &c.Name, &c.Phone, &c.Version, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Customer{}, application.ErrNotFound
	}
	return c, err
}

// CreateCustomer inserts a new customer, mapping phone duplicates to DuplicatePhoneError.
func (s *Store) CreateCustomer(ctx context.Context, customer domain.Customer) (domain.Customer, error) {
	var created domain.Customer
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO crm_customers (id, name, phone, version) VALUES ($1, $2, $3, 1)
		 RETURNING id, name, phone, version, created_at, updated_at`,
		customer.ID, customer.Name, customer.Phone,
	).Scan(&created.ID, &created.Name, &created.Phone, &created.Version, &created.CreatedAt, &created.UpdatedAt)
	if isPGCode(err, pgErrUniqueViolation) {
		existing, lookupErr := s.GetCustomerByPhone(ctx, customer.Phone)
		if lookupErr != nil {
			if errors.Is(lookupErr, application.ErrNotFound) {
				return domain.Customer{}, application.ErrConflict
			}
			return domain.Customer{}, lookupErr
		}
		return domain.Customer{}, application.DuplicatePhoneError{Existing: existing}
	}
	return created, err
}

// UpdateCustomer updates a customer with optimistic locking on version.
func (s *Store) UpdateCustomer(ctx context.Context, id string, expectedVersion int64, name, phone string) (domain.Customer, error) {
	var updated domain.Customer
	err := s.db.QueryRowContext(ctx,
		`UPDATE crm_customers SET name = $1, phone = $2, version = version + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3 AND version = $4
		 RETURNING id, name, phone, version, created_at, updated_at`,
		name, phone, id, expectedVersion,
	).Scan(&updated.ID, &updated.Name, &updated.Phone, &updated.Version, &updated.CreatedAt, &updated.UpdatedAt)
	if isPGCode(err, pgErrUniqueViolation) {
		existing, lookupErr := s.GetCustomerByPhone(ctx, phone)
		if lookupErr != nil {
			if errors.Is(lookupErr, application.ErrNotFound) {
				return domain.Customer{}, application.ErrConflict
			}
			return domain.Customer{}, lookupErr
		}
		return domain.Customer{}, application.DuplicatePhoneError{Existing: existing}
	}
	if errors.Is(err, sql.ErrNoRows) {
		if _, getErr := s.GetCustomer(ctx, id); getErr != nil {
			return domain.Customer{}, getErr
		}
		return domain.Customer{}, application.ErrPreconditionFailed
	}
	return updated, err
}

// DeleteCustomer deletes a customer by ID, mapping FK violations to ErrConflict.
func (s *Store) DeleteCustomer(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM crm_customers WHERE id = $1`, id)
	if isPGCode(err, pgErrForeignKeyViolation) {
		return application.ErrConflict
	}
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return application.ErrNotFound
	}
	return nil
}
