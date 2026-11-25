package domain

// Status represents the lifecycle state of an appointment.
type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusCompleted Status = "completed"
	StatusCancelled Status = "cancelled"
	StatusNoShow    Status = "no_show"
)

// Channel represents the origin channel of an appointment.
type Channel string

const (
	ChannelManual   Channel = "manual"
	ChannelWhatsApp Channel = "whatsapp"
)

// Person represents a bookable staff member in the calendar context.
type Person struct {
	ID     string
	Name   string
	Active bool
}

// ParseStatus validates and converts a raw string into a Status.
func ParseStatus(raw string) (Status, error) {
	switch Status(raw) {
	case StatusPending, StatusConfirmed, StatusCompleted, StatusCancelled, StatusNoShow:
		return Status(raw), nil
	default:
		return "", InvalidFieldError{Field: "status", Reason: "must be pending, confirmed, completed, cancelled, or no_show"}
	}
}

// ParseChannel validates and converts a raw string into a Channel.
func ParseChannel(raw string) (Channel, error) {
	switch Channel(raw) {
	case ChannelManual, ChannelWhatsApp:
		return Channel(raw), nil
	default:
		return "", InvalidFieldError{Field: "channel", Reason: "must be manual or whatsapp"}
	}
}

// IsBlocking reports whether the status reserves the person's time slot.
func (s Status) IsBlocking() bool {
	return s == StatusPending || s == StatusConfirmed
}

// CanTransition reports whether transitioning from one status to another is permitted.
func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}
	switch from {
	case StatusPending:
		return to == StatusConfirmed || to == StatusCancelled || to == StatusCompleted || to == StatusNoShow
	case StatusConfirmed:
		return to == StatusCompleted || to == StatusCancelled || to == StatusNoShow
	default:
		return false
	}
}
