package calendar

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/infrastructure/acl"
	"go-service/internal/calendar/infrastructure/persistence/postgres"
	"go-service/internal/catalog"
	"go-service/internal/organizations"
	sharedpostgres "go-service/internal/shared/postgres"
)

// AssistantCalendarGateway exposes assistant-scoped calendar reads and WhatsApp booking mutations.
type AssistantCalendarGateway struct {
	db             *sql.DB
	store          *postgres.Store
	people         application.PersonResolver
	serviceReader  *catalog.ServiceReader
	scheduleReader *organizations.ScheduleReader
	now            func() time.Time
}

// NewAssistantCalendarGateway creates an assistant calendar gateway backed by PostgreSQL and ACL readers.
func NewAssistantCalendarGateway(
	db *sharedpostgres.Client,
	people application.PersonResolver,
	serviceReader *catalog.ServiceReader,
	scheduleReader *organizations.ScheduleReader,
) *AssistantCalendarGateway {
	var sqlDB *sql.DB
	if db != nil {
		sqlDB = db.DB
	}
	return &AssistantCalendarGateway{
		db:             sqlDB,
		store:          postgres.NewStore(db),
		people:         people,
		serviceReader:  serviceReader,
		scheduleReader: scheduleReader,
		now:            time.Now,
	}
}

// ListAvailability returns single-day 30-minute open slots in the organization's timezone.
func (g *AssistantCalendarGateway) ListAvailability(ctx context.Context, query AvailabilityQuery) ([]AvailabilitySlot, error) {
	if g == nil || g.db == nil || g.serviceReader == nil {
		return nil, ErrGatewayUnavailable
	}
	svc, err := g.serviceReader.GetService(ctx, query.ServiceID)
	if errors.Is(err, catalog.ErrServiceNotFound) {
		return nil, ErrServiceNotFound
	}
	if err != nil {
		return nil, err
	}
	policy, err := acl.NewOrganizationAdapter(g.scheduleReader).GetSchedulePolicy(ctx)
	if err != nil {
		return nil, err
	}
	day, _, err := parseDateInPolicy(query.Date, policy)
	if err != nil {
		return nil, err
	}
	people, err := g.people.ListActivePeople(ctx)
	if err != nil {
		return nil, err
	}
	dayEnd := day.AddDate(0, 0, 1)
	existing, err := g.store.ListAppointments(ctx, application.AppointmentFilter{From: &day, To: &dayEnd})
	if err != nil {
		return nil, err
	}
	return computeAvailability(g.now(), policy, query.Date, svc.DurationMinutes, people, existing, query)
}

// ListCustomerUpcomingAppointments returns future pending/confirmed appointments owned by customerID.
func (g *AssistantCalendarGateway) ListCustomerUpcomingAppointments(ctx context.Context, customerID string) ([]UpcomingAppointment, error) {
	if g == nil || g.db == nil || g.serviceReader == nil {
		return nil, ErrGatewayUnavailable
	}
	rows, err := g.db.QueryContext(ctx,
		`SELECT id, service_id, person_id, starts_at, ends_at, status
		 FROM calendar_appointments
		 WHERE customer_id = $1 AND starts_at > NOW() AND status IN ('pending', 'confirmed')
		 ORDER BY starts_at ASC, id ASC`, customerID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items, svcIDs, personIDs := make([]UpcomingAppointment, 0), make([]string, 0), make([]string, 0)
	for rows.Next() {
		var u UpcomingAppointment
		if err := rows.Scan(&u.ID, &u.ServiceID, &u.PersonID, &u.StartAt, &u.EndAt, &u.Status); err != nil {
			return nil, err
		}
		items, svcIDs, personIDs = append(items, u), append(svcIDs, u.ServiceID), append(personIDs, u.PersonID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return g.hydrateUpcoming(ctx, items, svcIDs, personIDs)
}

func (g *AssistantCalendarGateway) hydrateUpcoming(ctx context.Context, items []UpcomingAppointment, svcIDs, personIDs []string) ([]UpcomingAppointment, error) {
	if len(items) == 0 {
		return items, nil
	}
	svcs, err := g.serviceReader.GetServicesByIDs(ctx, svcIDs)
	if err != nil {
		return nil, err
	}
	ppl, err := g.people.GetPeopleByIDs(ctx, personIDs)
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].ServiceName = svcs[items[i].ServiceID].Name
		items[i].PersonName = ppl[items[i].PersonID].Name
	}
	return items, nil
}
