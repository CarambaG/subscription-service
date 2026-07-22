package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/CarambaG/subscription-service/internal/domain"
	"github.com/CarambaG/subscription-service/internal/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCalculateTotalIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	if err := migrations.NewRunner(pool, "../../../migrations").Up(ctx); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE subscriptions"); err != nil {
		t.Fatalf("truncate subscriptions: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "TRUNCATE subscriptions")
	})

	userA := uuid.MustParse("60601fee-2bf1-4721-ae6f-7636e79a0cba")
	userB := uuid.MustParse("a662a6d8-42cd-462f-9431-8c893d44f1fa")
	insertSubscription(t, pool, "Yandex Plus", 400, userA, "01-2025", ptrMonth(t, "03-2025"))
	insertSubscription(t, pool, "Yandex Plus", 500, userA, "04-2025", nil)
	insertSubscription(t, pool, "VK Music", 100, userB, "02-2025", ptrMonth(t, "05-2025"))

	periodStart := month(t, "02-2025")
	periodEnd := month(t, "04-2025")
	yandex := "Yandex Plus"

	tests := []struct {
		name        string
		userID      *uuid.UUID
		serviceName *string
		want        int64
	}{
		{name: "all overlapping subscriptions", want: 1600},
		{name: "filter by user", userID: &userA, want: 1300},
		{name: "filter by service", serviceName: &yandex, want: 1300},
		{name: "combine filters", userID: &userB, serviceName: &yandex, want: 0},
	}

	repository := New(pool)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			total, err := repository.CalculateTotal(ctx, domain.TotalFilter{
				PeriodStart: periodStart,
				PeriodEnd:   periodEnd,
				UserID:      tt.userID,
				ServiceName: tt.serviceName,
			})
			if err != nil {
				t.Fatalf("CalculateTotal(): %v", err)
			}
			if total != tt.want {
				t.Fatalf("total = %d, want %d", total, tt.want)
			}
		})
	}
}

func insertSubscription(
	t *testing.T,
	pool *pgxpool.Pool,
	serviceName string,
	price int64,
	userID uuid.UUID,
	startDate string,
	endDate any,
) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO subscriptions (id, service_name, price, user_id, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), serviceName, price, userID, month(t, startDate), endDate,
	)
	if err != nil {
		t.Fatalf("insert subscription: %v", err)
	}
}

func month(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := domain.ParseMonth(value)
	if err != nil {
		t.Fatalf("ParseMonth(%q): %v", value, err)
	}
	return parsed
}

func ptrMonth(t *testing.T, value string) *time.Time {
	parsed := month(t, value)
	return &parsed
}
