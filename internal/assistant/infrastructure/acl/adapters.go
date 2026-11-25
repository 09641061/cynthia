package acl

import (
	"context"
	"errors"
	"fmt"

	"go-service/internal/assistant/application"
	"go-service/internal/assistant/domain"
	"go-service/internal/catalog"
	"go-service/internal/crm"
	"go-service/internal/organizations"
)

// OrgAdapter adapts organizations.AssistantConfigReader to application.OrgReader.
type OrgAdapter struct {
	reader *organizations.AssistantConfigReader
}

// NewOrgAdapter creates an OrgAdapter.
func NewOrgAdapter(reader *organizations.AssistantConfigReader) *OrgAdapter {
	return &OrgAdapter{reader: reader}
}

// GetAssistantConfig loads and maps organization assistant configuration.
func (a *OrgAdapter) GetAssistantConfig(ctx context.Context) (application.OrgConfig, error) {
	if a == nil || a.reader == nil {
		return application.OrgConfig{}, nil
	}
	cfg, err := a.reader.GetAssistantConfig(ctx)
	if err != nil {
		return application.OrgConfig{}, err
	}
	bh := cfg.BusinessHours
	return application.OrgConfig{
		AiEnabled:       cfg.AiEnabled,
		AiInstructions:  cfg.AiInstructions,
		Timezone:        cfg.Timezone,
		CurrencyCode:    cfg.CurrencyCode,
		WhatsAppEnabled: cfg.WhatsAppEnabled,
		BusinessHours: []domain.DayHours{
			{Day: "monday", IsOpen: bh.Monday.IsOpen, OpenTime: bh.Monday.OpenTime, CloseTime: bh.Monday.CloseTime},
			{Day: "tuesday", IsOpen: bh.Tuesday.IsOpen, OpenTime: bh.Tuesday.OpenTime, CloseTime: bh.Tuesday.CloseTime},
			{Day: "wednesday", IsOpen: bh.Wednesday.IsOpen, OpenTime: bh.Wednesday.OpenTime, CloseTime: bh.Wednesday.CloseTime},
			{Day: "thursday", IsOpen: bh.Thursday.IsOpen, OpenTime: bh.Thursday.OpenTime, CloseTime: bh.Thursday.CloseTime},
			{Day: "friday", IsOpen: bh.Friday.IsOpen, OpenTime: bh.Friday.OpenTime, CloseTime: bh.Friday.CloseTime},
			{Day: "saturday", IsOpen: bh.Saturday.IsOpen, OpenTime: bh.Saturday.OpenTime, CloseTime: bh.Saturday.CloseTime},
			{Day: "sunday", IsOpen: bh.Sunday.IsOpen, OpenTime: bh.Sunday.OpenTime, CloseTime: bh.Sunday.CloseTime},
		},
	}, nil
}

// CatalogAdapter adapts catalog.ServiceReader to application.CatalogReader.
type CatalogAdapter struct {
	reader *catalog.ServiceReader
}

// NewCatalogAdapter creates a CatalogAdapter.
func NewCatalogAdapter(reader *catalog.ServiceReader) *CatalogAdapter {
	return &CatalogAdapter{reader: reader}
}

// ListCatalogServices maps catalog services to application.ServiceItem.
func (a *CatalogAdapter) ListCatalogServices(ctx context.Context) ([]application.ServiceItem, error) {
	if a == nil || a.reader == nil {
		return []application.ServiceItem{}, nil
	}
	items, err := a.reader.ListCatalogServices(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]application.ServiceItem, 0, len(items))
	for _, it := range items {
		out = append(out, application.ServiceItem{
			ID: it.ID, CategoryName: it.CategoryName, Name: it.Name,
			DurationMinutes: it.DurationMinutes, PriceMinorUnits: it.PriceMinorUnits,
		})
	}
	return out, nil
}

// CRMAdapter adapts crm.AssistantCRMGateway to application.CRMGateway.
type CRMAdapter struct {
	gw *crm.AssistantCRMGateway
}

// NewCRMAdapter creates a CRMAdapter.
func NewCRMAdapter(gw *crm.AssistantCRMGateway) *CRMAdapter {
	return &CRMAdapter{gw: gw}
}

func (a *CRMAdapter) GetConversation(ctx context.Context, convID string) (application.Conversation, error) {
	c, err := a.gw.GetConversation(ctx, convID)
	if err != nil {
		return application.Conversation{}, mapCRMErr(err)
	}
	return application.Conversation{
		ID: c.ID, CustomerID: c.CustomerID, Status: c.Status,
		AiPaused: c.AiPaused, HandoffReason: c.HandoffReason,
	}, nil
}

func (a *CRMAdapter) ListRecentMessages(ctx context.Context, convID string, limit int) ([]application.Message, error) {
	msgs, err := a.gw.ListRecentMessages(ctx, convID, limit)
	if err != nil {
		return nil, mapCRMErr(err)
	}
	out := make([]application.Message, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, application.Message{
			ID: m.ID, Direction: m.Direction, Content: m.Content, Status: m.Status, CreatedAt: m.CreatedAt,
		})
	}
	return out, nil
}

func (a *CRMAdapter) SendAssistantReply(ctx context.Context, convID, idempotencyKey, content string) (application.Message, error) {
	m, err := a.gw.SendAssistantReply(ctx, convID, idempotencyKey, content)
	if err != nil {
		return application.Message{}, mapCRMErr(err)
	}
	return application.Message{
		ID: m.ID, Direction: m.Direction, Content: m.Content, Status: m.Status, CreatedAt: m.CreatedAt,
	}, nil
}

func (a *CRMAdapter) PauseForHandoff(ctx context.Context, convID, reason string) error {
	return mapCRMErr(a.gw.PauseForHandoff(ctx, convID, reason))
}

func mapCRMErr(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, crm.ErrConversationClosed):
		return fmt.Errorf("%w: %w", application.ErrConversationClosed, err)
	case errors.Is(err, crm.ErrDeliveryFailed):
		return fmt.Errorf("%w: %w", application.ErrDeliveryFailed, err)
	case errors.Is(err, crm.ErrDeliveryUncertain):
		return fmt.Errorf("%w: %w", application.ErrDeliveryUncertain, err)
	default:
		return err
	}
}
