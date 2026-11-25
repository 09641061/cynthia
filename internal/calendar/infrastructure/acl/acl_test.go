package acl_test

import (
	"context"
	"errors"
	"testing"

	"go-service/internal/calendar/domain"
	"go-service/internal/calendar/infrastructure/acl"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/organizations"
)

func TestCustomerAdapterNilReaderTranslatesNotFound(t *testing.T) {
	adapter := acl.NewCustomerAdapter(nil)
	_, err := adapter.GetCustomer(context.Background(), "cust-1")
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("got %v, want %v", err, domain.ErrCustomerNotFound)
	}
	batch, err := adapter.GetCustomersByIDs(context.Background(), []string{"cust-1"})
	if err != nil || len(batch) != 0 {
		t.Fatalf("got (%v, %v), want empty map and nil error", batch, err)
	}
}

func TestCustomerAdapterWithNilDBReaderTranslatesNotFound(t *testing.T) {
	adapter := acl.NewCustomerAdapter(crm.NewCustomerReader(nil))
	_, err := adapter.GetCustomer(context.Background(), "cust-1")
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("got %v, want %v", err, domain.ErrCustomerNotFound)
	}
	batch, err := adapter.GetCustomersByIDs(context.Background(), []string{"cust-1"})
	if err != nil || len(batch) != 0 {
		t.Fatalf("got (%v, %v), want empty map and nil error", batch, err)
	}
}

func TestServiceAdapterNilReaderTranslatesNotFound(t *testing.T) {
	adapter := acl.NewServiceAdapter(nil)
	_, err := adapter.GetService(context.Background(), "svc-1")
	if !errors.Is(err, domain.ErrServiceNotFound) {
		t.Fatalf("got %v, want %v", err, domain.ErrServiceNotFound)
	}
	batch, err := adapter.GetServicesByIDs(context.Background(), []string{"svc-1"})
	if err != nil || len(batch) != 0 {
		t.Fatalf("got (%v, %v), want empty map and nil error", batch, err)
	}
}

func TestServiceAdapterWithNilDBReaderTranslatesNotFound(t *testing.T) {
	adapter := acl.NewServiceAdapter(catalog.NewServiceReader(nil))
	_, err := adapter.GetService(context.Background(), "svc-1")
	if !errors.Is(err, domain.ErrServiceNotFound) {
		t.Fatalf("got %v, want %v", err, domain.ErrServiceNotFound)
	}
	batch, err := adapter.GetServicesByIDs(context.Background(), []string{"svc-1"})
	if err != nil || len(batch) != 0 {
		t.Fatalf("got (%v, %v), want empty map and nil error", batch, err)
	}
}

func TestOrganizationAdapterNilAndNilDBReader(t *testing.T) {
	for _, adapter := range []acl.OrganizationAdapter{
		acl.NewOrganizationAdapter(nil),
		acl.NewOrganizationAdapter(organizations.NewScheduleReader(nil)),
	} {
		policy, err := adapter.GetSchedulePolicy(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if policy.Timezone != "" || len(policy.Days) != 0 {
			t.Fatalf("expected empty policy, got %+v", policy)
		}
	}
}
