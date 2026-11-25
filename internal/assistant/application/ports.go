package application

import (
	"context"
	"errors"
	"time"

	"go-service/internal/assistant/domain"
)

// Sentinel errors translated from CRM and Calendar ACLs.
var (
	ErrDeliveryFailed       = errors.New("assistant delivery failed")
	ErrDeliveryUncertain    = errors.New("assistant delivery uncertain")
	ErrConversationClosed   = errors.New("conversation closed")
	ErrSlotConflict         = errors.New("slot conflict")
	ErrOutsideBusinessHours = errors.New("outside business hours")
	ErrSlotInPast           = errors.New("slot in past")
	ErrServiceNotFound      = errors.New("service not found")
	ErrPersonNotFound       = errors.New("person not found")
	ErrAppointmentNotFound  = errors.New("appointment not found")
	ErrNoFieldsToUpdate     = errors.New("no fields to update")
)

// RetryableError is implemented by infrastructure errors that classify retryability.
type RetryableError interface {
	IsRetryable() bool
}

// IsRetryableError reports whether err implements RetryableError and returns true.
func IsRetryableError(err error) bool {
	var re RetryableError
	return errors.As(err, &re) && re.IsRetryable()
}

// JobStore persists and coordinates assistant background jobs.
type JobStore interface {
	Enqueue(ctx context.Context, convID, triggerMsgID string, debounce time.Duration) error
	ClaimNext(ctx context.Context, lease time.Duration) (Job, bool, error)
	HasPendingForConversation(ctx context.Context, convID string) (bool, error)
	CountRecentCompleted(ctx context.Context, convID string, since time.Time) (int, error)
	SaveReply(ctx context.Context, jobID, replyText string) error
	Complete(ctx context.Context, jobID string) error
	FailOrRetry(ctx context.Context, jobID, convID, sanitizedErr string, retryable bool, backoff time.Duration) error
}

// LLMClient executes chat completion requests.
type LLMClient interface {
	Complete(ctx context.Context, messages []domain.ChatMessage, tools []domain.ToolDefinition) (domain.CompletionResult, error)
}

// OrgReader loads organization assistant configuration.
type OrgReader interface {
	GetAssistantConfig(ctx context.Context) (OrgConfig, error)
}

// CRMGateway accesses conversations, messages, replies, and handoff state.
type CRMGateway interface {
	GetConversation(ctx context.Context, convID string) (Conversation, error)
	ListRecentMessages(ctx context.Context, convID string, limit int) ([]Message, error)
	SendAssistantReply(ctx context.Context, convID, idempotencyKey, content string) (Message, error)
	PauseForHandoff(ctx context.Context, convID, reason string) error
}

// CatalogReader lists available services.
type CatalogReader interface {
	ListCatalogServices(ctx context.Context) ([]ServiceItem, error)
}

// CalendarGateway queries availability and mutates WhatsApp appointments.
type CalendarGateway interface {
	ListAvailability(ctx context.Context, query AvailabilityQuery) ([]AvailabilitySlot, error)
	ListCustomerUpcomingAppointments(ctx context.Context, customerID string) ([]UpcomingAppointment, error)
	CreateWhatsAppAppointment(ctx context.Context, in CreateAppointmentInput) (AppointmentResult, error)
	RescheduleWhatsAppAppointment(ctx context.Context, in RescheduleAppointmentInput) (AppointmentResult, error)
	CancelWhatsAppAppointment(ctx context.Context, appointmentID, customerID string) (AppointmentResult, error)
}
