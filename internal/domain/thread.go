package domain

import "time"

type Thread struct {
	ID          int64     `json:"id" db:"id"`
	RootMessage int64     `json:"root_message" db:"root_message"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
