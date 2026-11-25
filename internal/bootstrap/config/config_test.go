package config

import "testing"

func TestLoadServerAndMigrationDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://database.example/test")
	t.Setenv("DATABASE_USER", "")
	t.Setenv("DATABASE_PASSWORD", "")
	t.Setenv("PORT", "")
	t.Setenv("MIGRATIONS_DIR", "")
	t.Setenv("DEEPSEEK_API_KEY", "")
	t.Setenv("DEEPSEEK_BASE_URL", "")
	t.Setenv("DEEPSEEK_MODEL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "3000" || cfg.MigrationsDir != "migrations" {
		t.Fatalf("unexpected defaults: port %q, migrations %q", cfg.Port, cfg.MigrationsDir)
	}
	if cfg.DeepSeekAPIKey != "" || cfg.DeepSeekBaseURL != "https://api.deepseek.com" || cfg.DeepSeekModel != "deepseek-chat" {
		t.Fatalf("unexpected deepseek defaults: %+v", cfg)
	}
	t.Setenv("PORT", "8081")
	t.Setenv("MIGRATIONS_DIR", "/custom/migrations")
	t.Setenv("DEEPSEEK_API_KEY", "sk-test")
	t.Setenv("DEEPSEEK_BASE_URL", "https://custom.deepseek.local")
	t.Setenv("DEEPSEEK_MODEL", "deepseek-reasoner")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8081" || cfg.MigrationsDir != "/custom/migrations" {
		t.Fatalf("unexpected configuration: port %q, migrations %q", cfg.Port, cfg.MigrationsDir)
	}
	if cfg.DeepSeekAPIKey != "sk-test" || cfg.DeepSeekBaseURL != "https://custom.deepseek.local" || cfg.DeepSeekModel != "deepseek-reasoner" {
		t.Fatalf("unexpected deepseek configuration: %+v", cfg)
	}
}

func TestLoadAssistantWorkers(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://database.example/test")
	cases := []struct {
		env  string
		want int
	}{
		{"", 2},
		{"4", 4},
		{"0", 1},
		{"-3", 1},
		{"not-a-number", 2},
	}
	for _, tc := range cases {
		t.Setenv("ASSISTANT_WORKERS", tc.env)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("env=%q: %v", tc.env, err)
		}
		if cfg.AssistantWorkers != tc.want {
			t.Fatalf("env=%q: got %d, want %d", tc.env, cfg.AssistantWorkers, tc.want)
		}
	}
}

func TestRedisURLFromHostEnv(t *testing.T) {
	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_HOST", "redis.example.com")
	t.Setenv("REDIS_PORT", "6380")
	t.Setenv("REDIS_USERNAME", "default")
	t.Setenv("REDIS_PASSWORD", "s3cr3t+pass")
	got := RedisURLFromEnv()
	want := "redis://default:s3cr3t+pass@redis.example.com:6380/0"
	if got != want {
		t.Fatalf("RedisURLFromEnv() = %q, want %q", got, want)
	}
}
