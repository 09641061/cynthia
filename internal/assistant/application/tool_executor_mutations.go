package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

func (e *ToolExecutor) checkMutationGuard(confirmedByCustomer bool) (string, bool) {
	if e.mutationsThisJob >= 1 {
		return marshalToolError("mutation_limit_reached"), false
	}
	if !confirmedByCustomer || !e.hasPriorAssistantTurn {
		return marshalToolError("confirmation_required"), false
	}
	return "", true
}

func (e *ToolExecutor) execCreateAppointment(ctx context.Context, rawArgs string) string {
	var args struct {
		ServiceID           string `json:"service_id"`
		PersonID            string `json:"person_id"`
		StartAt             string `json:"start_at"`
		Notes               string `json:"notes"`
		ConfirmedByCustomer bool   `json:"confirmed_by_customer"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return marshalToolError("invalid_arguments")
	}
	if msg, ok := e.checkMutationGuard(args.ConfirmedByCustomer); !ok {
		return msg
	}
	startAt, err := time.Parse(time.RFC3339, strings.TrimSpace(args.StartAt))
	if err != nil {
		return marshalToolError("invalid_arguments")
	}
	serviceID, okSvc := parseValidUUID(args.ServiceID)
	personID, okPer := parseValidUUID(args.PersonID)
	if !okSvc || !okPer {
		return marshalInvalidID()
	}
	res, err := e.calendar.CreateWhatsAppAppointment(ctx, CreateAppointmentInput{
		CustomerID: e.customerID,
		ServiceID:  serviceID,
		PersonID:   personID,
		StartAt:    startAt,
		Notes:      strings.TrimSpace(args.Notes),
	})
	if err != nil {
		return marshalMappedError(err)
	}
	e.mutationsThisJob++
	return marshalToolPayload(map[string]any{"appointment": res})
}

func (e *ToolExecutor) execRescheduleAppointment(ctx context.Context, rawArgs string) string {
	var args struct {
		AppointmentID       string  `json:"appointment_id"`
		StartAt             *string `json:"start_at"`
		ServiceID           *string `json:"service_id"`
		PersonID            *string `json:"person_id"`
		ConfirmedByCustomer bool    `json:"confirmed_by_customer"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return marshalToolError("invalid_arguments")
	}
	if msg, ok := e.checkMutationGuard(args.ConfirmedByCustomer); !ok {
		return msg
	}
	aptID, ok := parseValidUUID(args.AppointmentID)
	if !ok {
		return marshalInvalidID()
	}
	svcID, okSvc := parseOptionalUUID(args.ServiceID)
	perID, okPer := parseOptionalUUID(args.PersonID)
	if !okSvc || !okPer {
		return marshalInvalidID()
	}
	in := RescheduleAppointmentInput{AppointmentID: aptID, CustomerID: e.customerID, ServiceID: svcID, PersonID: perID}
	if args.StartAt != nil && strings.TrimSpace(*args.StartAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*args.StartAt))
		if err != nil {
			return marshalToolError("invalid_arguments")
		}
		in.StartAt = &parsed
	}
	res, err := e.calendar.RescheduleWhatsAppAppointment(ctx, in)
	if err != nil {
		return marshalMappedError(err)
	}
	e.mutationsThisJob++
	return marshalToolPayload(map[string]any{"appointment": res})
}

func (e *ToolExecutor) execCancelAppointment(ctx context.Context, rawArgs string) string {
	var args struct {
		AppointmentID       string `json:"appointment_id"`
		ConfirmedByCustomer bool   `json:"confirmed_by_customer"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return marshalToolError("invalid_arguments")
	}
	if msg, ok := e.checkMutationGuard(args.ConfirmedByCustomer); !ok {
		return msg
	}
	aptID, ok := parseValidUUID(args.AppointmentID)
	if !ok {
		return marshalInvalidID()
	}
	res, err := e.calendar.CancelWhatsAppAppointment(ctx, aptID, e.customerID)
	if err != nil {
		return marshalMappedError(err)
	}
	e.mutationsThisJob++
	return marshalToolPayload(map[string]any{"appointment": res})
}

func (e *ToolExecutor) execHandoffToHuman(ctx context.Context, rawArgs string) string {
	var args struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(rawArgs), &args); err != nil {
		return marshalToolError("invalid_arguments")
	}
	reason := strings.TrimSpace(args.Reason)
	if reason == "" {
		reason = "customer_requested_human"
	}
	if err := e.crm.PauseForHandoff(ctx, e.convID, reason); err != nil {
		return marshalMappedError(err)
	}
	e.handedOffInTurn = true
	return marshalToolPayload(map[string]any{"handed_off": true, "reason": reason})
}
