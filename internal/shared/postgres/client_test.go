package postgres

import (
	"net/url"
	"testing"
)

func TestConfigFromEnvAppliesCredentialsAndPreservesQuery(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://database.example:5432/takodu?sslmode=require")
	t.Setenv("DATABASE_USER", "catalog-user")
	t.Setenv("DATABASE_PASSWORD", "secret@value")
	connectionString, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv(): %v", err)
	}
	parsed, err := url.Parse(connectionString)
	if err != nil {
		t.Fatalf("parse connection string: %v", err)
	}
	username := parsed.User.Username()
	password, hasPassword := parsed.User.Password()
	if username != "catalog-user" || !hasPassword || password != "secret@value" {
		t.Fatal("database credentials were not applied correctly")
	}
	if parsed.Query().Get("sslmode") != "require" {
		t.Fatal("database URL query was not preserved")
	}
}

func TestConfigFromEnvRejectsInvalidURL(t *testing.T) {
	for _, value := range []string{"not-a-postgres-url", "https://database.example/takodu"} {
		t.Setenv("DATABASE_URL", value)
		if _, err := ConfigFromEnv(); err == nil {
			t.Errorf("ConfigFromEnv() accepted %q", value)
		}
	}
}
