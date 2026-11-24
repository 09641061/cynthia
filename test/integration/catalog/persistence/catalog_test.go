//go:build integration

package persistence_test

import (
	"errors"
	"sync"
	"testing"

	"go-service/internal/catalog/domain"
)

func TestMovePreservesIdentityAndRenumbersBothLists(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	source, err := store.CreateCategory(ctx, "Hair")
	if err != nil {
		t.Fatalf("create source category: %v", err)
	}
	target, err := store.CreateCategory(ctx, "Nails")
	if err != nil {
		t.Fatalf("create target category: %v", err)
	}
	first, err := store.CreateService(ctx, source.ID, domain.Service{Name: "Cut", DurationMinute: 30, PriceMinor: 2500, CurrencyCode: "USD"}, 0)
	if err != nil {
		t.Fatalf("create first service: %v", err)
	}
	second, err := store.CreateService(ctx, source.ID, domain.Service{Name: "Color", DurationMinute: 60, PriceMinor: 7000, CurrencyCode: "USD"}, 0)
	if err != nil {
		t.Fatalf("create second service: %v", err)
	}
	movedService, err := store.MoveService(ctx, first.ID, target.ID, 1, first.Version)
	if err != nil {
		t.Fatalf("move service: %v", err)
	}
	categories, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list catalog: %v", err)
	}
	if len(categories) != 2 || len(categories[0].Services) != 1 || categories[0].Services[0].ID != second.ID || categories[0].Services[0].SortOrder != 1 {
		t.Fatalf("source list not compacted: %+v", categories)
	}
	moved := categories[1].Services
	if len(moved) != 1 || moved[0].ID != first.ID || moved[0].CategoryID != target.ID || moved[0].SortOrder != 1 {
		t.Fatalf("destination list mismatch: %+v", moved)
	}
	if _, err := store.MoveService(ctx, first.ID, target.ID, 0, movedService.Version); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid move error = %v, want ErrInvalid", err)
	}
	categories, err = store.List(ctx)
	if err != nil {
		t.Fatalf("read catalog after rejected move: %v", err)
	}
	if categories[0].Services[0].ID != second.ID || categories[1].Services[0].ID != first.ID {
		t.Fatalf("rejected move changed stored state: %+v", categories)
	}
}

func TestConcurrentMovesAndCategoryCascade(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	source, err := store.CreateCategory(ctx, "Source")
	if err != nil {
		t.Fatal(err)
	}
	secondSource, err := store.CreateCategory(ctx, "Second Source")
	if err != nil {
		t.Fatal(err)
	}
	target, err := store.CreateCategory(ctx, "Target")
	if err != nil {
		t.Fatal(err)
	}
	services := make([]domain.Service, 2)
	categories := []domain.Category{source, secondSource}
	for i := range services {
		created, err := store.CreateService(ctx, categories[i].ID, domain.Service{Name: "Service", DurationMinute: 20, CurrencyCode: "USD"}, 0)
		if err != nil {
			t.Fatal(err)
		}
		services[i] = created
	}
	start := make(chan struct{})
	errorsFound := make(chan error, len(services))
	var workers sync.WaitGroup
	for _, service := range services {
		workers.Add(1)
		go func() {
			defer workers.Done()
			<-start
			_, err := store.MoveService(ctx, service.ID, target.ID, 1, service.Version)
			errorsFound <- err
		}()
	}
	close(start)
	workers.Wait()
	close(errorsFound)
	for err := range errorsFound {
		if err != nil {
			t.Fatalf("concurrent move: %v", err)
		}
	}
	categories, err = store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(categories) != 3 || len(categories[0].Services) != 0 || len(categories[1].Services) != 0 || len(categories[2].Services) != 2 || categories[2].Services[0].SortOrder != 1 || categories[2].Services[1].SortOrder != 2 {
		t.Fatalf("concurrent moves left inconsistent order: %+v", categories)
	}
	movedCategory, err := store.MoveCategory(ctx, target.ID, 1, target.Version)
	if err != nil {
		t.Fatalf("move category: %v", err)
	}
	if err := store.DeleteCategory(ctx, target.ID, movedCategory.Version); err != nil {
		t.Fatalf("delete category: %v", err)
	}
	categories, err = store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(categories) != 2 || categories[0].ID != source.ID || categories[0].SortOrder != 1 || categories[1].ID != secondSource.ID || categories[1].SortOrder != 2 || len(categories[0].Services) != 0 || len(categories[1].Services) != 0 {
		t.Fatalf("category cascade or category renumber failed: %+v", categories)
	}
}
