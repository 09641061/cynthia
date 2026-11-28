package application

import (
	"strings"

	"github.com/google/uuid"
)

const invalidIDHint = "call list_services, get_available_slots, or list_my_appointments to obtain valid UUIDs"

func parseValidUUID(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if _, err := uuid.Parse(s); err != nil {
		return "", false
	}
	return s, true
}

func parseOptionalUUID(v *string) (*string, bool) {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil, true
	}
	id, ok := parseValidUUID(*v)
	if !ok {
		return nil, false
	}
	return &id, true
}

func marshalInvalidID() string {
	return marshalToolPayload(map[string]string{"error": "invalid_id", "hint": invalidIDHint})
}
