package rest

import (
	"context"

	"github.com/google/uuid"
	"go-service/internal/catalog/domain"
)

func newTestUUID() string { return uuid.NewString() }

type fakeStore struct {
	categories       []domain.Category
	created          domain.Category
	listErr          error
	updateErr        error
	createdService   domain.Service
	movedService     domain.Service
	createServiceErr error
}

func (f *fakeStore) List(context.Context) ([]domain.Category, error) { return f.categories, f.listErr }
func (f *fakeStore) GetCategory(context.Context, string) (domain.Category, error) {
	return domain.Category{Version: 1}, nil
}
func (f *fakeStore) GetService(context.Context, string) (domain.Service, error) {
	return domain.Service{Version: 1}, nil
}
func (f *fakeStore) CreateCategory(_ context.Context, name string) (domain.Category, error) {
	f.created = domain.Category{ID: newTestUUID(), Name: name, SortOrder: 1, Version: 1}
	return f.created, nil
}
func (f *fakeStore) UpdateCategory(_ context.Context, _ string, name string, _ int64) (domain.Category, error) {
	return domain.Category{Name: name, Version: 2}, f.updateErr
}
func (f *fakeStore) DeleteCategory(context.Context, string, int64) error { return nil }
func (f *fakeStore) MoveCategory(_ context.Context, id string, position int, _ int64) (domain.Category, error) {
	return domain.Category{ID: id, SortOrder: position, Version: 2}, nil
}
func (f *fakeStore) CreateService(_ context.Context, categoryID string, service domain.Service, position int) (domain.Service, error) {
	service.ID, service.CategoryID, service.SortOrder, service.Version = newTestUUID(), categoryID, position, 1
	f.createdService = service
	return service, f.createServiceErr
}
func (f *fakeStore) UpdateService(_ context.Context, _ string, service domain.Service, _ int64) (domain.Service, error) {
	service.Version = 2
	return service, nil
}
func (f *fakeStore) DeleteService(context.Context, string, int64) error { return nil }
func (f *fakeStore) MoveService(_ context.Context, id, categoryID string, position int, _ int64) (domain.Service, error) {
	f.movedService = domain.Service{ID: id, CategoryID: categoryID, SortOrder: position, Version: 2}
	return f.movedService, nil
}
