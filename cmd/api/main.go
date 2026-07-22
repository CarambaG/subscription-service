package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/CarambaG/subscription-service/internal/config"
	"github.com/CarambaG/subscription-service/internal/httpapi"
	"github.com/CarambaG/subscription-service/internal/observability"
	"github.com/CarambaG/subscription-service/internal/repository/postgres"
	"github.com/CarambaG/subscription-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	logger := observability.NewLogger(cfg.Log.Level)
	slog.SetDefault(logger)

	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(rootCtx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	repository := postgres.New(pool)
	subscriptionService := service.New(repository, logger)
	handler := httpapi.NewHandler(subscriptionService, repository)
	server := &http.Server{
		Addr:              cfg.App.Address,
		Handler:           httpapi.NewRouter(handler, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	serverError := make(chan error, 1)
	go func() {
		logger.Info("server started", "address", cfg.App.Address, "log_level", cfg.Log.Level)
		serverError <- server.ListenAndServe()
	}()

	select {
	case err := <-serverError:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-rootCtx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.App.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
