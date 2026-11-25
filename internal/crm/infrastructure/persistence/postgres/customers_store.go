package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
	sharedpostgres "go-service/internal/shared/postgres"
)

// Store implements CRM persistence against PostgreSQL.
type Store struct{ db *sharedpostgres.Client }

// NewStore creates a PostgreSQL CRM store.
func NewStore(db *sharedpostgres.Client) *Store { return &Store{db: db} }

// ListCustomers returns paginated customers matching optional query.
func (s *Store) ListCustomers(ctx context.Context, query string, page, pageSize int) ([]domain.Customer, int, error) {
	where, args := buildCustomerFilter(query)
	var total int
	var items []domain.Customer
	err := s.db.Do(ctx, func() error {
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM crm_customers `+where, args...).Scan(&total); err != nil {
			return err
		}
		offset := (page - 1) * pageSize
		qArgs := append(append([]any(nil), args...), pageSize, offset)
		limitIdx, offsetIdx := len(qArgs)-1, len(qArgs)
		rows, err := s.db.QueryContext(ctx,
			`SELECT id, name, phone, created_at, updated_at FROM crm_customers `+where+
				` ORDER BY created_at DESC, id DESC LIMIT $`+strconv.Itoa(limitIdx)+` OFFSET $`+strconv.Itoa(offsetIdx), qArgs...)
		if err != nil {
			return err
		}
		defer rows.Close()
		items = make([]domain.Customer, 0)
		for rows.Next() {
			var c domain.Customer
			if err := rows.Scan(&c.ID, &c.Name, &c.Phone, &c.CreatedAt, &c.UpdatedAt); err != nil {
				return err
			}
			items = append(items, c)
		}
		return rows.Err()
	})
	return items, total, err
}

// GetCustomer fetches a customer by ID.
func (s *Store) GetCustomer(ctx context.Context, id string) (domain.Customer, error) {
	var c domain.Customer
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx,
			`SELECT id, name, phone, created_at, updated_at FROM crm_customers WHERE id = $1`, id,
		).Scan(&c.ID, &c.Name, &c.Phone, &c.CreatedAt, &c.UpdatedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Customer{}, application.ErrNotFound
	}
	return c, err
}

// GetCustomerByPhone fetches a customer by normalized phone.
func (s *Store) GetCustomerByPhone(ctx context.Context, phone string) (domain.Customer, error) {
	var c domain.Customer
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx,
			`SELECT id, name, phone, created_at, updated_at FROM crm_customers WHERE phone = $1`, phone,
		).Scan(&c.ID, &c.Name, &c.Phone, &c.CreatedAt, &c.UpdatedAt)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Customer{}, application.ErrNotFound
	}
	return c, err
}

// CreateCustomer inserts a new customer, mapping phone duplicates to DuplicatePhoneError.
func (s *Store) CreateCustomer(ctx context.Context, customer domain.Customer) (domain.Customer, error) {
	var created domain.Customer
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx,
			`INSERT INTO crm_customers (id, name, phone) VALUES ($1, $2, $3)
			 RETURNING id, name, phone, created_at, updated_at`,
			customer.ID, customer.Name, customer.Phone,
		).Scan(&created.ID, &created.Name, &created.Phone, &created.CreatedAt, &created.UpdatedAt)
	})
	if isPGCode(err, pgErrUniqueViolation) {
		return domain.Customer{}, s.duplicatePhoneError(ctx, customer.Phone)
	}
	return created, err
}

// UpdateCustomer updates a customer.
func (s *Store) UpdateCustomer(ctx context.Context, id, name, phone string) (domain.Customer, error) {
	var updated domain.Customer
	err := s.db.Do(ctx, func() error {
		return s.db.QueryRowContext(ctx,
			`UPDATE crm_customers SET name = $1, phone = $2, updated_at = CURRENT_TIMESTAMP
			 WHERE id = $3 RETURNING id, name, phone, created_at, updated_at`,
			name, phone, id,
		).Scan(&updated.ID, &updated.Name, &updated.Phone, &updated.CreatedAt, &updated.UpdatedAt)
	})
	if isPGCode(err, pgErrUniqueViolation) {
		return domain.Customer{}, s.duplicatePhoneError(ctx, phone)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Customer{}, application.ErrNotFound
	}
	return updated, err
}

// DeleteCustomer deletes a customer by ID, mapping FK violations to ErrConflict.
func (s *Store) DeleteCustomer(ctx context.Context, id string) error {
	var affected int64
	err := s.db.Do(ctx, func() error {
		res, err := s.db.ExecContext(ctx, `DELETE FROM crm_customers WHERE id = $1`, id)
		if err != nil {
			return err
		}
		affected, err = res.RowsAffected()
		return err
	})
	if isPGCode(err, pgErrForeignKeyViolation) {
		return application.ErrConflict
	}
	if err != nil {
		return err
	}
	if affected == 0 {
		return application.ErrNotFound
	}
	return nil
}

func (s *Store) duplicatePhoneError(ctx context.Context, phone string) error {
	existing, err := s.GetCustomerByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return application.ErrConflict
		}
		return err
	}
	return application.DuplicatePhoneError{Existing: existing}
}
