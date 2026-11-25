package catalog

import (
	"context"
	"testing"
)

func TestListCatalogServicesNilDB(t *testing.T) {
	reader := NewServiceReader(nil)
	items, err := reader.ListCatalogServices(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}
