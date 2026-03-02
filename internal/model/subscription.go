package model

import "time"

type Subscription struct {
	ID          int64      `db:"id" json:"id"`
	ServiceName string     `db:"service_name" json:"service_name"`
	Price       int        `db:"price" json:"price"`
	UserID      string     `db:"user_id" json:"user_id"`
	StartDate   time.Time  `db:"start_date" json:"start_date"`
	EndDate     *time.Time `db:"end_date" json:"end_date,omitempty"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
}
