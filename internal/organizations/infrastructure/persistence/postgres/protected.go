package postgres

import (
	"context"

	"go-service/internal/organizations/domain"
)

// Get returns the singleton organization record.
func (s *Store) Get(ctx context.Context) (org domain.Organization, err error) {
	err = s.db.Do(ctx, func() error {
		org, err = s.get(ctx)
		return err
	})
	return org, err
}

// Update applies a partial update to the singleton organization using optimistic concurrency.
func (s *Store) Update(ctx context.Context, version int64, patch domain.UpdatePatch) (org domain.Organization, err error) {
	err = s.db.Do(ctx, func() error {
		org, err = s.update(ctx, version, patch)
		return err
	})
	return org, err
}
