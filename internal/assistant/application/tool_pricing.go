package application

import (
	"fmt"
	"strings"
)

type toolServiceItem struct {
	ID              string `json:"id"`
	CategoryName    string `json:"category_name"`
	Name            string `json:"name"`
	DurationMinutes int    `json:"duration_minutes"`
	PriceMinorUnits int64  `json:"price_minor_units"`
	PriceFormatted  string `json:"price_formatted"`
	CurrencyCode    string `json:"currency_code,omitempty"`
}

// FormatMinorPrice formats a minor-unit price with currency code (0 decimals for CLP/PYG/JPY/KRW, 2 default).
func FormatMinorPrice(minorUnits int64, currency string) string {
	curr := strings.ToUpper(strings.TrimSpace(currency))
	sign := ""
	if minorUnits < 0 {
		sign = "-"
		minorUnits = -minorUnits
	}
	var amount string
	switch curr {
	case "CLP", "PYG", "JPY", "KRW":
		amount = fmt.Sprintf("%s%d", sign, minorUnits)
	default:
		amount = fmt.Sprintf("%s%d.%02d", sign, minorUnits/100, minorUnits%100)
	}
	if curr == "" {
		return amount
	}
	return curr + " " + amount
}

func mapToolServices(items []ServiceItem, currency string) []toolServiceItem {
	curr := strings.ToUpper(strings.TrimSpace(currency))
	out := make([]toolServiceItem, 0, len(items))
	for _, item := range items {
		out = append(out, toolServiceItem{
			ID:              item.ID,
			CategoryName:    item.CategoryName,
			Name:            item.Name,
			DurationMinutes: item.DurationMinutes,
			PriceMinorUnits: item.PriceMinorUnits,
			PriceFormatted:  FormatMinorPrice(item.PriceMinorUnits, curr),
			CurrencyCode:    curr,
		})
	}
	return out
}
