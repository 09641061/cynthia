package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-service/internal/bootstrap/config"
	"go-service/internal/shared/circuitbreaker"
	"go-service/internal/shared/postgres"
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
	breaker, err := circuitbreaker.New(5, 30*time.Second)
	if err != nil {
		return fmt.Errorf("configure PostgreSQL circuit breaker: %w", err)
	}
	client := postgres.NewClient(db, breaker)
	router, startWorker := newApp(client, cfg)
	if client != nil && strings.TrimSpace(cfg.DeepSeekAPIKey) != "" {
		workerCtx, cancelWorker := context.WithCancel(ctx)
		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			startWorker(workerCtx)
		}()
		defer func() {
			cancelWorker()
			wg.Wait()
		}()
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
