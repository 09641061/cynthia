package config

import "testing"

func TestLoadServerAndMigrationDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://database.example/test")
	t.Setenv("DATABASE_USER", "")
	t.Setenv("DATABASE_PASSWORD", "")
	t.Setenv("PORT", "")
	t.Setenv("MIGRATIONS_DIR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "3000" || cfg.MigrationsDir != "migrations" {
		t.Fatalf("unexpected defaults: port %q, migrations %q", cfg.Port, cfg.MigrationsDir)
	}
	t.Setenv("PORT", "8081")
	t.Setenv("MIGRATIONS_DIR", "/custom/migrations")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8081" || cfg.MigrationsDir != "/custom/migrations" {
		t.Fatalf("unexpected configuration: port %q, migrations %q", cfg.Port, cfg.MigrationsDir)
	}
}
