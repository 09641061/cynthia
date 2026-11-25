package domain

import (
	"net/mail"
	"regexp"
	"strings"
	"time"
)

var (
	e164Pattern           = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)
	phoneNumberIDPattern  = regexp.MustCompile(`^[0-9]{1,64}$`)
	whatsappPhoneReplacer = strings.NewReplacer(" ", "", "-", "", "(", "", ")", "")
)

func stripPhoneFormatting(raw string) string {
	return whatsappPhoneReplacer.Replace(raw)
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
	if err := validateWhatsApp(org); err != nil {
		return err
	}
	return validateBusinessHours(org.BusinessHours)
}

func validateWhatsApp(org Organization) error {
	if org.WhatsAppPhoneNumber != "" && !e164Pattern.MatchString(org.WhatsAppPhoneNumber) {
		return InvalidFieldError{Field: "whatsapp_phone_number", Reason: "must be in E.164 format (+ followed by 7 to 15 digits)"}
	}
	if org.WhatsAppPhoneNumberID != "" && !phoneNumberIDPattern.MatchString(org.WhatsAppPhoneNumberID) {
		return InvalidFieldError{Field: "whatsapp_phone_number_id", Reason: "must contain only digits (1 to 64 characters)"}
	}
	if org.WhatsAppEnabled {
		if org.WhatsAppPhoneNumber == "" {
			return InvalidFieldError{Field: "whatsapp_phone_number", Reason: "is required when WhatsApp is enabled"}
		}
		if org.WhatsAppPhoneNumberID == "" {
			return InvalidFieldError{Field: "whatsapp_phone_number_id", Reason: "is required when WhatsApp is enabled"}
		}
	}
	return nil
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
