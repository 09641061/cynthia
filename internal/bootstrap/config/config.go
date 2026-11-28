package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL                string
	RedisURL                   string
	Port                       string
	MigrationsDir              string
	ResendAPIKey               string
	ResendFrom                 string
	AppURL                     string
	SessionTTL                 time.Duration
	RefreshTTL                 time.Duration
	InviteTTL                  time.Duration
	WhatsAppAccessToken        string
	WhatsAppPhoneNumberID      string
	WhatsAppGraphAPIVersion    string
	WhatsAppVerifyToken        string
	WhatsAppAppSecret          string
	StartupTimeout             time.Duration
	ShutdownTimeout            time.Duration
	ReadHeaderTimeout          time.Duration
	DeepSeekAPIKey             string
	DeepSeekBaseURL            string
	DeepSeekModel              string
	AssistantWorkers           int
	CRMConversationIdleTimeout time.Duration
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Config{}, fmt.Errorf("load environment: %w", err)
	}
	databaseURL, err := ConfigFromEnv()
	if err != nil {
		return Config{}, err
	}
	return Config{
		DatabaseURL:                databaseURL,
		RedisURL:                   RedisURLFromEnv(),
		Port:                       envOrDefault("PORT", "3000"),
		MigrationsDir:              envOrDefault("MIGRATIONS_DIR", "migrations"),
		ResendAPIKey:               os.Getenv("RESEND_API_KEY"),
		ResendFrom:                 envOrDefault("RESEND_FROM", "Cynthia <no-reply@example.com>"),
		AppURL:                     envOrDefault("APP_URL", "http://localhost:3000"),
		SessionTTL:                 time.Duration(envIntOrDefault("SESSION_TTL_MINUTES", 30)) * time.Minute,
		RefreshTTL:                 time.Duration(envIntOrDefault("REFRESH_TTL_DAYS", 14)) * 24 * time.Hour,
		InviteTTL:                  time.Duration(envIntOrDefault("INVITE_TTL_HOURS", 72)) * time.Hour,
		WhatsAppAccessToken:        os.Getenv("WHATSAPP_ACCESS_TOKEN"),
		WhatsAppPhoneNumberID:      os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
		WhatsAppGraphAPIVersion:    envOrDefault("WHATSAPP_GRAPH_API_VERSION", "v25.0"),
		WhatsAppVerifyToken:        os.Getenv("WHATSAPP_VERIFY_TOKEN"),
		WhatsAppAppSecret:          os.Getenv("WHATSAPP_APP_SECRET"),
		StartupTimeout:             10 * time.Second,
		ShutdownTimeout:            10 * time.Second,
		ReadHeaderTimeout:          5 * time.Second,
		DeepSeekAPIKey:             os.Getenv("DEEPSEEK_API_KEY"),
		DeepSeekBaseURL:            envOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
		DeepSeekModel:              envOrDefault("DEEPSEEK_MODEL", "deepseek-chat"),
		AssistantWorkers:           envIntClamped("ASSISTANT_WORKERS", 2, 1),
		CRMConversationIdleTimeout: envDurationOrDefault("CRM_CONVERSATION_IDLE_TIMEOUT", 15*time.Minute),
	}, nil
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

// RedisURLFromEnv builds a Redis URL from REDIS_URL or REDIS_HOST/PORT/USERNAME/PASSWORD.
func RedisURLFromEnv() string {
	if raw := os.Getenv("REDIS_URL"); raw != "" {
		return raw
	}
	host := envOrDefault("REDIS_HOST", "127.0.0.1")
	port := envOrDefault("REDIS_PORT", "6379")
	u := &url.URL{
		Scheme: "redis",
		Host:   net.JoinHostPort(host, port),
		Path:   "/0",
	}
	username, password := os.Getenv("REDIS_USERNAME"), os.Getenv("REDIS_PASSWORD")
	if username != "" || password != "" {
		if username == "" {
			username = "default"
		}
		u.User = url.UserPassword(username, password)
	}
	return u.String()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func envIntClamped(key string, fallback, minVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			if n < minVal {
				return minVal
			}
			return n
		}
	}
	return fallback
}

func envDurationOrDefault(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return fallback
}
