package rest

import "go-service/internal/catalog/domain"

type categoryRequest struct {
	Name string `json:"name"`
}

type serviceRequest struct {
	Name           string `json:"name"`
	DurationMinute int    `json:"duration_minutes"`
	PriceMinor     int64  `json:"price_minor_units"`
}

type createServiceRequest struct {
	Name           string `json:"name"`
	DurationMinute int    `json:"duration_minutes"`
	PriceMinor     int64  `json:"price_minor_units"`
	SortOrder      int    `json:"sort_order"`
}

type placementRequest struct {
	CategoryID string `json:"category_id"`
	SortOrder  int    `json:"sort_order"`
}

type categoryPlacementRequest struct {
	SortOrder int `json:"sort_order"`
}

func (r serviceRequest) service() domain.Service {
	return domain.Service{
		Name: r.Name, DurationMinute: r.DurationMinute,
		PriceMinor: r.PriceMinor,
	}
}

func (r createServiceRequest) service() domain.Service {
	return serviceRequest{
		Name: r.Name, DurationMinute: r.DurationMinute,
		PriceMinor: r.PriceMinor,
	}.service()
}
