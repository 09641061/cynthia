package application_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go-service/internal/assistant/application"
	"go-service/internal/assistant/domain"
)

type stubCRM struct {
	handoffReason string
}

func (s *stubCRM) GetConversation(context.Context, string) (application.Conversation, error) {
	return application.Conversation{}, nil
}
func (s *stubCRM) ListRecentMessages(context.Context, string, int) ([]application.Message, error) {
	return nil, nil
}
func (s *stubCRM) SendAssistantReply(context.Context, string, string, string) (application.Message, error) {
	return application.Message{}, nil
}
func (s *stubCRM) PauseForHandoff(_ context.Context, _, reason string) error {
	s.handoffReason = reason
	return nil
}

type stubCatalog struct{}

func (stubCatalog) ListCatalogServices(context.Context) ([]application.ServiceItem, error) {
	return []application.ServiceItem{{ID: "svc-1", Name: "Cut", DurationMinutes: 30, PriceMinorUnits: 2500}}, nil
}

type stubCalendar struct {
	lastCustomerID string
	createCalls    int
}

func (s *stubCalendar) ListAvailability(_ context.Context, q application.AvailabilityQuery) ([]application.AvailabilitySlot, error) {
	s.lastCustomerID = q.CustomerID
	return []application.AvailabilitySlot{{PersonID: "p-1", PersonName: "Ana"}}, nil
}
func (s *stubCalendar) ListCustomerUpcomingAppointments(_ context.Context, customerID string) ([]application.UpcomingAppointment, error) {
	s.lastCustomerID = customerID
	return []application.UpcomingAppointment{{ID: "apt-1"}}, nil
}
func (s *stubCalendar) CreateWhatsAppAppointment(_ context.Context, in application.CreateAppointmentInput) (application.AppointmentResult, error) {
	s.lastCustomerID = in.CustomerID
	s.createCalls++
	return application.AppointmentResult{ID: "apt-new", Status: "confirmed", StartAt: in.StartAt, EndAt: in.StartAt.Add(30 * time.Minute)}, nil
}
func (s *stubCalendar) RescheduleWhatsAppAppointment(_ context.Context, in application.RescheduleAppointmentInput) (application.AppointmentResult, error) {
	s.lastCustomerID = in.CustomerID
	return application.AppointmentResult{ID: in.AppointmentID, Status: "confirmed"}, nil
}
func (s *stubCalendar) CancelWhatsAppAppointment(_ context.Context, id, customerID string) (application.AppointmentResult, error) {
	s.lastCustomerID = customerID
	return application.AppointmentResult{ID: id, Status: "cancelled"}, nil
}

func TestToolExecutorConfirmationGateAndMutationLimit(t *testing.T) {
	conv := application.Conversation{ID: uuid.NewString(), CustomerID: "cust-server"}
	crm := &stubCRM{}
	cal := &stubCalendar{}
	svcID, personID, aptID := uuid.NewString(), uuid.NewString(), uuid.NewString()
	argsConfirmed := fmt.Sprintf(`{"service_id":%q,"person_id":%q,"start_at":"2026-12-01T15:00:00Z","confirmed_by_customer":true}`, svcID, personID)

	// 1. No prior assistant turn -> confirmation_required
	execNoPrior := application.NewToolExecutor(crm, stubCatalog{}, cal, conv, false, "PEN")
	out := execNoPrior.Execute(context.Background(), domain.ToolCall{
		ID: "1", Function: domain.ToolCallFunction{Name: "create_appointment", Arguments: argsConfirmed},
	})
	if !strings.Contains(out, "confirmation_required") {
		t.Fatalf("expected confirmation_required without prior turn, got %s", out)
	}

	// 2. Prior turn + read tools (UUID recovery) + confirmed mutation -> succeeds once, second hits mutation_limit_reached
	exec := application.NewToolExecutor(crm, stubCatalog{}, cal, conv, true, "PEN")
	_ = exec.Execute(context.Background(), domain.ToolCall{
		ID: "2", Function: domain.ToolCallFunction{Name: "list_services", Arguments: `{}`},
	})
	_ = exec.Execute(context.Background(), domain.ToolCall{
		ID: "3", Function: domain.ToolCallFunction{Name: "get_available_slots", Arguments: fmt.Sprintf(`{"date":"2026-12-01","service_id":%q}`, svcID)},
	})
	out = exec.Execute(context.Background(), domain.ToolCall{
		ID: "4", Function: domain.ToolCallFunction{Name: "create_appointment", Arguments: argsConfirmed},
	})
	if !strings.Contains(out, "apt-new") || cal.lastCustomerID != "cust-server" {
		t.Fatalf("expected successful create bound to cust-server, got %s (cust=%s)", out, cal.lastCustomerID)
	}
	out = exec.Execute(context.Background(), domain.ToolCall{
		ID: "5", Function: domain.ToolCallFunction{Name: "cancel_appointment", Arguments: fmt.Sprintf(`{"appointment_id":%q,"confirmed_by_customer":true}`, aptID)},
	})
	if !strings.Contains(out, "mutation_limit_reached") {
		t.Fatalf("expected mutation_limit_reached, got %s", out)
	}
}

