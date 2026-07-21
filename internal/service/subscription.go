package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/CarambaG/subscription-service/internal/domain"
	"github.com/google/uuid"
)

const (
	DefaultPageSize = 50
	MaxPageSize     = 200
)

type SubscriptionRepository interface {
	Create(context.Context, *domain.Subscription) error
	Get(context.Context, uuid.UUID) (domain.Subscription, error)
	List(context.Context, domain.ListFilter) ([]domain.Subscription, int, error)
	Update(context.Context, *domain.Subscription) error
	Delete(context.Context, uuid.UUID) error
	CalculateTotal(context.Context, domain.TotalFilter) (int64, error)
}

type Service struct {
	repository SubscriptionRepository
	logger     *slog.Logger
}

func New(repository SubscriptionRepository, logger *slog.Logger) *Service {
	return &Service{repository: repository, logger: logger}
}

func (s *Service) Create(ctx context.Context, subscription domain.Subscription) (domain.Subscription, error) {
	subscription.ID = uuid.New()
	normalize(&subscription)
	if err := subscription.Validate(); err != nil {
		return domain.Subscription{}, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}
	if err := s.repository.Create(ctx, &subscription); err != nil {
		s.logger.ErrorContext(ctx, "failed to create subscription", "error", err)
		return domain.Subscription{}, err
	}
	s.logger.InfoContext(ctx, "subscription created",
		"subscription_id", subscription.ID,
		"user_id", subscription.UserID,
	)
	return subscription, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Subscription, error) {
	subscription, err := s.repository.Get(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		s.logger.WarnContext(ctx, "subscription not found", "subscription_id", id)
		return domain.Subscription{}, err
	}
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get subscription", "subscription_id", id, "error", err)
		return domain.Subscription{}, err
	}
	s.logger.DebugContext(ctx, "subscription retrieved", "subscription_id", id)
	return subscription, nil
}

func (s *Service) List(ctx context.Context, filter domain.ListFilter) ([]domain.Subscription, int, error) {
	if filter.Limit == 0 {
		filter.Limit = DefaultPageSize
	}
	if filter.Limit < 1 || filter.Limit > MaxPageSize {
		return nil, 0, fmt.Errorf("%w: limit must be between 1 and %d", domain.ErrInvalidInput, MaxPageSize)
	}
	if filter.Offset < 0 {
		return nil, 0, fmt.Errorf("%w: offset must not be negative", domain.ErrInvalidInput)
	}
	trimFilter(&filter.ServiceName)

	items, total, err := s.repository.List(ctx, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list subscriptions", "error", err)
		return nil, 0, err
	}
	s.logger.DebugContext(ctx, "subscriptions listed", "count", len(items), "total", total)
	return items, total, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, subscription domain.Subscription) (domain.Subscription, error) {
	subscription.ID = id
	normalize(&subscription)
	if err := subscription.Validate(); err != nil {
		return domain.Subscription{}, fmt.Errorf("%w: %v", domain.ErrInvalidInput, err)
	}

	if err := s.repository.Update(ctx, &subscription); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.WarnContext(ctx, "cannot update missing subscription", "subscription_id", id)
			return domain.Subscription{}, err
		}
		s.logger.ErrorContext(ctx, "failed to update subscription", "subscription_id", id, "error", err)
		return domain.Subscription{}, err
	}
	s.logger.InfoContext(ctx, "subscription updated", "subscription_id", id)
	return subscription, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repository.Delete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.WarnContext(ctx, "cannot delete missing subscription", "subscription_id", id)
			return err
		}
		s.logger.ErrorContext(ctx, "failed to delete subscription", "subscription_id", id, "error", err)
		return err
	}
	s.logger.InfoContext(ctx, "subscription deleted", "subscription_id", id)
	return nil
}

func (s *Service) CalculateTotal(ctx context.Context, filter domain.TotalFilter) (int64, error) {
	if filter.PeriodStart.IsZero() || filter.PeriodEnd.IsZero() {
		return 0, fmt.Errorf("%w: period_start and period_end are required", domain.ErrInvalidInput)
	}
	if filter.PeriodEnd.Before(filter.PeriodStart) {
		return 0, fmt.Errorf("%w: period_end must not be earlier than period_start", domain.ErrInvalidInput)
	}
	trimFilter(&filter.ServiceName)

	total, err := s.repository.CalculateTotal(ctx, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to calculate subscription total", "error", err)
		return 0, err
	}
	s.logger.DebugContext(ctx, "subscription total calculated", "total", total)
	return total, nil
}

func normalize(subscription *domain.Subscription) {
	subscription.ServiceName = strings.TrimSpace(subscription.ServiceName)
	subscription.StartDate = subscription.StartDate.UTC()
	if subscription.EndDate != nil {
		value := subscription.EndDate.UTC()
		subscription.EndDate = &value
	}
}

func trimFilter(value **string) {
	if *value == nil {
		return
	}
	trimmed := strings.TrimSpace(**value)
	if trimmed == "" {
		*value = nil
		return
	}
	*value = &trimmed
}
