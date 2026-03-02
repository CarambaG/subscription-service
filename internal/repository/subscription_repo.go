package repository

import (
	"context"
	"subscription-service/internal/model"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionRepo struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepo(pool *pgxpool.Pool) *SubscriptionRepo {
	return &SubscriptionRepo{pool: pool}
}

func (r *SubscriptionRepo) Create(ctx context.Context, sub *model.Subscription) error {
	query := `
	INSERT INTO subscriptions 
	(service_name, price, user_id, start_date, end_date)
	VALUES ($1,$2,$3,$4,$5) RETURNING id, created_at`

	return r.pool.QueryRow(
		ctx,
		query,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
	).Scan(
		&sub.ID,
		&sub.CreatedAt,
	)
}

func (r *SubscriptionRepo) GetByID(ctx context.Context, id int64) (*model.Subscription, error) {
	var sub model.Subscription
	query := `
		SELECT
		id, 
		service_name, 
		price, 
		user_id, 
		start_date, 
		end_date, 
		created_at FROM subscriptions WHERE id=$1`
	err := r.pool.QueryRow(
		ctx,
		query,
		id).Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&sub.StartDate,
		&sub.EndDate,
		&sub.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (r *SubscriptionRepo) Update(ctx context.Context, sub *model.Subscription) error {
	query := `
		UPDATE subscriptions 
		SET
			service_name = $1,
			price = $2,
			user_id = $3,
			start_date = $4,
			end_date = $5
		WHERE id=$6`
	_, err := r.pool.Exec(
		ctx,
		query,
		sub.ServiceName,
		sub.Price,
		sub.UserID,
		sub.StartDate,
		sub.EndDate,
		sub.ID,
	)
	return err
}

func (r *SubscriptionRepo) DeleteById(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM subscriptions WHERE id=$1", id)
	return err
}

func (r *SubscriptionRepo) SumCost(ctx context.Context, userID, service string, from, to time.Time) (int, error) {
	var total int
	query := `
	SELECT COALESCE(SUM(price),0)
	FROM subscriptions
	WHERE user_id=$1::uuid
	  AND service_name=$2
	  AND start_date >= $3
	  AND (end_date IS NULL OR end_date <= $4)`

	err := r.pool.QueryRow(ctx, query, userID, service, from, to).Scan(&total)
	return total, err
}
