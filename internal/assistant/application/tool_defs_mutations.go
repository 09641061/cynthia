package application

import "go-service/internal/assistant/domain"

func mutationToolDefinitions() []domain.ToolDefinition {
	return []domain.ToolDefinition{
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "create_appointment",
				Description: "Create a confirmed appointment for the customer after receiving explicit customer confirmation in a subsequent turn.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"service_id":            map[string]any{"type": "string", "description": "Catalog service UUID."},
						"person_id":             map[string]any{"type": "string", "description": "Staff member UUID."},
						"start_at":              map[string]any{"type": "string", "description": "Start timestamp in RFC3339 format."},
						"notes":                 map[string]any{"type": "string", "description": "Optional booking notes."},
						"confirmed_by_customer": map[string]any{"type": "boolean", "description": "Must be true only after explicit confirmation from the customer."},
					},
					"required":             []string{"service_id", "person_id", "start_at", "confirmed_by_customer"},
					"additionalProperties": false,
				},
			},
		},
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "reschedule_appointment",
				Description: "Reschedule or update service/staff on one of the customer's upcoming appointments after explicit confirmation.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"appointment_id":        map[string]any{"type": "string", "description": "Upcoming appointment UUID."},
						"start_at":              map[string]any{"type": "string", "description": "Optional new RFC3339 start timestamp."},
						"service_id":            map[string]any{"type": "string", "description": "Optional new service UUID."},
						"person_id":             map[string]any{"type": "string", "description": "Optional new staff member UUID."},
						"confirmed_by_customer": map[string]any{"type": "boolean", "description": "Must be true only after explicit confirmation from the customer."},
					},
					"required":             []string{"appointment_id", "confirmed_by_customer"},
					"additionalProperties": false,
				},
			},
		},
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "cancel_appointment",
				Description: "Cancel one of the customer's upcoming appointments after explicit confirmation.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"appointment_id":        map[string]any{"type": "string", "description": "Upcoming appointment UUID."},
						"confirmed_by_customer": map[string]any{"type": "boolean", "description": "Must be true only after explicit confirmation from the customer."},
					},
					"required":             []string{"appointment_id", "confirmed_by_customer"},
					"additionalProperties": false,
				},
			},
		},
		{
			Type: "function",
			Function: domain.ToolFunction{
				Name:        "handoff_to_human",
				Description: "Pause AI on this conversation and transfer the customer to a human team member.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"reason": map[string]any{"type": "string", "description": "Short reason for transferring to a human."},
					},
					"required":             []string{"reason"},
					"additionalProperties": false,
				},
			},
		},
	}
}
