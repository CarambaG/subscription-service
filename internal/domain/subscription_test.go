package domain

import (
	"testing"
	"time"
)

func mustMonth(t *testing.T, value string) time.Time {
	t.Helper()
	month, err := ParseMonth(value)
	if err != nil {
		t.Fatalf("ParseMonth(%q): %v", value, err)
	}
	return month
}

func TestMonthsInIntersection(t *testing.T) {
	tests := []struct {
		name              string
		subscriptionStart string
		subscriptionEnd   *string
		periodStart       string
		periodEnd         string
		want              int64
	}{
		{name: "one month", subscriptionStart: "07-2025", periodStart: "07-2025", periodEnd: "07-2025", want: 1},
		{name: "whole subscription", subscriptionStart: "01-2025", subscriptionEnd: ptr("03-2025"), periodStart: "01-2025", periodEnd: "03-2025", want: 3},
		{name: "left overlap", subscriptionStart: "01-2025", subscriptionEnd: ptr("04-2025"), periodStart: "03-2025", periodEnd: "06-2025", want: 2},
		{name: "right overlap", subscriptionStart: "05-2025", subscriptionEnd: ptr("08-2025"), periodStart: "03-2025", periodEnd: "06-2025", want: 2},
		{name: "no overlap", subscriptionStart: "01-2025", subscriptionEnd: ptr("02-2025"), periodStart: "03-2025", periodEnd: "06-2025", want: 0},
		{name: "open subscription", subscriptionStart: "01-2025", periodStart: "11-2025", periodEnd: "02-2026", want: 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var end *time.Time
			if tt.subscriptionEnd != nil {
				value := mustMonth(t, *tt.subscriptionEnd)
				end = &value
			}
			got := MonthsInIntersection(
				mustMonth(t, tt.subscriptionStart),
				end,
				mustMonth(t, tt.periodStart),
				mustMonth(t, tt.periodEnd),
			)
			if got != tt.want {
				t.Fatalf("MonthsInIntersection() = %d, want %d", got, tt.want)
			}
		})
	}
}

func ptr(value string) *string {
	return &value
}
