package catalog

import "context"

// CatalogServiceItem is the full service projection exposed by catalog for assistant queries.
type CatalogServiceItem struct {
	ID              string
	CategoryName    string
	Name            string
	DurationMinutes int
	PriceMinorUnits int64
}

// ListCatalogServices returns all services ordered by category sort_order/name then service sort_order/name.
func (r *ServiceReader) ListCatalogServices(ctx context.Context) ([]CatalogServiceItem, error) {
	if r == nil || r.db == nil {
		return []CatalogServiceItem{}, nil
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT s.id, c.name, s.name, s.duration_minutes, s.price_minor_units
		 FROM catalog_services s
		 JOIN catalog_categories c ON c.id = s.category_id
		 ORDER BY c.sort_order ASC, c.name ASC, s.sort_order ASC, s.name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]CatalogServiceItem, 0)
	for rows.Next() {
		var item CatalogServiceItem
		if err := rows.Scan(&item.ID, &item.CategoryName, &item.Name, &item.DurationMinutes, &item.PriceMinorUnits); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
