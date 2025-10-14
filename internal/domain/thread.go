package domain

import "time"

type Thread struct {
	ID          int64
	RootMessage int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
