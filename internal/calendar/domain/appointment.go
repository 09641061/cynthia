package domain

import (
	"strings"
	"time"
)

// Appointment represents a scheduled booking for a person, customer, and service.
type Appointment struct {
	ID         string
	PersonID   string
	CustomerID string
	ServiceID  string
	StartsAt   time.Time
	EndsAt     time.Time
	Status     Status
	Channel    Channel
	Notes      string
	Version    int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// CreateInput holds parameters to construct a new Appointment.
type CreateInput struct {
	ID         string
	PersonID   string
	CustomerID string
	ServiceID  string
	StartsAt   time.Time
	EndsAt     time.Time
	Status     Status
	Channel    Channel
	Notes      string
}

// Patch holds optional fields to update on an existing Appointment.
type Patch struct {
	PersonID   *string
	CustomerID *string
	ServiceID  *string
	StartsAt   *time.Time
	EndsAt     *time.Time
	Status     *Status
	Notes      *string
}

// NewAppointment validates inputs and constructs a new Appointment in an initial state.
func NewAppointment(in CreateInput) (Appointment, error) {
	for _, item := range []struct{ field, value string }{
		{"id", in.ID}, {"person_id", in.PersonID}, {"customer_id", in.CustomerID}, {"service_id", in.ServiceID},
	} {
		if strings.TrimSpace(item.value) == "" {
			return Appointment{}, InvalidFieldError{Field: item.field, Reason: "must not be blank"}
		}
	}
	if err := validateInterval(in.StartsAt, in.EndsAt); err != nil {
		return Appointment{}, err
	}
	status := in.Status
	if status == "" {
		status = StatusPending
	} else if status != StatusPending && status != StatusConfirmed {
		return Appointment{}, InvalidFieldError{Field: "status", Reason: "new appointments must start as pending or confirmed"}
	}
	channel := in.Channel
	if channel == "" {
		channel = ChannelManual
	} else if _, err := ParseChannel(string(channel)); err != nil {
		return Appointment{}, err
	}
	if len([]rune(in.Notes)) > 1000 {
		return Appointment{}, InvalidFieldError{Field: "notes", Reason: "must not exceed 1000 characters"}
	}
	return Appointment{
		ID: in.ID, PersonID: in.PersonID, CustomerID: in.CustomerID, ServiceID: in.ServiceID,
		StartsAt: in.StartsAt.UTC(), EndsAt: in.EndsAt.UTC(),
		Status: status, Channel: channel, Notes: in.Notes, Version: 1,
	}, nil
}

// ApplyPatch validates and applies partial updates, returning a new Appointment on success.
func (a Appointment) ApplyPatch(patch Patch) (Appointment, error) {
	next := a
	if err := applyNonBlank(&next.PersonID, patch.PersonID, "person_id"); err != nil {
		return a, err
	}
	if err := applyNonBlank(&next.CustomerID, patch.CustomerID, "customer_id"); err != nil {
		return a, err
	}
	if err := applyNonBlank(&next.ServiceID, patch.ServiceID, "service_id"); err != nil {
		return a, err
	}
	if patch.StartsAt != nil {
		next.StartsAt = patch.StartsAt.UTC()
	}
	if patch.EndsAt != nil {
		next.EndsAt = patch.EndsAt.UTC()
	}
	if err := validateInterval(next.StartsAt, next.EndsAt); err != nil {
		return a, err
	}
	if patch.Status != nil {
		parsed, err := ParseStatus(string(*patch.Status))
		if err != nil {
			return a, err
		}
		if !CanTransition(a.Status, parsed) {
			return a, InvalidFieldError{Field: "status", Reason: "transition not allowed"}
		}
		next.Status = parsed
	}
	if patch.Notes != nil {
		if len([]rune(*patch.Notes)) > 1000 {
			return a, InvalidFieldError{Field: "notes", Reason: "must not exceed 1000 characters"}
		}
		next.Notes = *patch.Notes
	}
	return next, nil
}

func applyNonBlank(dst, src *string, field string) error {
	if src == nil {
		return nil
	}
	if strings.TrimSpace(*src) == "" {
		return InvalidFieldError{Field: field, Reason: "must not be blank"}
	}
	*dst = *src
	return nil
}

func validateInterval(startsAt, endsAt time.Time) error {
	if startsAt.IsZero() {
		return InvalidFieldError{Field: "starts_at", Reason: "must be provided"}
	}
	if endsAt.IsZero() {
		return InvalidFieldError{Field: "ends_at", Reason: "must be provided"}
	}
	if !endsAt.After(startsAt) {
		return InvalidFieldError{Field: "ends_at", Reason: "must be after starts_at"}
	}
	return nil
}
