// Comando principal da API do sistema de controle financeiro.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nicolastomaschewski/controle-financeiro/internal/config"
	"github.com/nicolastomaschewski/controle-financeiro/internal/database"
	"github.com/nicolastomaschewski/controle-financeiro/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(logger); err != nil {
		logger.Error("application_exit_error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := database.Connect(cfg.DatabaseURL, database.Options{
		MaxOpenConns: cfg.DBMaxConns,
		ConnTimeout:  cfg.DBConnTimeout,
	})
	if err != nil {
		return err
	}
	defer db.Close()

	migrationsDir := envOr("MIGRATIONS_DIR", "migrations")
	applied, err := database.Migrate(context.Background(), db, migrationsDir)
	if err != nil {
		return err
	}
	for _, m := range applied {
		logger.Info("migration_applied", "file", m)
	}

	handler := server.New(db, cfg, logger)

	srv := &http.Server{
		Addr:         ":" + cfg.HTTPPort,
		Handler:      handler,
		ReadTimeout:  cfg.HTTPReadTimeout,
		WriteTimeout: cfg.HTTPWriteTimeout,
		IdleTimeout:  cfg.HTTPIdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http_server_starting", "port", cfg.HTTPPort, "env", cfg.Env)
		errCh <- srv.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutdown_signal_received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		logger.Info("http_server_stopped_gracefully")
	}

	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
