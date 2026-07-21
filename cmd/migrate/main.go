package main

import (
	"context"
	"fmt"
	"os"

	"github.com/CarambaG/subscription-service/internal/config"
	"github.com/CarambaG/subscription-service/internal/migrations"
	"github.com/CarambaG/subscription-service/internal/repository/postgres"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	configPath := environment("CONFIG_PATH", "config/config.yaml")
	migrationsPath := environment("MIGRATIONS_PATH", "migrations")

	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	pool, err := postgres.Connect(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := migrations.NewRunner(pool, migrationsPath).Up(ctx); err != nil {
		return err
	}
	fmt.Println("migrations applied")
	return nil
}

func environment(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
