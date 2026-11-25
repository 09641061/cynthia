package api_test

import (
	"os"
	"testing"

	"github.com/goccy/go-yaml"
)

func TestOpenAPISpecIsValidYAML(t *testing.T) {
	contents, err := os.ReadFile("openapi.yaml")
	if err != nil {
		t.Fatalf("read OpenAPI specification: %v", err)
	}
	var specification map[string]any
	if err := yaml.Unmarshal(contents, &specification); err != nil {
		t.Fatalf("parse OpenAPI YAML: %v", err)
	}
	if specification["openapi"] != "3.1.0" {
		t.Fatalf("OpenAPI version = %v, want 3.1.0", specification["openapi"])
	}
}
