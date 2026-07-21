package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MonthLayout = "01-2006"

var ErrNotFound = errors.New("subscription not found")

type Subscription struct {
	ID          uuid.UUID
	ServiceName string
	Price       int64
	UserID      uuid.UUID
	StartDate   time.Time
	EndDate     *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ListFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	Limit       int
	Offset      int
}

type TotalFilter struct {
	PeriodStart time.Time
	PeriodEnd   time.Time
	UserID      *uuid.UUID
	ServiceName *string
}

func (s Subscription) Validate() error {
	if strings.TrimSpace(s.ServiceName) == "" {
		return errors.New("service_name is required")
	}
	if len(s.ServiceName) > 255 {
		return errors.New("service_name must contain at most 255 characters")
	}
	if s.Price <= 0 {
		return errors.New("price must be greater than zero")
	}
	if s.UserID == uuid.Nil {
		return errors.New("user_id must be a valid UUID")
	}
	if s.StartDate.IsZero() {
		return errors.New("start_date is required")
	}
	if s.EndDate != nil && s.EndDate.Before(s.StartDate) {
		return errors.New("end_date must not be earlier than start_date")
	}
	return nil
}

func ParseMonth(value string) (time.Time, error) {
	parsed, err := time.Parse(MonthLayout, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("expected MM-YYYY: %w", err)
	}
	return parsed.UTC(), nil
}

func FormatMonth(value time.Time) string {
	return value.UTC().Format(MonthLayout)
}

func MonthsInclusive(start, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return int64((end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1)
}

func MonthsInIntersection(subscriptionStart time.Time, subscriptionEnd *time.Time, periodStart, periodEnd time.Time) int64 {
	if periodStart.After(periodEnd) {
		return 0
	}
	start := subscriptionStart
	if periodStart.After(start) {
		start = periodStart
	}
	end := periodEnd
	if subscriptionEnd != nil && subscriptionEnd.Before(end) {
		end = *subscriptionEnd
	}
	return MonthsInclusive(start, end)
}
