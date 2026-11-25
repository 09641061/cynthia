//go:build integration

package persistence_test

import (
	"testing"

	"github.com/google/uuid"
	"go-service/internal/crm/domain"
)

func TestCustomerSearchMultiWordAndEscaping(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := t.Context()

	ana, err := store.CreateCustomer(ctx, domain.Customer{
		ID: uuid.NewString(), Name: "Ana Maria Perez", Phone: "+573001112233",
	})
	if err != nil {
		t.Fatalf("create ana: %v", err)
	}
	if _, err := store.CreateCustomer(ctx, domain.Customer{
		ID: uuid.NewString(), Name: "Luis Alberto Gomez", Phone: "+573119998877",
	}); err != nil {
		t.Fatalf("create luis: %v", err)
	}

	for _, query := range []string{"Ana Perez", "Ana 300", "+ 300"} {
		items, total, err := store.ListCustomers(ctx, query, 1, 10)
		if err != nil || total != 1 || len(items) != 1 || items[0].ID != ana.ID {
			t.Fatalf("query %q: total=%d items=%+v err=%v", query, total, items, err)
		}
	}

	for _, query := range []string{"Ana 999", "%", "_", `\`} {
		items, total, err := store.ListCustomers(ctx, query, 1, 10)
		if err != nil || total != 0 || len(items) != 0 {
			t.Fatalf("query %q: got total=%d len=%d err=%v, want 0", query, total, len(items), err)
		}
	}
}
