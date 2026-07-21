package migrations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const migrationLockID int64 = 732984721

type Runner struct {
	pool      *pgxpool.Pool
	directory string
}

func NewRunner(pool *pgxpool.Pool, directory string) *Runner {
	return &Runner{pool: pool, directory: directory}
}

func (r *Runner) Up(ctx context.Context) error {
	if _, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(r.directory, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("find migrations: %w", err)
	}
	sort.Strings(files)

	for _, file := range files {
		if err := r.apply(ctx, file); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runner) apply(ctx context.Context, file string) error {
	version := strings.TrimSuffix(filepath.Base(file), ".up.sql")
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", version, err)
	}

	return pgx.BeginFunc(ctx, r.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", migrationLockID); err != nil {
			return fmt.Errorf("lock migrations: %w", err)
		}

		var applied bool
		if err := tx.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
			version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			return nil
		}

		if _, err := tx.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := tx.Exec(ctx,
			"INSERT INTO schema_migrations(version) VALUES ($1)",
			version,
		); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
		return nil
	})
}
