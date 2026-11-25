package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL       string
	Port              string
	MigrationsDir     string
	StartupTimeout    time.Duration
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	DeepSeekAPIKey    string
	DeepSeekBaseURL   string
	DeepSeekModel     string
	AssistantWorkers  int
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load environment: %w", err)
	}
	databaseURL, err := ConfigFromEnv()
	if err != nil {
		return Config{}, err
	}
	port := envOrDefault("PORT", "3000")
	dir := envOrDefault("MIGRATIONS_DIR", "migrations")
	return Config{
		DatabaseURL:       databaseURL,
		Port:              port,
		MigrationsDir:     dir,
		StartupTimeout:    10 * time.Second,
		ShutdownTimeout:   10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		DeepSeekAPIKey:    os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL:   envOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekModel:     envOrDefault("DEEPSEEK_MODEL", "deepseek-chat"),
		AssistantWorkers:  envIntOrDefault("ASSISTANT_WORKERS", 2, 1),
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback, minVal int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if n < minVal {
		return minVal
	}
	return n
}

// ConfigFromEnv builds a PostgreSQL URL from the process environment.
func ConfigFromEnv() (string, error) {
	parsed, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || (parsed.Scheme != "postgres" && parsed.Scheme != "postgresql") || parsed.Host == "" {
		return "", errors.New("DATABASE_URL must be a PostgreSQL URL")
	}
	username, password := os.Getenv("DATABASE_USER"), os.Getenv("DATABASE_PASSWORD")
	if username != "" {
		parsed.User = url.UserPassword(username, password)
	}
	return parsed.String(), nil
}
