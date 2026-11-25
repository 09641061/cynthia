package application

import (
	"context"
	"time"

	"go-service/internal/calendar/domain"
)

// AppointmentFilter constrains appointment listing queries.
type AppointmentFilter struct {
	From       *time.Time
	To         *time.Time
	PersonID   string
	CustomerID string
	Status     string
}

// AppointmentStore defines persistence operations for calendar appointments.
type AppointmentStore interface {
	ListAppointments(ctx context.Context, filter AppointmentFilter) ([]domain.Appointment, error)
	GetAppointment(ctx context.Context, id string) (domain.Appointment, error)
	CreateAppointment(ctx context.Context, appt domain.Appointment) (domain.Appointment, error)
	UpdateAppointment(ctx context.Context, expectedVersion int64, appt domain.Appointment) (domain.Appointment, error)
	DeleteAppointment(ctx context.Context, id string, expectedVersion int64) error
}

// PersonResolver resolves bookable people in the calendar context.
type PersonResolver interface {
	ListActivePeople(ctx context.Context) ([]domain.Person, error)
	GetPerson(ctx context.Context, id string) (domain.Person, error)
	GetPeopleByIDs(ctx context.Context, ids []string) (map[string]domain.Person, error)
}

// CustomerSummary is the minimal customer projection needed by calendar.
type CustomerSummary struct {
	ID    string
	Name  string
	Phone string
}

// CustomerReader reads customer summaries from CRM.
type CustomerReader interface {
	GetCustomer(ctx context.Context, id string) (CustomerSummary, error)
	GetCustomersByIDs(ctx context.Context, ids []string) (map[string]CustomerSummary, error)
}

// ServiceSummary is the minimal service projection needed by calendar.
type ServiceSummary struct {
	ID              string
	Name            string
	DurationMinutes int
}

// ServiceReader reads service summaries from catalog.
type ServiceReader interface {
	GetService(ctx context.Context, id string) (ServiceSummary, error)
	GetServicesByIDs(ctx context.Context, ids []string) (map[string]ServiceSummary, error)
}

// SchedulePolicyReader reads the organization's timezone and business hours policy.
type SchedulePolicyReader interface {
	GetSchedulePolicy(ctx context.Context) (domain.SchedulePolicy, error)
}

// AppointmentView combines an appointment with hydrated person, customer, and service details.
type AppointmentView struct {
	Appointment            domain.Appointment
	PersonName             string
	CustomerName           string
	CustomerPhone          string
	ServiceName            string
	ServiceDurationMinutes int
}

// CreateAppointmentInput holds input fields for creating an appointment.
type CreateAppointmentInput struct {
	PersonID   string
	CustomerID string
	ServiceID  string
	StartsAt   time.Time
	EndsAt     time.Time
	Status     string
	Channel    string
	Notes      string
}

// UpdateAppointmentInput holds partial update fields for an appointment.
type UpdateAppointmentInput struct {
	PersonID   *string
	CustomerID *string
	ServiceID  *string
	StartsAt   *time.Time
	EndsAt     *time.Time
	Status     *string
	Notes      *string
}
