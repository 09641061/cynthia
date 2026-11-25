package application

import (
	"context"

	"go-service/internal/crm/domain"
)

// CustomerStore defines persistence operations for CRM customers.
type CustomerStore interface {
	ListCustomers(ctx context.Context, query string, page, pageSize int) ([]domain.Customer, int, error)
	GetCustomer(ctx context.Context, id string) (domain.Customer, error)
	GetCustomerByPhone(ctx context.Context, phone string) (domain.Customer, error)
	CreateCustomer(ctx context.Context, customer domain.Customer) (domain.Customer, error)
	UpdateCustomer(ctx context.Context, id string, expectedVersion int64, name, phone string) (domain.Customer, error)
	DeleteCustomer(ctx context.Context, id string) error
}
