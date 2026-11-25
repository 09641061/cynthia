package application

import "go-service/internal/assistant/domain"

// AllToolDefinitions returns the 7 DeepSeek function tool schemas exposed to the assistant.
func AllToolDefinitions() []domain.ToolDefinition {
	return append(readToolDefinitions(), mutationToolDefinitions()...)
}

func readToolDefinitions() []domain.ToolDefinition {
	return []domain.ToolDefinition{
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "list_services",
				Description: "List all available business services with their ID, category, duration in minutes, and price in minor units.",
				Parameters: map[string]any{
					"type":                 "object",
					"properties":           map[string]any{},
					"additionalProperties": false,
				},
			},
		},
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "get_available_slots",
				Description: "Find open 30-minute appointment slots for a given service on a single date (YYYY-MM-DD).",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"date":                   map[string]any{"type": "string", "description": "Date in YYYY-MM-DD format."},
						"service_id":             map[string]any{"type": "string", "description": "Catalog service UUID."},
						"person_id":              map[string]any{"type": "string", "description": "Optional staff member UUID."},
						"exclude_appointment_id": map[string]any{"type": "string", "description": "Optional appointment UUID to exclude when rescheduling."},
					},
					"required":             []string{"date", "service_id"},
					"additionalProperties": false,
				},
			},
		},
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "list_my_appointments",
				Description: "List the current customer's upcoming pending or confirmed appointments.",
				Parameters: map[string]any{
					"type":                 "object",
					"properties":           map[string]any{},
					"additionalProperties": false,
				},
			},
		},
	}
}
