package postgres

import (
	"context"

	"go-service/internal/catalog/domain"
)

func (s *Store) List(ctx context.Context) (categories []domain.Category, err error) {
	err = s.db.Do(ctx, func() error {
		categories, err = s.list(ctx)
		return err
	})
	return categories, err
}

func (s *Store) GetCategory(ctx context.Context, id string) (category domain.Category, err error) {
	err = s.db.Do(ctx, func() error {
		category, err = s.getCategory(ctx, id)
		return err
	})
	return category, err
}

func (s *Store) GetService(ctx context.Context, id string) (service domain.Service, err error) {
	err = s.db.Do(ctx, func() error {
		service, err = s.getService(ctx, id)
		return err
	})
	return service, err
}
