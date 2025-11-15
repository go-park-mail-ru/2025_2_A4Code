package domain

import "time"

type Appeal struct {
	Id        int64
	Email     string
	Topic     string
	Text      string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type AppealsInfo struct {
	TotalAppeals      int
	OpenAppeals       int
	InProgressAppeals int
	ClosedAppeals     int
	LastAppeal        Appeal
}
