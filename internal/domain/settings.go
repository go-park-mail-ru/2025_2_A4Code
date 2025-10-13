package domain

import "time"

type Settings struct {
	ID                    int64     `json:"id" db:"id"`
	ProfileID             int64     `json:"profile_id" db:"profile_id"`
	NotificationTolerance string    `json:"notification_tolerance" db:"notification_tolerance"`
	Language              string    `json:"language" db:"language"`
	Theme                 string    `json:"theme" db:"theme"`
	Signature             string    `json:"signature" db:"signature"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time `json:"updated_at" db:"updated_at"`
}
