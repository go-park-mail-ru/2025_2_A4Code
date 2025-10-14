package domain

import "time"

type Settings struct {
	ID                    int64
	ProfileID             int64
	NotificationTolerance string
	Language              string
	Theme                 string
	Signature             string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func DefaultSettings(profileID int64) Settings {
	return Settings{
		ProfileID:             profileID,
		Language:              "ru",
		Theme:                 "light",
		NotificationTolerance: "normal",
		Signature:             "",
		CreatedAt:             time.Now(),
		UpdatedAt:             time.Now(),
	}
}
