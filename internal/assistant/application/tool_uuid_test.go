package application_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"go-service/internal/assistant/application"
	"go-service/internal/assistant/domain"
)

func TestToolExecutorInvalidUUIDRejection(t *testing.T) {
	conv := application.Conversation{ID: uuid.NewString(), CustomerID: uuid.NewString()}
	validID := uuid.NewString()
	cases := []struct {
		name string
		tool string
		args string
	}{
		{"create invalid service_id", "create_appointment", fmt.Sprintf(`{"service_id":"not-a-uuid","person_id":%q,"start_at":"2026-12-01T15:00:00Z","confirmed_by_customer":true}`, validID)},
		{"create invalid person_id", "create_appointment", fmt.Sprintf(`{"service_id":%q,"person_id":"bad-person","start_at":"2026-12-01T15:00:00Z","confirmed_by_customer":true}`, validID)},
		{"reschedule invalid appointment_id", "reschedule_appointment", `{"appointment_id":"bad-apt","start_at":"2026-12-01T15:00:00Z","confirmed_by_customer":true}`},
		{"reschedule invalid optional service_id", "reschedule_appointment", fmt.Sprintf(`{"appointment_id":%q,"service_id":"bad-svc","confirmed_by_customer":true}`, validID)},
		{"reschedule invalid optional person_id", "reschedule_appointment", fmt.Sprintf(`{"appointment_id":%q,"person_id":"bad-per","confirmed_by_customer":true}`, validID)},
		{"cancel invalid appointment_id", "cancel_appointment", `{"appointment_id":"bad-apt","confirmed_by_customer":true}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cal := &stubCalendar{}
			exec := application.NewToolExecutor(&stubCRM{}, stubCatalog{}, cal, conv, true, "PEN")
			out := exec.Execute(context.Background(), domain.ToolCall{
				ID: "1", Function: domain.ToolCallFunction{Name: tc.tool, Arguments: tc.args},
			})
			var payload map[string]string
			if err := json.Unmarshal([]byte(out), &payload); err != nil {
				t.Fatalf("expected valid json, got %s: %v", out, err)
			}
			if payload["error"] != "invalid_id" || payload["hint"] == "" {
				t.Fatalf("expected invalid_id with hint, got %s", out)
			}
			if cal.createCalls != 0 {
				t.Fatalf("expected calendar not to be called on invalid UUID")
			}
		})
	}
}

func TestToolExecutorConfirmedBookingFlowWithoutHandoff(t *testing.T) {
	conv := application.Conversation{ID: uuid.NewString(), CustomerID: uuid.NewString()}
	crm := &stubCRM{}
	cal := &stubCalendar{}
	svcID, personID := uuid.NewString(), uuid.NewString()
	exec := application.NewToolExecutor(crm, stubCatalog{}, cal, conv, true, "PEN")

	_ = exec.Execute(context.Background(), domain.ToolCall{
		ID: "1", Function: domain.ToolCallFunction{Name: "list_services", Arguments: `{}`},
	})
	_ = exec.Execute(context.Background(), domain.ToolCall{
		ID: "2", Function: domain.ToolCallFunction{Name: "get_available_slots", Arguments: fmt.Sprintf(`{"date":"2026-12-01","service_id":%q}`, svcID)},
	})
	out := exec.Execute(context.Background(), domain.ToolCall{
		ID: "3", Function: domain.ToolCallFunction{
			Name:      "create_appointment",
			Arguments: fmt.Sprintf(`{"service_id":%q,"person_id":%q,"start_at":"2026-12-01T15:00:00Z","confirmed_by_customer":true}`, svcID, personID),
		},
	})
	if cal.createCalls != 1 || exec.HandedOff() || crm.handoffReason != "" {
		t.Fatalf("expected confirmed booking without handoff: calls=%d handedOff=%v reason=%q out=%s", cal.createCalls, exec.HandedOff(), crm.handoffReason, out)
	}
}
