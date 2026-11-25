package calendar

import (
	"errors"
	"time"
)

var (
	// ErrSlotConflict indicates the requested slot overlaps an existing blocking appointment.
	ErrSlotConflict = errors.New("slot conflict")
	// ErrOutsideBusinessHours indicates the requested interval falls outside organization business hours.
	ErrOutsideBusinessHours = errors.New("outside business hours")
	// ErrServiceNotFound indicates the referenced catalog service does not exist.
	ErrServiceNotFound = errors.New("service not found")
	// ErrPersonNotFound indicates the referenced active staff member does not exist.
	ErrPersonNotFound = errors.New("person not found")
	// ErrAppointmentNotFound indicates no matching upcoming mutable appointment was found for the customer.
	ErrAppointmentNotFound = errors.New("appointment not found")
	// ErrNoFieldsToUpdate indicates a reschedule request provided no fields to change.
	ErrNoFieldsToUpdate = errors.New("no fields to update")
	// ErrGatewayUnavailable indicates the calendar gateway has no configured database.
	ErrGatewayUnavailable = errors.New("calendar gateway unavailable")
)

// UpcomingAppointment is a customer-scoped upcoming appointment projection for the assistant.
type UpcomingAppointment struct {
	ID          string
	ServiceID   string
	ServiceName string
	PersonID    string
	PersonName  string
	StartAt     time.Time
	EndAt       time.Time
	Status      string
}

// AssistantAppointment is the mutation result returned by the assistant calendar gateway.
type AssistantAppointment struct {
	ID        string
	ServiceID string
	PersonID  string
	StartAt   time.Time
	EndAt     time.Time
	Status    string
}

// CreateWhatsAppAppointmentInput holds fields for creating a WhatsApp-originated appointment.
type CreateWhatsAppAppointmentInput struct {
	CustomerID string
	ServiceID  string
	PersonID   string
	StartAt    time.Time
	Notes      string
}

// RescheduleWhatsAppAppointmentInput holds fields for rescheduling a customer's upcoming appointment.
type RescheduleWhatsAppAppointmentInput struct {
	AppointmentID string
	CustomerID    string
	StartAt       *time.Time
	ServiceID     *string
	PersonID      *string
}
