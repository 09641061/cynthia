package domain

import (
	"errors"
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
	ID                    string        `json:"id"`
	Name                  string        `json:"name"`
	BusinessType          string        `json:"business_type"`
	ContactEmail          string        `json:"contact_email"`
	Timezone              string        `json:"timezone"`
	BusinessHours         BusinessHours `json:"business_hours"`
	AiInstructions        string        `json:"ai_instructions"`
	WhatsAppPhoneNumber   string        `json:"whatsapp_phone_number"`
	WhatsAppPhoneNumberID string        `json:"whatsapp_phone_number_id"`
	WhatsAppEnabled       bool          `json:"whatsapp_enabled"`
	Version               int64         `json:"version"`
	CreatedAt             time.Time     `json:"created_at"`
	UpdatedAt             time.Time     `json:"updated_at"`
}

// UpdatePatch carries partial updates to the organization profile.
type UpdatePatch struct {
	Name                  *string        `json:"name"`
	BusinessType          *string        `json:"business_type"`
	ContactEmail          *string        `json:"contact_email"`
	Timezone              *string        `json:"timezone"`
	BusinessHours         *BusinessHours `json:"business_hours"`
	AiInstructions        *string        `json:"ai_instructions"`
	WhatsAppPhoneNumber   *string        `json:"whatsapp_phone_number"`
	WhatsAppPhoneNumberID *string        `json:"whatsapp_phone_number_id"`
	WhatsAppEnabled       *bool          `json:"whatsapp_enabled"`
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
	if patch.WhatsAppPhoneNumber != nil {
		next.WhatsAppPhoneNumber = stripPhoneFormatting(strings.TrimSpace(*patch.WhatsAppPhoneNumber))
	}
	if patch.WhatsAppPhoneNumberID != nil {
		next.WhatsAppPhoneNumberID = strings.TrimSpace(*patch.WhatsAppPhoneNumberID)
	}
	if patch.WhatsAppEnabled != nil {
		next.WhatsAppEnabled = *patch.WhatsAppEnabled
	}
	if err := validateOrganization(next); err != nil {
		return Organization{}, err
	}
	return next, nil
}
