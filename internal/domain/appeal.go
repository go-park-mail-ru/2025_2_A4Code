package domain

import "time"

type Appeal struct {
	id        int64
	email     int64
	topic     string
	text      string
	status    string
	createdAt time.Time
	updatedAt time.Time
}
