package application

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"go-service/internal/crm/domain"
)

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 100
)

// CustomerPage holds a paginated list of customers.
type CustomerPage struct {
	Items    []domain.Customer
	Total    int
	Page     int
	PageSize int
}

// CustomerService implements CRM customer use cases.
type CustomerService struct {
	store CustomerStore
}

// NewCustomerService constructs a CustomerService.
func NewCustomerService(store CustomerStore) *CustomerService {
	return &CustomerService{store: store}
}

// List returns paginated customers filtered by optional query.
func (s *CustomerService) List(ctx context.Context, query string, page, pageSize int) (CustomerPage, error) {
	if page < 1 {
		page = defaultPage
	}
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	items, total, err := s.store.ListCustomers(ctx, strings.TrimSpace(query), page, pageSize)
	if err != nil {
		return CustomerPage{}, err
	}
	if items == nil {
		items = []domain.Customer{}
	}
	return CustomerPage{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// Get retrieves a customer by ID.
func (s *CustomerService) Get(ctx context.Context, id string) (domain.Customer, error) {
	return s.store.GetCustomer(ctx, id)
}

// Create validates and creates a new customer.
func (s *CustomerService) Create(ctx context.Context, name, phone string) (domain.Customer, error) {
	customer, err := domain.NewCustomer(uuid.NewString(), name, phone)
	if err != nil {
		return domain.Customer{}, err
	}
	return s.store.CreateCustomer(ctx, customer)
}

// Update validates and updates an existing customer.
func (s *CustomerService) Update(ctx context.Context, id, name, phone string) (domain.Customer, error) {
	validName, err := domain.ValidateName(name)
	if err != nil {
		return domain.Customer{}, err
	}
	validPhone, err := domain.NormalizePhone(phone)
	if err != nil {
		return domain.Customer{}, err
	}
	return s.store.UpdateCustomer(ctx, id, validName, validPhone)
}

// Delete removes a customer if it has no conversations.
func (s *CustomerService) Delete(ctx context.Context, id string) error {
	return s.store.DeleteCustomer(ctx, id)
}
