package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

	"github.com/elinavikhareva/ai-tutor/backend/internal/api"
	"github.com/elinavikhareva/ai-tutor/backend/internal/auth"
	"github.com/elinavikhareva/ai-tutor/backend/internal/compactor"
	"github.com/elinavikhareva/ai-tutor/backend/internal/config"
	"github.com/elinavikhareva/ai-tutor/backend/internal/db"
	"github.com/elinavikhareva/ai-tutor/backend/internal/gemini"
	"github.com/elinavikhareva/ai-tutor/backend/internal/logger"
	"github.com/elinavikhareva/ai-tutor/backend/internal/metrics"
	"github.com/elinavikhareva/ai-tutor/backend/internal/tutor"
)

const (
	shutdownTimeout = 10 * time.Second
	adminUsername   = "admin"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}
	log := logger.New(cfg.LogFormat, cfg.LogLevel)

	if err := run(cfg, log); err != nil {
		log.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	database, err := db.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	defer database.Close()

	if err := ensureAdmin(ctx, database, cfg.AppPassword); err != nil {
		return fmt.Errorf("bootstrap admin user: %w", err)
	}

	authSvc, err := auth.New([]byte(cfg.JWTSecret))
	if err != nil {
		return err
	}
	tutorSvc := tutor.New(database, gemini.NewClient(cfg.GeminiAPIKey))
	handler := api.NewHandler(authSvc, database, tutorSvc, log)

	apiServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler.Router(cfg.AllowedOrigins),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 3 * time.Minute,
		IdleTimeout:  60 * time.Second,
	}
	metricsServer := &http.Server{
		Addr:              cfg.MetricsAddr,
		Handler:           metrics.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error { return serve(ctx, apiServer, log.With("server", "api")) })
	g.Go(func() error { return serve(ctx, metricsServer, log.With("server", "metrics")) })
	g.Go(func() error {
		compactor.New(database, tutorSvc, log).Run(ctx)
		return nil
	})
	return g.Wait()
}

// serve runs srv until ctx is cancelled, then shuts it down gracefully.
func serve(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown %s: %w", srv.Addr, err)
	}
	if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	log.Info("stopped")
	return nil
}

// ensureAdmin creates the admin user on first start. APP_PASSWORD may be
// either plain text or a bcrypt hash.
func ensureAdmin(ctx context.Context, database *db.DB, password string) error {
	count, err := db.CountUsers(ctx, database)
	if err != nil {
		return fmt.Errorf("count users: %w", err)
	}
	if count > 0 {
		return nil
	}
	hash := password
	if !strings.HasPrefix(hash, "$2") {
		if hash, err = auth.HashPassword(password); err != nil {
			return err
		}
	}
	if _, err := db.CreateUser(ctx, database, adminUsername, hash); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}
