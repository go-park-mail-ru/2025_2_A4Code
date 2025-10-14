package domain

import "time"

type Message struct {
	ID              int64
	Topic           string
	Text            string
	DateOfDispatch  time.Time
	SenderProfileID int64
	ThreadID        int64
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
