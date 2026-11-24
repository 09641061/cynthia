package application

import (
	"context"

	"go-service/internal/catalog/domain"
)

// CreateService adds a service to the selected category.
func (c *Catalog) CreateService(ctx context.Context, categoryID string, service domain.Service, position int) (domain.Service, error) {
	if err := domain.ValidateService(service.Name, service.DurationMinute, service.PriceMinor, service.CurrencyCode); err != nil {
		return domain.Service{}, err
	}
	return c.store.CreateService(ctx, categoryID, service, position)
}

// UpdateService changes service details if its version is current.
func (c *Catalog) UpdateService(ctx context.Context, id string, service domain.Service, version int64) (domain.Service, error) {
	if err := domain.ValidateService(service.Name, service.DurationMinute, service.PriceMinor, service.CurrencyCode); err != nil {
		return domain.Service{}, err
	}
	return c.store.UpdateService(ctx, id, service, version)
}

// DeleteService removes a service and compacts its category if its version is current.
func (c *Catalog) DeleteService(ctx context.Context, id string, version int64) error {
	return c.store.DeleteService(ctx, id, version)
}

// MoveService reorders a service using its current category state.
func (c *Catalog) MoveService(ctx context.Context, id, categoryID string, position int, version int64) (domain.Service, error) {
	return c.store.MoveService(ctx, id, categoryID, position, version)
}
