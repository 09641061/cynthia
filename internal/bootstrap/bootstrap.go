package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"go-service/internal/bootstrap/config"
	"go-service/internal/crm"
	"go-service/internal/iam"
	"go-service/internal/shared/circuitbreaker"
	"go-service/internal/shared/postgres"
	sharedredis "go-service/internal/shared/redis"
)

// Run loads configuration, initializes dependencies and serves requests until cancellation.
func Run(ctx context.Context) (resultErr error) {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	startupCtx, cancel := context.WithTimeout(ctx, cfg.StartupTimeout)
	defer cancel()
	db, err := postgres.Open(startupCtx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close PostgreSQL connection: %w", err))
		}
	}()
	if err := postgres.Migrate(startupCtx, db, cfg.MigrationsDir); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	redisClient, err := sharedredis.Open(startupCtx, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			resultErr = errors.Join(resultErr, fmt.Errorf("close Redis connection: %w", err))
		}
	}()
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		return fmt.Errorf("configure PostgreSQL circuit breaker: %w", err)
	}
	router, err := newRouter(postgres.NewClient(db, breaker), redisClient, iam.ModuleConfig{
		ResendAPIKey: cfg.ResendAPIKey,
		ResendFrom:   cfg.ResendFrom,
		AppURL:       cfg.AppURL,
		SessionTTL:   cfg.SessionTTL,
		RefreshTTL:   cfg.RefreshTTL,
		InviteTTL:    cfg.InviteTTL,
	}, crm.WhatsAppConfig{
		AccessToken:     cfg.WhatsAppAccessToken,
		PhoneNumberID:   cfg.WhatsAppPhoneNumberID,
		GraphAPIVersion: cfg.WhatsAppGraphAPIVersion,
		VerifyToken:     cfg.WhatsAppVerifyToken,
		AppSecret:       cfg.WhatsAppAppSecret,
	})
	if err != nil {
		return fmt.Errorf("configure HTTP router: %w", err)
	}
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router, ReadHeaderTimeout: cfg.ReadHeaderTimeout}
	return serve(ctx, server, cfg.ShutdownTimeout)
}

func serve(ctx context.Context, server *http.Server, timeout time.Duration) error {
	serverErrors := make(chan error, 1)
	go func() { serverErrors <- server.ListenAndServe() }()
	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		return fmt.Errorf("shut down HTTP server: %w", err)
	}
	if err := <-serverErrors; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}
