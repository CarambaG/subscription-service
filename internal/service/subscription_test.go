package service

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/CarambaG/subscription-service/internal/domain"
	"github.com/google/uuid"
)

type repositoryStub struct {
	created     *domain.Subscription
	updated     *domain.Subscription
	updateErr   error
	totalFilter domain.TotalFilter
	total       int64
}

func (r *repositoryStub) Create(_ context.Context, subscription *domain.Subscription) error {
	copy := *subscription
	r.created = &copy
	return nil
}

func (r *repositoryStub) Get(context.Context, uuid.UUID) (domain.Subscription, error) {
	return domain.Subscription{}, domain.ErrNotFound
}

func (r *repositoryStub) List(context.Context, domain.ListFilter) ([]domain.Subscription, int, error) {
	return nil, 0, nil
}

func (r *repositoryStub) Update(_ context.Context, subscription *domain.Subscription) error {
	copy := *subscription
	r.updated = &copy
	return r.updateErr
}

func (r *repositoryStub) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (r *repositoryStub) CalculateTotal(_ context.Context, filter domain.TotalFilter) (int64, error) {
	r.totalFilter = filter
	return r.total, nil
}

func testService(repository SubscriptionRepository) *Service {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(repository, logger)
}

func TestCreateNormalizesAndAssignsID(t *testing.T) {
	repository := &repositoryStub{}
	service := testService(repository)
	userID := uuid.New()
	start, _ := domain.ParseMonth("07-2025")

	created, err := service.Create(context.Background(), domain.Subscription{
		ServiceName: "  Yandex Plus  ",
		Price:       400,
		UserID:      userID,
		StartDate:   start,
	})
	if err != nil {
		t.Fatalf("Create(): %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("Create() did not assign an ID")
	}
	if created.ServiceName != "Yandex Plus" {
		t.Fatalf("ServiceName = %q", created.ServiceName)
	}
	if repository.created == nil || repository.created.ID != created.ID {
		t.Fatal("repository did not receive the created subscription")
	}
}

func TestUpdateMissingSubscription(t *testing.T) {
	repository := &repositoryStub{updateErr: domain.ErrNotFound}
	service := testService(repository)
	start, _ := domain.ParseMonth("07-2025")

	_, err := service.Update(context.Background(), uuid.New(), domain.Subscription{
		ServiceName: "Yandex Plus",
		Price:       400,
		UserID:      uuid.New(),
		StartDate:   start,
	})
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Update() error = %v, want ErrNotFound", err)
	}
}

func TestCalculateTotalPassesOptionalFilters(t *testing.T) {
	repository := &repositoryStub{total: 2400}
	service := testService(repository)
	start, _ := domain.ParseMonth("01-2025")
	end, _ := domain.ParseMonth("06-2025")
	userID := uuid.New()
	serviceName := "  Yandex Plus  "

	total, err := service.CalculateTotal(context.Background(), domain.TotalFilter{
		PeriodStart: start,
		PeriodEnd:   end,
		UserID:      &userID,
		ServiceName: &serviceName,
	})
	if err != nil {
		t.Fatalf("CalculateTotal(): %v", err)
	}
	if total != 2400 {
		t.Fatalf("total = %d, want 2400", total)
	}
	if repository.totalFilter.UserID == nil || *repository.totalFilter.UserID != userID {
		t.Fatal("user filter was not passed")
	}
	if repository.totalFilter.ServiceName == nil || *repository.totalFilter.ServiceName != "Yandex Plus" {
		t.Fatalf("service filter = %v", repository.totalFilter.ServiceName)
	}
}

func TestCalculateTotalRejectsInvalidPeriod(t *testing.T) {
	service := testService(&repositoryStub{})
	_, err := service.CalculateTotal(context.Background(), domain.TotalFilter{
		PeriodStart: time.Date(2025, time.June, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2025, time.May, 1, 0, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("CalculateTotal() returned no error")
	}
}
