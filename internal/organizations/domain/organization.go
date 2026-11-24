package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"
)

// ErrInvalid indicates an organization value violates business rules.
var ErrInvalid = errors.New("invalid organization data")

// InvalidFieldError identifies an organization field that violates a constraint.
type InvalidFieldError struct {
	Field  string
	Reason string
}

// Error returns a safe field-specific validation message.
func (e InvalidFieldError) Error() string { return e.Field + ": " + e.Reason }

// Unwrap identifies the general organization validation failure.
func (e InvalidFieldError) Unwrap() error { return ErrInvalid }

// DayHours represents opening status and times for a single weekday.
type DayHours struct {
	IsOpen    bool   `json:"is_open"`
	OpenTime  string `json:"open_time"`
	CloseTime string `json:"close_time"`
}

// BusinessHours represents the weekly schedule of the organization.
type BusinessHours struct {
	Monday    DayHours `json:"monday"`
	Tuesday   DayHours `json:"tuesday"`
	Wednesday DayHours `json:"wednesday"`
	Thursday  DayHours `json:"thursday"`
	Friday    DayHours `json:"friday"`
	Saturday  DayHours `json:"saturday"`
	Sunday    DayHours `json:"sunday"`
}

// Organization holds the single-tenant business profile and assistant settings.
type Organization struct {
	ID             string        `json:"id"`
	Name           string        `json:"name"`
	BusinessType   string        `json:"business_type"`
	ContactEmail   string        `json:"contact_email"`
	Timezone       string        `json:"timezone"`
	BusinessHours  BusinessHours `json:"business_hours"`
	AiInstructions string        `json:"ai_instructions"`
	Version        int64         `json:"version"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

// UpdatePatch carries partial updates to the organization profile.
type UpdatePatch struct {
	Name           *string        `json:"name"`
	BusinessType   *string        `json:"business_type"`
	ContactEmail   *string        `json:"contact_email"`
	Timezone       *string        `json:"timezone"`
	BusinessHours  *BusinessHours `json:"business_hours"`
	AiInstructions *string        `json:"ai_instructions"`
}

// ApplyPatch applies and validates a partial update on the current organization.
func ApplyPatch(current Organization, patch UpdatePatch) (Organization, error) {
	next := current
	if patch.Name != nil {
		next.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.BusinessType != nil {
		next.BusinessType = strings.TrimSpace(*patch.BusinessType)
	}
	if patch.ContactEmail != nil {
		next.ContactEmail = strings.TrimSpace(*patch.ContactEmail)
	}
	if patch.Timezone != nil {
		next.Timezone = strings.TrimSpace(*patch.Timezone)
	}
	if patch.BusinessHours != nil {
		next.BusinessHours = *patch.BusinessHours
	}
	if patch.AiInstructions != nil {
		next.AiInstructions = *patch.AiInstructions
	}
	if err := validateOrganization(next); err != nil {
		return Organization{}, err
	}
	return next, nil
}

func validateOrganization(org Organization) error {
	nameLen := len([]rune(org.Name))
	if nameLen < 2 || nameLen > 120 {
		return InvalidFieldError{Field: "name", Reason: "must be between 2 and 120 characters"}
	}
	if len([]rune(org.BusinessType)) > 120 {
		return InvalidFieldError{Field: "business_type", Reason: "must not exceed 120 characters"}
	}
	if org.ContactEmail != "" {
		emailLen := len([]rune(org.ContactEmail))
		if emailLen < 5 || emailLen > 254 {
			return InvalidFieldError{Field: "contact_email", Reason: "must be between 5 and 254 characters"}
		}
		addr, err := mail.ParseAddress(org.ContactEmail)
		if err != nil || addr.Address != org.ContactEmail {
			return InvalidFieldError{Field: "contact_email", Reason: "must be a valid email address"}
		}
	}
	if org.Timezone == "" {
		return InvalidFieldError{Field: "timezone", Reason: "must not be blank"}
	}
	if _, err := time.LoadLocation(org.Timezone); err != nil {
		return InvalidFieldError{Field: "timezone", Reason: "must be a valid IANA timezone"}
	}
	if len([]rune(org.AiInstructions)) > 4000 {
		return InvalidFieldError{Field: "ai_instructions", Reason: "must not exceed 4000 characters"}
	}
	return validateBusinessHours(org.BusinessHours)
}

func validateBusinessHours(hours BusinessHours) error {
	days := []struct {
		name string
		day  DayHours
	}{
		{"monday", hours.Monday}, {"tuesday", hours.Tuesday}, {"wednesday", hours.Wednesday},
		{"thursday", hours.Thursday}, {"friday", hours.Friday}, {"saturday", hours.Saturday},
		{"sunday", hours.Sunday},
	}
	for _, entry := range days {
		if !validHHMM(entry.day.OpenTime) || !validHHMM(entry.day.CloseTime) {
			return InvalidFieldError{Field: "business_hours." + entry.name, Reason: "times must use HH:MM format"}
		}
		if entry.day.IsOpen && entry.day.OpenTime >= entry.day.CloseTime {
			return InvalidFieldError{Field: "business_hours." + entry.name, Reason: "open_time must be before close_time"}
		}
	}
	return nil
}

func validHHMM(value string) bool {
	if len(value) != 5 || value[2] != ':' {
		return false
	}
	_, err := time.Parse("15:04", value)
	return err == nil
}
