package domain

import "time"

type Message struct {
	ID                  int64     `json:"id" db:"id"`
	Topic               string    `json:"topic" db:"topic"`
	Text                string    `json:"text" db:"text"`
	DateOfDispatch      time.Time `json:"date_of_dispatch" db:"date_of_dispatch"`
	SenderBaseProfileID int64     `json:"sender_base_profile_id" db:"sender_base_profile_id"`
	ThreadID            int64     `json:"thread_id" db:"thread_id"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}
