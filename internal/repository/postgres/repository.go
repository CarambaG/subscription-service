package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/CarambaG/subscription-service/internal/config"
	"github.com/CarambaG/subscription-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func Connect(ctx context.Context, cfg config.Database) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns

	connectCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(connectCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, subscription *domain.Subscription) error {
	const query = `
		INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		subscription.ID,
		subscription.ServiceName,
		subscription.Price,
		subscription.UserID,
		subscription.StartDate,
		subscription.EndDate,
	).Scan(&subscription.CreatedAt, &subscription.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert subscription: %w", err)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (domain.Subscription, error) {
	const query = `
		SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions
		WHERE id = $1`

	subscription, err := scanSubscription(r.pool.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Subscription{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("select subscription: %w", err)
	}
	return subscription, nil
}

func (r *Repository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Subscription, int, error) {
	const countQuery = `
		SELECT COUNT(*)
		FROM subscriptions
		WHERE ($1::uuid IS NULL OR user_id = $1)
		  AND ($2::text IS NULL OR service_name = $2)`
	const listQuery = `
		SELECT id, service_name, price, user_id, start_date, end_date, created_at, updated_at
		FROM subscriptions
		WHERE ($1::uuid IS NULL OR user_id = $1)
		  AND ($2::text IS NULL OR service_name = $2)
		ORDER BY created_at DESC, id
		LIMIT $3 OFFSET $4`

	userID := nullableUUID(filter.UserID)
	serviceName := nullableString(filter.ServiceName)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, userID, serviceName).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count subscriptions: %w", err)
	}

	rows, err := r.pool.Query(ctx, listQuery, userID, serviceName, filter.Limit, filter.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list subscriptions: %w", err)
	}
	defer rows.Close()

	items := make([]domain.Subscription, 0, min(filter.Limit, total))
	for rows.Next() {
		subscription, scanErr := scanSubscription(rows)
		if scanErr != nil {
			return nil, 0, fmt.Errorf("scan subscription: %w", scanErr)
		}
		items = append(items, subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate subscriptions: %w", err)
	}
	return items, total, nil
}

func (r *Repository) Update(ctx context.Context, subscription *domain.Subscription) error {
	const query = `
		UPDATE subscriptions
		SET service_name = $2,
		    price = $3,
		    user_id = $4,
		    start_date = $5,
		    end_date = $6,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		subscription.ID,
		subscription.ServiceName,
		subscription.Price,
		subscription.UserID,
		subscription.StartDate,
		subscription.EndDate,
	).Scan(&subscription.CreatedAt, &subscription.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update subscription: %w", err)
	}
	return nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	command, err := r.pool.Exec(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete subscription: %w", err)
	}
	if command.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CalculateTotal(ctx context.Context, filter domain.TotalFilter) (int64, error) {
	const query = `
		SELECT COALESCE(SUM(
			price * (
				EXTRACT(YEAR FROM age(
					LEAST(COALESCE(end_date, $2::date), $2::date),
					GREATEST(start_date, $1::date)
				))::bigint * 12
				+ EXTRACT(MONTH FROM age(
					LEAST(COALESCE(end_date, $2::date), $2::date),
					GREATEST(start_date, $1::date)
				))::bigint
				+ 1
			)
		), 0)::bigint
		FROM subscriptions
		WHERE start_date <= $2::date
		  AND (end_date IS NULL OR end_date >= $1::date)
		  AND ($3::uuid IS NULL OR user_id = $3)
		  AND ($4::text IS NULL OR service_name = $4)`

	var total int64
	if err := r.pool.QueryRow(ctx, query,
		filter.PeriodStart,
		filter.PeriodEnd,
		nullableUUID(filter.UserID),
		nullableString(filter.ServiceName),
	).Scan(&total); err != nil {
		return 0, fmt.Errorf("calculate subscription total: %w", err)
	}
	return total, nil
}

func (r *Repository) Ping(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSubscription(row rowScanner) (domain.Subscription, error) {
	var subscription domain.Subscription
	err := row.Scan(
		&subscription.ID,
		&subscription.ServiceName,
		&subscription.Price,
		&subscription.UserID,
		&subscription.StartDate,
		&subscription.EndDate,
		&subscription.CreatedAt,
		&subscription.UpdatedAt,
	)
	return subscription, err
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return value.String()
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
