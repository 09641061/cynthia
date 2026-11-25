package crm

import (
	"context"
	"database/sql"
	"errors"

	sharedpostgres "go-service/internal/shared/postgres"
)

// ErrCustomerNotFound indicates a referenced CRM customer does not exist.
var ErrCustomerNotFound = errors.New("customer not found")

// CustomerSummary is the minimal customer projection exposed by the CRM context.
type CustomerSummary struct {
	ID    string
	Name  string
	Phone string
}

// CustomerReader reads CRM customer summaries for cross-context queries.
type CustomerReader struct {
	db *sharedpostgres.Client
}

// NewCustomerReader creates a reader for CRM customer records.
func NewCustomerReader(db *sharedpostgres.Client) *CustomerReader {
	return &CustomerReader{db: db}
}

// GetCustomer fetches a single customer summary by ID.
func (r *CustomerReader) GetCustomer(ctx context.Context, id string) (CustomerSummary, error) {
	if r.db == nil {
		return CustomerSummary{}, ErrCustomerNotFound
	}
	var c CustomerSummary
	err := r.db.Do(ctx, func() error {
		return r.db.QueryRowContext(ctx,
			`SELECT id, name, phone FROM crm_customers WHERE id = $1`, id,
		).Scan(&c.ID, &c.Name, &c.Phone)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return CustomerSummary{}, ErrCustomerNotFound
	}
	return c, err
}

// GetCustomersByIDs batch-fetches customer summaries keyed by customer ID.
func (r *CustomerReader) GetCustomersByIDs(ctx context.Context, ids []string) (map[string]CustomerSummary, error) {
	result := make(map[string]CustomerSummary)
	if r.db == nil || len(ids) == 0 {
		return result, nil
	}
	err := r.db.Do(ctx, func() error {
		rows, err := r.db.QueryContext(ctx,
			`SELECT id, name, phone FROM crm_customers WHERE id = ANY($1::uuid[])`, ids,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c CustomerSummary
			if err := rows.Scan(&c.ID, &c.Name, &c.Phone); err != nil {
				return err
			}
			result[c.ID] = c
		}
		return rows.Err()
	})
	return result, err
}
