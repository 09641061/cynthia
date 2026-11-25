package rest

import (
	"time"

	"go-service/internal/crm/application"
	"go-service/internal/crm/domain"
)

type customerInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type customerResource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type customerPageResource struct {
	Items    []customerResource `json:"items"`
	Total    int                `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

func toCustomerResource(c domain.Customer) customerResource {
	return customerResource{
		ID:        c.ID,
		Name:      c.Name,
		Phone:     c.Phone,
		Version:   c.Version,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func toCustomerPageResource(p application.CustomerPage) customerPageResource {
	items := make([]customerResource, 0, len(p.Items))
	for _, c := range p.Items {
		items = append(items, toCustomerResource(c))
	}
	return customerPageResource{
		Items:    items,
		Total:    p.Total,
		Page:     p.Page,
		PageSize: p.PageSize,
	}
}
