package application

import (
	"time"

	"go-service/internal/assistant/domain"
)

// Job represents a queued or processing assistant job.
type Job struct {
	ID               string
	ConversationID   string
	TriggerMessageID string
	Status           string
	RunAfter         time.Time
	Attempts         int
	MaxAttempts      int
	ReplyText        string
	LastError        string
}

// OrgConfig holds organization settings required by the assistant processor.
type OrgConfig struct {
	AiEnabled       bool
	AiInstructions  string
	Timezone        string
	CurrencyCode    string
	BusinessHours   []domain.DayHours
	WhatsAppEnabled bool
}

// Conversation is the assistant's view of a CRM conversation.
type Conversation struct {
	ID            string
	CustomerID    string
	Status        string
	AiPaused      bool
	HandoffReason string
}

// Message is the assistant's view of a CRM message.
type Message struct {
	ID        string
	Direction string
	Content   string
	Status    string
	CreatedAt time.Time
}

// ServiceItem is a catalog service entry exposed to assistant tools.
type ServiceItem struct {
	ID              string
	CategoryName    string
	Name            string
	DurationMinutes int
	PriceMinorUnits int64
}

// AvailabilityQuery specifies single-day availability search parameters.
type AvailabilityQuery struct {
	Date                 string
	ServiceID            string
	PersonID             string
	ExcludeAppointmentID string
	CustomerID           string
}

// AvailabilitySlot is a single open booking slot.
type AvailabilitySlot struct {
	StartAt    time.Time
	EndAt      time.Time
	PersonID   string
	PersonName string
}

// UpcomingAppointment is a customer-owned upcoming appointment.
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

// AppointmentResult is the outcome of a booking mutation.
type AppointmentResult struct {
	ID        string
	ServiceID string
	PersonID  string
	StartAt   time.Time
	EndAt     time.Time
	Status    string
}

// CreateAppointmentInput holds inputs for creating a WhatsApp appointment.
type CreateAppointmentInput struct {
	CustomerID string
	ServiceID  string
	PersonID   string
	StartAt    time.Time
	Notes      string
}

// RescheduleAppointmentInput holds inputs for rescheduling a WhatsApp appointment.
type RescheduleAppointmentInput struct {
	AppointmentID string
	CustomerID    string
	StartAt       *time.Time
	ServiceID     *string
	PersonID      *string
}
