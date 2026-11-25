package acl

import (
	"context"
	"errors"
	"fmt"

	"go-service/internal/assistant/application"
	"go-service/internal/calendar"
)

// CalendarAdapter adapts calendar.AssistantCalendarGateway to application.CalendarGateway.
type CalendarAdapter struct {
	gw *calendar.AssistantCalendarGateway
}

// NewCalendarAdapter creates a CalendarAdapter.
func NewCalendarAdapter(gw *calendar.AssistantCalendarGateway) *CalendarAdapter {
	return &CalendarAdapter{gw: gw}
}

func (a *CalendarAdapter) ListAvailability(ctx context.Context, q application.AvailabilityQuery) ([]application.AvailabilitySlot, error) {
	slots, err := a.gw.ListAvailability(ctx, calendar.AvailabilityQuery{
		Date: q.Date, ServiceID: q.ServiceID, PersonID: q.PersonID,
		ExcludeAppointmentID: q.ExcludeAppointmentID, CustomerID: q.CustomerID,
	})
	if err != nil {
		return nil, mapCalendarErr(err)
	}
	out := make([]application.AvailabilitySlot, 0, len(slots))
	for _, s := range slots {
		out = append(out, application.AvailabilitySlot{
			StartAt: s.StartAt, EndAt: s.EndAt, PersonID: s.PersonID, PersonName: s.PersonName,
		})
	}
	return out, nil
}

func (a *CalendarAdapter) ListCustomerUpcomingAppointments(ctx context.Context, customerID string) ([]application.UpcomingAppointment, error) {
	items, err := a.gw.ListCustomerUpcomingAppointments(ctx, customerID)
	if err != nil {
		return nil, mapCalendarErr(err)
	}
	out := make([]application.UpcomingAppointment, 0, len(items))
	for _, it := range items {
		out = append(out, application.UpcomingAppointment{
			ID: it.ID, ServiceID: it.ServiceID, ServiceName: it.ServiceName,
			PersonID: it.PersonID, PersonName: it.PersonName,
			StartAt: it.StartAt, EndAt: it.EndAt, Status: it.Status,
		})
	}
	return out, nil
}

func (a *CalendarAdapter) CreateWhatsAppAppointment(ctx context.Context, in application.CreateAppointmentInput) (application.AppointmentResult, error) {
	res, err := a.gw.CreateWhatsAppAppointment(ctx, calendar.CreateWhatsAppAppointmentInput{
		CustomerID: in.CustomerID, ServiceID: in.ServiceID, PersonID: in.PersonID, StartAt: in.StartAt, Notes: in.Notes,
	})
	if err != nil {
		return application.AppointmentResult{}, mapCalendarErr(err)
	}
	return toAppointmentResult(res), nil
}

func (a *CalendarAdapter) RescheduleWhatsAppAppointment(ctx context.Context, in application.RescheduleAppointmentInput) (application.AppointmentResult, error) {
	res, err := a.gw.RescheduleWhatsAppAppointment(ctx, calendar.RescheduleWhatsAppAppointmentInput{
		AppointmentID: in.AppointmentID, CustomerID: in.CustomerID,
		StartAt: in.StartAt, ServiceID: in.ServiceID, PersonID: in.PersonID,
	})
	if err != nil {
		return application.AppointmentResult{}, mapCalendarErr(err)
	}
	return toAppointmentResult(res), nil
}

func (a *CalendarAdapter) CancelWhatsAppAppointment(ctx context.Context, appointmentID, customerID string) (application.AppointmentResult, error) {
	res, err := a.gw.CancelWhatsAppAppointment(ctx, appointmentID, customerID)
	if err != nil {
		return application.AppointmentResult{}, mapCalendarErr(err)
	}
	return toAppointmentResult(res), nil
}

func toAppointmentResult(r calendar.AssistantAppointment) application.AppointmentResult {
	return application.AppointmentResult{
		ID: r.ID, ServiceID: r.ServiceID, PersonID: r.PersonID, StartAt: r.StartAt, EndAt: r.EndAt, Status: r.Status,
	}
}

func mapCalendarErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, calendar.ErrSlotConflict):
		return fmt.Errorf("%w: %w", application.ErrSlotConflict, err)
	case errors.Is(err, calendar.ErrOutsideBusinessHours):
		return fmt.Errorf("%w: %w", application.ErrOutsideBusinessHours, err)
	case errors.Is(err, calendar.ErrSlotInPast):
		return fmt.Errorf("%w: %w", application.ErrSlotInPast, err)
	case errors.Is(err, calendar.ErrServiceNotFound):
		return fmt.Errorf("%w: %w", application.ErrServiceNotFound, err)
	case errors.Is(err, calendar.ErrPersonNotFound):
		return fmt.Errorf("%w: %w", application.ErrPersonNotFound, err)
	case errors.Is(err, calendar.ErrAppointmentNotFound):
		return fmt.Errorf("%w: %w", application.ErrAppointmentNotFound, err)
	case errors.Is(err, calendar.ErrNoFieldsToUpdate):
		return fmt.Errorf("%w: %w", application.ErrNoFieldsToUpdate, err)
	default:
		return err
	}
}
