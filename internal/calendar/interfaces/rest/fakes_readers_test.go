package rest_test

import (
	"context"

	"go-service/internal/calendar/application"
	"go-service/internal/calendar/domain"
)

type memPeople struct{ people map[string]domain.Person }

func (m memPeople) ListActivePeople(_ context.Context) ([]domain.Person, error) {
	out := make([]domain.Person, 0, len(m.people))
	for _, p := range m.people {
		out = append(out, p)
	}
	return out, nil
}

func (m memPeople) GetPerson(_ context.Context, id string) (domain.Person, error) {
	p, ok := m.people[id]
	if !ok {
		return domain.Person{}, domain.ErrPersonNotFound
	}
	return p, nil
}

func (m memPeople) GetPeopleByIDs(_ context.Context, _ []string) (map[string]domain.Person, error) {
	return m.people, nil
}

type memCustomers struct {
	customers map[string]application.CustomerSummary
}

func (m memCustomers) GetCustomer(_ context.Context, id string) (application.CustomerSummary, error) {
	c, ok := m.customers[id]
	if !ok {
		return application.CustomerSummary{}, domain.ErrCustomerNotFound
	}
	return c, nil
}

func (m memCustomers) GetCustomersByIDs(_ context.Context, _ []string) (map[string]application.CustomerSummary, error) {
	return m.customers, nil
}

type memServices struct {
	services map[string]application.ServiceSummary
}

func (m memServices) GetService(_ context.Context, id string) (application.ServiceSummary, error) {
	s, ok := m.services[id]
	if !ok {
		return application.ServiceSummary{}, domain.ErrServiceNotFound
	}
	return s, nil
}

func (m memServices) GetServicesByIDs(_ context.Context, _ []string) (map[string]application.ServiceSummary, error) {
	return m.services, nil
}
