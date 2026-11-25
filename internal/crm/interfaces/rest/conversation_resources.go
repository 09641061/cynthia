package rest

import (
	"time"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type sendMessageInput struct {
	Content string `json:"content"`
}

type updateStatusInput struct {
	Status string `json:"status"`
}

type updateAIInput struct {
	AIPaused *bool `json:"ai_paused"`
}

type conversationResource struct {
	ID                 string     `json:"id"`
	CustomerID         string     `json:"customer_id"`
	Channel            string     `json:"channel"`
	Status             string     `json:"status"`
	AiPaused           bool       `json:"ai_paused"`
	HandoffReason      string     `json:"handoff_reason"`
	Version            int64      `json:"version"`
	LastMessageAt      *time.Time `json:"last_message_at,omitempty"`
	LastMessagePreview *string    `json:"last_message_preview,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type conversationPageResource struct {
	Items    []conversationResource `json:"items"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}

type messageResource struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	ExternalID     *string   `json:"external_id,omitempty"`
	IdempotencyKey *string   `json:"idempotency_key,omitempty"`
	Direction      string    `json:"direction"`
	Content        string    `json:"content"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

func toConversationResource(c domain.Conversation) conversationResource {
	return conversationResource{
		ID:                 c.ID,
		CustomerID:         c.CustomerID,
		Channel:            c.Channel,
		Status:             c.Status,
		AiPaused:           c.AiPaused,
		HandoffReason:      c.HandoffReason,
		Version:            c.Version,
		LastMessageAt:      c.LastMessageAt,
		LastMessagePreview: c.LastMessagePreview,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

func toConversationPageResource(p application.ConversationPage) conversationPageResource {
	items := make([]conversationResource, 0, len(p.Items))
	for _, c := range p.Items {
		items = append(items, toConversationResource(c))
	}
	return conversationPageResource{
		Items:    items,
		Total:    p.Total,
		Page:     p.Page,
		PageSize: p.PageSize,
	}
}

func toMessageResource(m domain.Message) messageResource {
	return messageResource{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		ExternalID:     m.ExternalID,
		IdempotencyKey: m.IdempotencyKey,
		Direction:      m.Direction,
		Content:        m.Content,
		Status:         m.Status,
		CreatedAt:      m.CreatedAt,
	}
}
