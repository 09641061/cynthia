package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"go-service/internal/assistant/domain"
)

// ToolExecutor executes DeepSeek tool calls for a single job with server-side guards.
type ToolExecutor struct {
	crm                   CRMGateway
	catalog               CatalogReader
	calendar              CalendarGateway
	convID                string
	customerID            string
	currencyCode          string
	hasPriorAssistantTurn bool
	mutationsThisJob      int
	handedOffInTurn       bool
}

// NewToolExecutor creates a per-job tool executor bound to the conversation's customer.
func NewToolExecutor(
	crm CRMGateway,
	catalog CatalogReader,
	calendar CalendarGateway,
	conv Conversation,
	hasPriorAssistantTurn bool,
	currencyCode string,
) *ToolExecutor {
	return &ToolExecutor{
		crm:                   crm,
		catalog:               catalog,
		calendar:              calendar,
		convID:                conv.ID,
		customerID:            conv.CustomerID,
		currencyCode:          currencyCode,
		hasPriorAssistantTurn: hasPriorAssistantTurn,
	}
}

// HandedOff reports whether handoff_to_human was invoked during this job.
func (e *ToolExecutor) HandedOff() bool { return e.handedOffInTurn }

// MutatedOrHandedOff reports whether any appointment mutation or handoff succeeded.
func (e *ToolExecutor) MutatedOrHandedOff() bool { return e.mutationsThisJob > 0 || e.handedOffInTurn }

// Execute dispatches a single tool call and returns a JSON-encoded tool result string.
func (e *ToolExecutor) Execute(ctx context.Context, call domain.ToolCall) string {
	switch call.Function.Name {
	case "list_services":
		return e.execListServices(ctx)
	case "get_available_slots":
		return e.execGetAvailableSlots(ctx, call.Function.Arguments)
	case "list_my_appointments":
		return e.execListMyAppointments(ctx)
	case "create_appointment":
		return e.execCreateAppointment(ctx, call.Function.Arguments)
	case "reschedule_appointment":
		return e.execRescheduleAppointment(ctx, call.Function.Arguments)
	case "cancel_appointment":
		return e.execCancelAppointment(ctx, call.Function.Arguments)
	case "handoff_to_human":
		return e.execHandoffToHuman(ctx, call.Function.Arguments)
	default:
		return marshalToolError("unknown_tool")
	}
}

func (e *ToolExecutor) execListServices(ctx context.Context) string {
	items, err := e.catalog.ListCatalogServices(ctx)
	if err != nil {
		return marshalMappedError(err)
	}
	return marshalToolPayload(map[string]any{"services": mapToolServices(items, e.currencyCode)})
}

func (e *ToolExecutor) execGetAvailableSlots(ctx context.Context, rawArgs string) string {
	var args struct {
		Date                 string `json:"date"`
		ServiceID            string `json:"service_id"`
		PersonID             string `json:"person_id"`
		ExcludeAppointmentID string `json:"exclude_appointment_id"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil || strings.TrimSpace(args.Date) == "" || strings.TrimSpace(args.ServiceID) == "" {
		return marshalToolError("invalid_arguments")
	}
	slots, err := e.calendar.ListAvailability(ctx, AvailabilityQuery{
		Date:                 strings.TrimSpace(args.Date),
		ServiceID:            strings.TrimSpace(args.ServiceID),
		PersonID:             strings.TrimSpace(args.PersonID),
		ExcludeAppointmentID: strings.TrimSpace(args.ExcludeAppointmentID),
		CustomerID:           e.customerID,
	})
	if err != nil {
		return marshalMappedError(err)
	}
	return marshalToolPayload(map[string]any{"slots": slots})
}

func (e *ToolExecutor) execListMyAppointments(ctx context.Context) string {
	items, err := e.calendar.ListCustomerUpcomingAppointments(ctx, e.customerID)
	if err != nil {
		return marshalMappedError(err)
	}
	return marshalToolPayload(map[string]any{"appointments": items})
}

func marshalToolError(code string) string {
	return marshalToolPayload(map[string]string{"error": code})
}

func marshalToolPayload(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `{"error":"internal_error"}`
	}
	return string(b)
}

func marshalMappedError(err error) string {
	switch {
	case errors.Is(err, ErrSlotConflict):
		return marshalToolError("slot_conflict")
	case errors.Is(err, ErrOutsideBusinessHours):
		return marshalToolError("outside_business_hours")
	case errors.Is(err, ErrSlotInPast):
		return marshalToolError("slot_in_past")
	case errors.Is(err, ErrServiceNotFound):
		return marshalToolError("service_not_found")
	case errors.Is(err, ErrPersonNotFound):
		return marshalToolError("person_not_found")
	case errors.Is(err, ErrAppointmentNotFound):
		return marshalToolError("appointment_not_found")
	case errors.Is(err, ErrNoFieldsToUpdate):
		return marshalToolError("no_fields_to_update")
	default:
		return marshalToolError("internal_error")
	}
}
