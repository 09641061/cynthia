package calendar

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"go-service/internal/calendar/domain"
	"go-service/internal/calendar/infrastructure/acl"
	"go-service/internal/catalog"
)

// CreateWhatsAppAppointment creates a confirmed WhatsApp appointment with server-side end_at calculation.
func (g *AssistantCalendarGateway) CreateWhatsAppAppointment(ctx context.Context, in CreateWhatsAppAppointmentInput) (AssistantAppointment, error) {
	if g == nil || g.db == nil || g.serviceReader == nil {
		return AssistantAppointment{}, ErrGatewayUnavailable
	}
	endAt, err := g.validateBookingTarget(ctx, in.ServiceID, in.PersonID, in.StartAt)
	if err != nil {
		return AssistantAppointment{}, err
	}
	created, err := g.store.CreateAppointment(ctx, domain.Appointment{
		ID: uuid.NewString(), PersonID: in.PersonID, CustomerID: in.CustomerID, ServiceID: in.ServiceID,
		StartsAt: in.StartAt.UTC(), EndsAt: endAt.UTC(), Status: domain.StatusConfirmed, Channel: domain.ChannelWhatsApp, Notes: in.Notes,
	})
	if err != nil {
		return AssistantAppointment{}, mapGatewayError(err)
	}
	return AssistantAppointment{
		ID: created.ID, ServiceID: created.ServiceID, PersonID: created.PersonID,
		StartAt: created.StartsAt, EndAt: created.EndsAt, Status: string(created.Status),
	}, nil
}

// RescheduleWhatsAppAppointment atomically reschedules or changes service/person on a customer's upcoming appointment.
func (g *AssistantCalendarGateway) RescheduleWhatsAppAppointment(ctx context.Context, in RescheduleWhatsAppAppointmentInput) (AssistantAppointment, error) {
	if g == nil || g.db == nil || g.serviceReader == nil {
		return AssistantAppointment{}, ErrGatewayUnavailable
	}
	if in.StartAt == nil && in.ServiceID == nil && in.PersonID == nil {
		return AssistantAppointment{}, ErrNoFieldsToUpdate
	}
	tx, err := g.db.BeginTx(ctx, nil)
	if err != nil {
		return AssistantAppointment{}, err
	}
	defer func() { _ = tx.Rollback() }()
	var curPersonID, curServiceID string
	var curStartsAt time.Time
	err = tx.QueryRowContext(ctx,
		`SELECT person_id, service_id, starts_at FROM calendar_appointments
		 WHERE id = $1 AND customer_id = $2 AND status IN ('pending', 'confirmed') AND starts_at > NOW() FOR UPDATE`,
		in.AppointmentID, in.CustomerID,
	).Scan(&curPersonID, &curServiceID, &curStartsAt)
	if errors.Is(err, sql.ErrNoRows) {
		return AssistantAppointment{}, ErrAppointmentNotFound
	}
	if err != nil {
		return AssistantAppointment{}, err
	}
	targetPerson, targetService, targetStart := in.resolveTargets(curPersonID, curServiceID, curStartsAt)
	targetEnd, err := g.validateBookingTarget(ctx, targetService, targetPerson, targetStart)
	if err != nil {
		return AssistantAppointment{}, err
	}
	var res AssistantAppointment
	err = tx.QueryRowContext(ctx,
		`UPDATE calendar_appointments
		 SET person_id = $1, service_id = $2, starts_at = $3, ends_at = $4, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $5 AND customer_id = $6 AND status IN ('pending', 'confirmed') AND starts_at > NOW()
		 RETURNING id, service_id, person_id, starts_at, ends_at, status`,
		targetPerson, targetService, targetStart.UTC(), targetEnd.UTC(), in.AppointmentID, in.CustomerID,
	).Scan(&res.ID, &res.ServiceID, &res.PersonID, &res.StartAt, &res.EndAt, &res.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return AssistantAppointment{}, ErrAppointmentNotFound
	}
	if err != nil {
		return AssistantAppointment{}, mapGatewayError(err)
	}
	if err := tx.Commit(); err != nil {
		return AssistantAppointment{}, err
	}
	return res, nil
}

// CancelWhatsAppAppointment atomically cancels a customer's upcoming pending/confirmed appointment.
func (g *AssistantCalendarGateway) CancelWhatsAppAppointment(ctx context.Context, appointmentID, customerID string) (AssistantAppointment, error) {
	if g == nil || g.db == nil {
		return AssistantAppointment{}, ErrGatewayUnavailable
	}
	var res AssistantAppointment
	err := g.db.QueryRowContext(ctx,
		`UPDATE calendar_appointments
		 SET status = 'cancelled', updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1 AND customer_id = $2 AND status IN ('pending', 'confirmed') AND starts_at > NOW()
		 RETURNING id, service_id, person_id, starts_at, ends_at, status`,
		appointmentID, customerID,
	).Scan(&res.ID, &res.ServiceID, &res.PersonID, &res.StartAt, &res.EndAt, &res.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return AssistantAppointment{}, ErrAppointmentNotFound
	}
	return res, err
}

func (g *AssistantCalendarGateway) validateBookingTarget(ctx context.Context, serviceID, personID string, startAt time.Time) (time.Time, error) {
	if startAt.IsZero() || !startAt.After(g.now()) {
		return time.Time{}, ErrSlotInPast
	}
	svc, err := g.serviceReader.GetService(ctx, serviceID)
	if errors.Is(err, catalog.ErrServiceNotFound) {
		return time.Time{}, ErrServiceNotFound
	}
	if err != nil {
		return time.Time{}, err
	}
	person, err := g.people.GetPerson(ctx, personID)
	if errors.Is(err, domain.ErrPersonNotFound) || (err == nil && !person.Active) {
		return time.Time{}, ErrPersonNotFound
	}
	if err != nil {
		return time.Time{}, err
	}
	endAt := startAt.Add(time.Duration(svc.DurationMinutes) * time.Minute)
	policy, err := acl.NewOrganizationAdapter(g.scheduleReader).GetSchedulePolicy(ctx)
	if err != nil {
		return time.Time{}, err
	}
	if err := domain.ValidateWithinBusinessHours(startAt, endAt, policy); err != nil {
		return time.Time{}, ErrOutsideBusinessHours
	}
	return endAt, nil
}

func mapGatewayError(err error) error {
	if errors.Is(err, domain.ErrTimeConflict) {
		return ErrSlotConflict
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
		return ErrSlotConflict
	}
	return err
}
