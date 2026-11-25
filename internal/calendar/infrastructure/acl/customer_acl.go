package acl

import (
	"context"
	"errors"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
	"go-service/internal/crm"
)

// CustomerAdapter translates CRM customer summaries into calendar's CustomerReader port.
type CustomerAdapter struct {
	reader *crm.CustomerReader
}

// NewCustomerAdapter wraps a CRM CustomerReader for calendar use cases.
func NewCustomerAdapter(reader *crm.CustomerReader) CustomerAdapter {
	return CustomerAdapter{reader: reader}
}

// GetCustomer fetches and translates a single customer summary.
func (a CustomerAdapter) GetCustomer(ctx context.Context, id string) (application.CustomerSummary, error) {
	if a.reader == nil {
		return application.CustomerSummary{}, domain.ErrCustomerNotFound
	}
	c, err := a.reader.GetCustomer(ctx, id)
	if errors.Is(err, crm.ErrCustomerNotFound) {
		return application.CustomerSummary{}, domain.ErrCustomerNotFound
	}
	if err != nil {
		return application.CustomerSummary{}, err
	}
	return application.CustomerSummary{ID: c.ID, Name: c.Name, Phone: c.Phone}, nil
}

// GetCustomersByIDs batch-fetches and translates customer summaries keyed by ID.
func (a CustomerAdapter) GetCustomersByIDs(ctx context.Context, ids []string) (map[string]application.CustomerSummary, error) {
	if a.reader == nil {
		return map[string]application.CustomerSummary{}, nil
	}
	raw, err := a.reader.GetCustomersByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]application.CustomerSummary, len(raw))
	for k, v := range raw {
		out[k] = application.CustomerSummary{ID: v.ID, Name: v.Name, Phone: v.Phone}
	}
	return out, nil
}
