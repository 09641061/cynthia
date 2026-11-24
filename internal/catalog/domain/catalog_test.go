package domain

import (
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "valid", input: "Haircut"},
		{name: "maximum length", input: strings.Repeat("a", 120)},
		{name: "unicode counts as characters", input: strings.Repeat("界", 120)},
		{name: "empty", wantErr: true},
		{name: "whitespace", input: " \t\n", wantErr: true},
		{name: "over maximum", input: strings.Repeat("a", 121), wantErr: true},
		{name: "unicode over maximum", input: strings.Repeat("界", 121), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateName(test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateName() error = %v, want error %t", err, test.wantErr)
			}
		})
	}
}

func TestValidateService(t *testing.T) {
	validName := "Consultation"
	tests := []struct {
		name        string
		serviceName string
		currency    string
		duration    int
		price       int64
		wantErr     bool
	}{
		{name: "minimum values", currency: "USD", duration: 1, price: 0},
		{name: "maximum values", currency: "EUR", duration: 1440, price: 1},
		{name: "invalid name", serviceName: " ", currency: "USD", duration: 1, wantErr: true},
		{name: "duration below minimum", currency: "USD", duration: 0, wantErr: true},
		{name: "duration above maximum", currency: "USD", duration: 1441, wantErr: true},
		{name: "negative price", currency: "USD", duration: 1, price: -1, wantErr: true},
		{name: "lowercase currency", currency: "usd", duration: 1, wantErr: true},
		{name: "currency with surrounding spaces", currency: " USD ", duration: 1, wantErr: true},
		{name: "non-letter currency", currency: "U1D", duration: 1, wantErr: true},
		{name: "wrong currency length", currency: "US", duration: 1, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			serviceName := test.serviceName
			if serviceName == "" {
				serviceName = validName
			}
			err := ValidateService(serviceName, test.duration, test.price, test.currency)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidateService() error = %v, want error %t", err, test.wantErr)
			}
		})
	}
}
