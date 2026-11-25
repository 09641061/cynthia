package acl

import (
	"context"
	"errors"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
	"go-service/internal/catalog"
)

// ServiceAdapter translates catalog service summaries into calendar's ServiceReader port.
type ServiceAdapter struct {
	reader *catalog.ServiceReader
}

// NewServiceAdapter wraps a catalog ServiceReader for calendar use cases.
func NewServiceAdapter(reader *catalog.ServiceReader) ServiceAdapter {
	return ServiceAdapter{reader: reader}
}

// GetService fetches and translates a single service summary.
func (a ServiceAdapter) GetService(ctx context.Context, id string) (application.ServiceSummary, error) {
	if a.reader == nil {
		return application.ServiceSummary{}, domain.ErrServiceNotFound
	}
	s, err := a.reader.GetService(ctx, id)
	if errors.Is(err, catalog.ErrServiceNotFound) {
		return application.ServiceSummary{}, domain.ErrServiceNotFound
	}
	if err != nil {
		return application.ServiceSummary{}, err
	}
	return application.ServiceSummary{ID: s.ID, Name: s.Name, DurationMinutes: s.DurationMinutes}, nil
}

// GetServicesByIDs batch-fetches and translates service summaries keyed by ID.
func (a ServiceAdapter) GetServicesByIDs(ctx context.Context, ids []string) (map[string]application.ServiceSummary, error) {
	if a.reader == nil {
		return map[string]application.ServiceSummary{}, nil
	}
	raw, err := a.reader.GetServicesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]application.ServiceSummary, len(raw))
	for k, v := range raw {
		out[k] = application.ServiceSummary{ID: v.ID, Name: v.Name, DurationMinutes: v.DurationMinutes}
	}
	return out, nil
}
