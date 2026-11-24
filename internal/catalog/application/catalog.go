package application

import (
	"context"

	"go-service/internal/catalog/domain"
)

// Store is the catalog persistence contract consumed by the application.
type Store interface {
	List(context.Context) ([]domain.Category, error)
	GetCategory(context.Context, string) (domain.Category, error)
	GetService(context.Context, string) (domain.Service, error)
	CreateCategory(context.Context, string) (domain.Category, error)
	UpdateCategory(context.Context, string, string, int64) (domain.Category, error)
	DeleteCategory(context.Context, string, int64) error
	MoveCategory(context.Context, string, int, int64) (domain.Category, error)
	CreateService(context.Context, string, domain.Service, int) (domain.Service, error)
	UpdateService(context.Context, string, domain.Service, int64) (domain.Service, error)
	DeleteService(context.Context, string, int64) error
	MoveService(context.Context, string, string, int, int64) (domain.Service, error)
}

// Catalog coordinates catalog use cases.
type Catalog struct{ store Store }

// NewCatalog constructs catalog use cases with their persistence capability.
func NewCatalog(store Store) *Catalog { return &Catalog{store: store} }

// List returns categories and services in their persisted order.
func (c *Catalog) List(ctx context.Context) ([]domain.Category, error) {
	return c.store.List(ctx)
}

// GetCategory returns one category and its ordered services.
func (c *Catalog) GetCategory(ctx context.Context, id string) (domain.Category, error) {
	return c.store.GetCategory(ctx, id)
}

// GetService returns one service by immutable identity.
func (c *Catalog) GetService(ctx context.Context, id string) (domain.Service, error) {
	return c.store.GetService(ctx, id)
}

// CreateCategory appends a category to the global category order.
func (c *Catalog) CreateCategory(ctx context.Context, name string) (domain.Category, error) {
	if err := domain.ValidateName(name); err != nil {
		return domain.Category{}, err
	}
	return c.store.CreateCategory(ctx, name)
}

// UpdateCategory changes the category name if its version is current.
func (c *Catalog) UpdateCategory(ctx context.Context, id, name string, version int64) (domain.Category, error) {
	if err := domain.ValidateName(name); err != nil {
		return domain.Category{}, err
	}
	return c.store.UpdateCategory(ctx, id, name, version)
}

// DeleteCategory removes a category and its assigned services if its version is current.
func (c *Catalog) DeleteCategory(ctx context.Context, id string, version int64) error {
	return c.store.DeleteCategory(ctx, id, version)
}

// MoveCategory inserts a category at a position in the current global order.
func (c *Catalog) MoveCategory(ctx context.Context, id string, position int, version int64) (domain.Category, error) {
	return c.store.MoveCategory(ctx, id, position, version)
}
