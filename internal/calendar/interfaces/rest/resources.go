package rest

import (
	"time"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
)

type personResource struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type appointmentResource struct {
	ID                     string    `json:"id"`
	PersonID               string    `json:"person_id"`
	PersonName             string    `json:"person_name"`
	CustomerID             string    `json:"customer_id"`
	CustomerName           string    `json:"customer_name"`
	CustomerPhone          string    `json:"customer_phone"`
	ServiceID              string    `json:"service_id"`
	ServiceName            string    `json:"service_name"`
	ServiceDurationMinutes int       `json:"service_duration_minutes"`
	StartsAt               time.Time `json:"starts_at"`
	EndsAt                 time.Time `json:"ends_at"`
	Status                 string    `json:"status"`
	Channel                string    `json:"channel"`
	Notes                  string    `json:"notes"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type createAppointmentInput struct {
	PersonID   string    `json:"person_id"`
	CustomerID string    `json:"customer_id"`
	ServiceID  string    `json:"service_id"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	Status     string    `json:"status"`
	Channel    string    `json:"channel"`
	Notes      string    `json:"notes"`
}

type updateAppointmentInput struct {
	PersonID   *string    `json:"person_id"`
	CustomerID *string    `json:"customer_id"`
	ServiceID  *string    `json:"service_id"`
	StartsAt   *time.Time `json:"starts_at"`
	EndsAt     *time.Time `json:"ends_at"`
	Status     *string    `json:"status"`
	Notes      *string    `json:"notes"`
}

func toPersonResources(people []domain.Person) []personResource {
	out := make([]personResource, len(people))
	for i, p := range people {
		out[i] = personResource{ID: p.ID, Name: p.Name, Active: p.Active}
	}
	return out
}

func toAppointmentResource(v application.AppointmentView) appointmentResource {
	a := v.Appointment
	return appointmentResource{
		ID:                     a.ID,
		PersonID:               a.PersonID,
		PersonName:             v.PersonName,
		CustomerID:             a.CustomerID,
		CustomerName:           v.CustomerName,
		CustomerPhone:          v.CustomerPhone,
		ServiceID:              a.ServiceID,
		ServiceName:            v.ServiceName,
		ServiceDurationMinutes: v.ServiceDurationMinutes,
		StartsAt:               a.StartsAt,
		EndsAt:                 a.EndsAt,
		Status:                 string(a.Status),
		Channel:                string(a.Channel),
		Notes:                  a.Notes,
		CreatedAt:              a.CreatedAt,
		UpdatedAt:              a.UpdatedAt,
	}
}

func toAppointmentResources(views []application.AppointmentView) []appointmentResource {
	out := make([]appointmentResource, len(views))
	for i, v := range views {
		out[i] = toAppointmentResource(v)
	}
	return out
}