func TestToolExecutorHandoffToHuman(t *testing.T) {
	crm := &stubCRM{}
	exec := application.NewToolExecutor(crm, stubCatalog{}, &stubCalendar{}, application.Conversation{ID: "c1", CustomerID: "u1"}, true, "PEN")
	out := exec.Execute(context.Background(), domain.ToolCall{
		ID: "h1", Function: domain.ToolCallFunction{Name: "handoff_to_human", Arguments: `{"reason":"billing question"}`},
	})
	if !exec.HandedOff() || crm.handoffReason != "billing question" || !strings.Contains(out, `"handed_off":true`) {
		t.Fatalf("unexpected handoff result: out=%s reason=%s handedOff=%v", out, crm.handoffReason, exec.HandedOff())
	}
}

func TestToolExecutorReadToolsRescheduleAndUnconfirmed(t *testing.T) {
	conv := application.Conversation{ID: uuid.NewString(), CustomerID: "cust-1"}
	cal := &stubCalendar{}
	exec := application.NewToolExecutor(&stubCRM{}, stubCatalog{}, cal, conv, true, "PEN")
	aptID := uuid.NewString()

	out := exec.Execute(context.Background(), domain.ToolCall{
		ID: "u1", Function: domain.ToolCallFunction{Name: "reschedule_appointment", Arguments: fmt.Sprintf(`{"appointment_id":%q,"start_at":"2026-12-02T15:00:00Z","confirmed_by_customer":false}`, aptID)},
	})
	if !strings.Contains(out, "confirmation_required") {
		t.Fatalf("expected confirmation_required for unconfirmed reschedule, got %s", out)
	}
	out = exec.Execute(context.Background(), domain.ToolCall{
		ID: "r1", Function: domain.ToolCallFunction{Name: "reschedule_appointment", Arguments: fmt.Sprintf(`{"appointment_id":%q,"start_at":"2026-12-02T15:00:00Z","confirmed_by_customer":true}`, aptID)},
	})
	if !strings.Contains(out, aptID) || cal.lastCustomerID != "cust-1" {
		t.Fatalf("unexpected reschedule output: %s", out)
	}
	slotsOut := exec.Execute(context.Background(), domain.ToolCall{
		ID: "s1", Function: domain.ToolCallFunction{Name: "get_available_slots", Arguments: `{"date":"2026-12-02","service_id":"svc-1"}`},
	})
	aptsOut := exec.Execute(context.Background(), domain.ToolCall{
		ID: "l1", Function: domain.ToolCallFunction{Name: "list_my_appointments", Arguments: `{}`},
	})
	if !strings.Contains(slotsOut, "Ana") || !strings.Contains(aptsOut, "apt-1") {
		t.Fatalf("unexpected read tool outputs: slots=%s apts=%s", slotsOut, aptsOut)
	}
}
