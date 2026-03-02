package service

import (
	"context"
	"subscription-service/internal/model"
	"time"
)

type SubscriptionRepository interface {
	Create(ctx context.Context, sub *model.Subscription) error
	GetByID(ctx context.Context, id int64) (*model.Subscription, error)
	Update(ctx context.Context, sub *model.Subscription) error
	DeleteById(ctx context.Context, id int64) error
	SumCost(ctx context.Context, userID, service string, from, to time.Time) (int, error)
}
type SubscriptionService struct {
	repo SubscriptionRepository
}

func NewSubscriptionService(r SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{repo: r}
}

func (s *SubscriptionService) Create(ctx context.Context, sub *model.Subscription) error {
	return s.repo.Create(ctx, sub)
}

func (s *SubscriptionService) Get(ctx context.Context, id int64) (*model.Subscription, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SubscriptionService) Update(ctx context.Context, sub *model.Subscription) error {
	return s.repo.Update(ctx, sub)
}

func (s *SubscriptionService) Delete(ctx context.Context, id int64) error {
	return s.repo.DeleteById(ctx, id)
}

func (s *SubscriptionService) Sum(ctx context.Context, userID, service string, from, to time.Time) (int, error) {
	return s.repo.SumCost(ctx, userID, service, from, to)
}
