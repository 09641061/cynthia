//go:build integration

package persistence_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

func TestCustomerSearchPaginationAndUpdates(t *testing.T) {
	store, db := newTestStore(t)
	ctx := t.Context()

	c1, err := store.CreateCustomer(ctx, domain.Customer{ID: uuid.NewString(), Name: "Ana Gomez", Phone: "+573001112233"})
	if err != nil {
		t.Fatalf("create c1: %v", err)
	}
	c2, err := store.CreateCustomer(ctx, domain.Customer{ID: uuid.NewString(), Name: "Luis Perez", Phone: "+573009998877"})
	if err != nil {
		t.Fatalf("create c2: %v", err)
	}

	items, total, err := store.ListCustomers(ctx, "300-111", 1, 10)
	if err != nil || total != 1 || len(items) != 1 || items[0].ID != c1.ID {
		t.Fatalf("search by phone digits: total=%d items=%+v err=%v", total, items, err)
	}
	items, total, err = store.ListCustomers(ctx, "", 1, 1)
	if err != nil || total != 2 || len(items) != 1 {
		t.Fatalf("pagination: total=%d len=%d err=%v", total, len(items), err)
	}

	updated, err := store.UpdateCustomer(ctx, c1.ID, "Ana Maria", c1.Phone)
	if err != nil || updated.Name != "Ana Maria" {
		t.Fatalf("update keeping own phone: updated=%+v err=%v", updated, err)
	}
	_, err = store.UpdateCustomer(ctx, c1.ID, "Conflict", c2.Phone)
	var dupErr application.DuplicatePhoneError
	if !errors.As(err, &dupErr) || dupErr.Existing.ID != c2.ID {
		t.Fatalf("expected DuplicatePhoneError(%s), got %v", c2.ID, err)
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO crm_conversations (id, customer_id, status) VALUES ($1, $2, 'open')`,
		uuid.NewString(), c1.ID,
	); err != nil {
		t.Fatalf("insert conversation: %v", err)
	}
	if err := store.DeleteCustomer(ctx, c1.ID); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("expected ErrConflict deleting customer with conversation, got %v", err)
	}
	if err := store.DeleteCustomer(ctx, c2.ID); err != nil {
		t.Fatalf("delete customer without conversations: %v", err)
	}
}

func TestConcurrentDuplicatePhoneCreation(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()
	phone := "+573112223344"

	const workers = 6
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := store.CreateCustomer(ctx, domain.Customer{
				ID: uuid.NewString(), Name: "Concurrent", Phone: phone,
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)

	createdCount, dupCount := 0, 0
	for err := range errs {
		if err == nil {
			createdCount++
			continue
		}
		var dupErr application.DuplicatePhoneError
		if errors.As(err, &dupErr) && dupErr.Existing.Phone == phone {
			dupCount++
			continue
		}
		t.Fatalf("unexpected concurrent error: %v", err)
	}
	if createdCount != 1 || dupCount != workers-1 {
		t.Fatalf("created=%d dup=%d, want 1 and %d", createdCount, dupCount, workers-1)
	}
}
