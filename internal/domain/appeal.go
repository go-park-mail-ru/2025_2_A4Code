package domain

import "time"

type Appeal struct {
	Id        int64
	Email     int64
	Topic     string
	Text      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
