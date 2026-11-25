package catalog

import (
	"context"
	"database/sql"
	"errors"

	sharedpostgres "go-service/internal/shared/postgres"
)

// ErrServiceNotFound indicates a referenced catalog service does not exist.
var ErrServiceNotFound = errors.New("service not found")

// ServiceSummary is the minimal service projection exposed by the catalog context.
type ServiceSummary struct {
	ID              string
	Name            string
	DurationMinutes int
}

// ServiceReader reads catalog service summaries for cross-context queries.
type ServiceReader struct {
	db *sql.DB
}

// NewServiceReader creates a reader for catalog service records.
func NewServiceReader(db *sharedpostgres.Client) *ServiceReader {
	if db == nil {
		return &ServiceReader{}
	}
	return &ServiceReader{db: db.DB}
}

// GetService fetches a single service summary by ID.
func (r *ServiceReader) GetService(ctx context.Context, id string) (ServiceSummary, error) {
	if r.db == nil {
		return ServiceSummary{}, ErrServiceNotFound
	}
	var s ServiceSummary
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, duration_minutes FROM catalog_services WHERE id = $1`, id,
	).Scan(&s.ID, &s.Name, &s.DurationMinutes)
	if errors.Is(err, sql.ErrNoRows) {
		return ServiceSummary{}, ErrServiceNotFound
	}
	return s, err
}

// GetServicesByIDs batch-fetches service summaries keyed by service ID.
func (r *ServiceReader) GetServicesByIDs(ctx context.Context, ids []string) (map[string]ServiceSummary, error) {
	result := make(map[string]ServiceSummary)
	if r.db == nil || len(ids) == 0 {
		return result, nil
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, duration_minutes FROM catalog_services WHERE id = ANY($1::uuid[])`, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s ServiceSummary
		if err := rows.Scan(&s.ID, &s.Name, &s.DurationMinutes); err != nil {
			return nil, err
		}
		result[s.ID] = s
	}
	return result, rows.Err()
}
