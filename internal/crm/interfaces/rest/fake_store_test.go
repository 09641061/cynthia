package rest_test

import (
	"context"
	"strings"
	"time"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type fakeCustomerStore struct {
	customers   map[string]domain.Customer
	blockedDel  map[string]bool
	createOrder []string
}

func newFakeCustomerStore() *fakeCustomerStore {
	return &fakeCustomerStore{
		customers:  map[string]domain.Customer{},
		blockedDel: map[string]bool{},
	}
}

func (f *fakeCustomerStore) ListCustomers(_ context.Context, query string, page, pageSize int) ([]domain.Customer, int, error) {
	rawTokens := strings.Fields(truncateRunes(query, 200))
	if len(rawTokens) > 8 {
		rawTokens = rawTokens[:8]
	}
	tokens := make([]string, len(rawTokens))
	for i, t := range rawTokens {
		tokens[i] = strings.ToLower(t)
	}
	matched := make([]domain.Customer, 0)
	for _, id := range f.createOrder {
		if c, ok := f.customers[id]; ok && (len(tokens) == 0 || customerMatchesTokens(c, tokens)) {
			matched = append(matched, c)
		}
	}
	total := len(matched)
	start := (page - 1) * pageSize
	if start >= total {
		return []domain.Customer{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return matched[start:end], total, nil
}

func truncateRunes(s string, maxRunes int) string {
	n := 0
	for i := range s {
		if n == maxRunes {
			return s[:i]
		}
		n++
	}
	return s
}

func asciiDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if '0' <= r && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func customerMatchesTokens(c domain.Customer, tokens []string) bool {
	name, phoneDigits := strings.ToLower(c.Name), asciiDigits(c.Phone)
	for _, tok := range tokens {
		if strings.Contains(name, tok) || (tok == "+" && strings.HasPrefix(c.Phone, "+")) {
			continue
		}
		if d := asciiDigits(tok); d != "" && strings.Contains(phoneDigits, d) {
			continue
		}
		return false
	}
	return true
}

func (f *fakeCustomerStore) GetCustomer(_ context.Context, id string) (domain.Customer, error) {
	c, ok := f.customers[id]
	if !ok {
		return domain.Customer{}, application.ErrNotFound
	}
	return c, nil
}

func (f *fakeCustomerStore) GetCustomerByPhone(_ context.Context, phone string) (domain.Customer, error) {
	for _, c := range f.customers {
		if c.Phone == phone {
			return c, nil
		}
	}
	return domain.Customer{}, application.ErrNotFound
}

func (f *fakeCustomerStore) CreateCustomer(_ context.Context, customer domain.Customer) (domain.Customer, error) {
	for _, existing := range f.customers {
		if existing.Phone == customer.Phone {
			return domain.Customer{}, application.DuplicatePhoneError{Existing: existing}
		}
	}
	now := time.Now().UTC()
	customer.Version = 1
	customer.CreatedAt = now
	customer.UpdatedAt = now
	f.customers[customer.ID] = customer
	f.createOrder = append(f.createOrder, customer.ID)
	return customer, nil
}

func (f *fakeCustomerStore) UpdateCustomer(_ context.Context, id string, expectedVersion int64, name, phone string) (domain.Customer, error) {
	c, ok := f.customers[id]
	if !ok {
		return domain.Customer{}, application.ErrNotFound
	}
	if c.Version != expectedVersion {
		return domain.Customer{}, application.ErrPreconditionFailed
	}
	for _, other := range f.customers {
		if other.ID != id && other.Phone == phone {
			return domain.Customer{}, application.DuplicatePhoneError{Existing: other}
		}
	}
	c.Name = name
	c.Phone = phone
	c.Version++
	c.UpdatedAt = time.Now().UTC()
	f.customers[id] = c
	return c, nil
}

func (f *fakeCustomerStore) DeleteCustomer(_ context.Context, id string) error {
	if _, ok := f.customers[id]; !ok {
		return application.ErrNotFound
	}
	if f.blockedDel[id] {
		return application.ErrConflict
	}
	delete(f.customers, id)
	return nil
}
