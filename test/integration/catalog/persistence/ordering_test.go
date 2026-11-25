//go:build integration

package persistence_test

import (
	"errors"
	"testing"

	"go-service/internal/catalog/domain"
)

func TestPositionMovesAndDeletesPreserveIdentity(t *testing.T) {
	store := newTestStore(t)
	ctx := t.Context()
	first, err := store.CreateCategory(ctx, "First")
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.CreateCategory(ctx, "Second")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.MoveCategory(ctx, second.ID, 1); err != nil {
		t.Fatalf("move category: %v", err)
	}
	loadedCategory, err := store.GetCategory(ctx, second.ID)
	if err != nil || loadedCategory.SortOrder != 1 {
		t.Fatalf("category read after move = %+v, error %v", loadedCategory, err)
	}
	services := make([]domain.Service, 3)
	for index := range services {
		position := 0
		if index == 1 {
			position = 1
		}
		services[index], err = store.CreateService(ctx, first.ID, domain.Service{Name: "Service", DurationMinute: 30}, position)
		if err != nil {
			t.Fatalf("create service: %v", err)
		}
	}
	categories, err := store.List(ctx)
	if err != nil {
		t.Fatalf("list after positional insert: %v", err)
	}
	initial := categories[1].Services
	if len(initial) != 3 || initial[0].ID != services[1].ID || initial[1].ID != services[0].ID || initial[2].ID != services[2].ID {
		t.Fatalf("positional insert order = %+v", initial)
	}
	if _, err := store.MoveService(ctx, services[2].ID, first.ID, 1); err != nil {
		t.Fatalf("move service to first position: %v", err)
	}
	current, err := store.GetService(ctx, services[0].ID)
	if err != nil {
		t.Fatalf("read service: %v", err)
	}
	moved, err := store.MoveService(ctx, current.ID, first.ID, 2)
	if err != nil {
		t.Fatalf("move service to second position: %v", err)
	}
	if _, err := store.MoveService(ctx, moved.ID, first.ID, 0); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("invalid position error = %v, want ErrInvalid", err)
	}
	if err := store.DeleteService(ctx, moved.ID); err != nil {
		t.Fatalf("delete service: %v", err)
	}
	categories, err = store.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if categories[0].ID != second.ID || categories[0].SortOrder != 1 || categories[1].ID != first.ID || categories[1].SortOrder != 2 {
		t.Fatalf("category positions = %+v", categories)
	}
	remaining := categories[1].Services
	if len(remaining) != 2 || remaining[0].ID != services[2].ID || remaining[0].SortOrder != 1 || remaining[1].ID != services[1].ID || remaining[1].SortOrder != 2 {
		t.Fatalf("service positions = %+v", remaining)
	}
}
