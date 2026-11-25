package application

import (
	"context"
	"strings"
	"time"

	"go-service/internal/crm/domain"
)

const (
	defaultMessageLimit = 50
	maxMessageLimit     = 200
)

// ConversationService orchestrates conversation listing, messages listing, and status changes.
type ConversationService struct {
	conversations ConversationStore
	messages      MessageStore
	events        EventPublisher
}

// NewConversationService creates a ConversationService backed by the supplied stores.
func NewConversationService(conversations ConversationStore, messages MessageStore) *ConversationService {
	return &ConversationService{conversations: conversations, messages: messages}
}

// SetEventPublisher configures an optional publisher for real-time CRM events.
func (s *ConversationService) SetEventPublisher(publisher EventPublisher) {
	s.events = publisher
}

// List returns a paginated list of conversations matching the filter.
func (s *ConversationService) List(ctx context.Context, filter ConversationFilter) (ConversationPage, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 {
		filter.PageSize = defaultPageSize
	}
	if filter.PageSize > maxPageSize {
		filter.PageSize = maxPageSize
	}
	filter.Query = strings.TrimSpace(filter.Query)
	filter.CustomerID = strings.TrimSpace(filter.CustomerID)
	status := strings.TrimSpace(filter.Status)
	if status == "" || status == "all" {
		filter.Status = ""
	} else {
		validated, err := domain.ValidateConversationStatus(status)
		if err != nil {
			return ConversationPage{}, err
		}
		filter.Status = validated
	}
	items, total, err := s.conversations.ListConversations(ctx, filter)
	if err != nil {
		return ConversationPage{}, err
	}
	return ConversationPage{Items: items, Total: total, Page: filter.Page, PageSize: filter.PageSize}, nil
}

// ListMessages returns messages for a conversation ordered chronologically.
func (s *ConversationService) ListMessages(ctx context.Context, convID string, limit int, before *time.Time) ([]domain.Message, error) {
	if _, err := s.conversations.GetConversation(ctx, convID); err != nil {
		return nil, err
	}
	if limit < 1 {
		limit = defaultMessageLimit
	}
	if limit > maxMessageLimit {
		limit = maxMessageLimit
	}
	return s.messages.ListMessages(ctx, convID, limit, before)
}

// UpdateStatus transitions a conversation between open and closed.
func (s *ConversationService) UpdateStatus(ctx context.Context, convID, rawStatus string) (domain.Conversation, error) {
	status, err := domain.ValidateConversationStatus(rawStatus)
	if err != nil {
		return domain.Conversation{}, err
	}
	conv, err := s.conversations.UpdateConversationStatus(ctx, convID, status)
	if err != nil {
		return domain.Conversation{}, err
	}
	if s.events != nil {
		s.events.Publish(Event{Type: EventConversationUpdated, ConversationID: conv.ID})
	}
	return conv, nil
}

// UpdateAI toggles AI pause state for an open conversation.
func (s *ConversationService) UpdateAI(ctx context.Context, convID string, aiPaused bool) (domain.Conversation, error) {
	current, err := s.conversations.GetConversation(ctx, convID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if current.Status == domain.ConversationStatusClosed {
		return domain.Conversation{}, ErrConflict
	}
	reason := ""
	if aiPaused {
		reason = current.HandoffReason
		if reason == "" {
			reason = "manual_pause"
		}
	}
	conv, err := s.conversations.UpdateConversationAI(ctx, convID, aiPaused, reason)
	if err != nil {
		return domain.Conversation{}, err
	}
	if s.events != nil {
		s.events.Publish(Event{Type: EventConversationUpdated, ConversationID: conv.ID})
	}
	return conv, nil
}
