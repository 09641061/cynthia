package application_test

import (
	"context"
	"strings"
	"testing"

	"go-service/internal/assistant/application"
	"go-service/internal/assistant/domain"
)

func TestFormatMinorPrice(t *testing.T) {
	cases := []struct {
		minor    int64
		currency string
		want     string
	}{
		{3000, "PEN", "PEN 30.00"},
		{15000, "CLP", "CLP 15000"},
		{500, "JPY", "JPY 500"},
		{3000, "", "30.00"},
		{0, "USD", "USD 0.00"},
	}
	for _, tc := range cases {
		if got := application.FormatMinorPrice(tc.minor, tc.currency); got != tc.want {
			t.Errorf("FormatMinorPrice(%d, %q) = %q, want %q", tc.minor, tc.currency, got, tc.want)
		}
	}
}

func TestExecListServicesReturnsFormattedPrice(t *testing.T) {
	conv := application.Conversation{ID: "c1", CustomerID: "cust-1"}
	exec := application.NewToolExecutor(&stubCRM{}, stubCatalog{}, &stubCalendar{}, conv, true, "PEN")
	out := exec.Execute(context.Background(), domain.ToolCall{
		ID: "s1", Function: domain.ToolCallFunction{Name: "list_services", Arguments: `{}`},
	})
	if !strings.Contains(out, `"price_formatted":"PEN 25.00"`) {
		t.Fatalf("expected price_formatted PEN 25.00 in output, got %s", out)
	}
	if !strings.Contains(out, `"currency_code":"PEN"`) {
		t.Fatalf("expected currency_code PEN in output, got %s", out)
	}
}
