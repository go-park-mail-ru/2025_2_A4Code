package domain

type Settings struct {
	ID                    int64
	ProfileID             int64
	NotificationTolerance string
	Language              string
	Theme                 string
	Signature             string
}

func SetDefaultSettings(profileID int64) Settings {
	return Settings{
		ProfileID:             profileID,
		Language:              "ru",
		Theme:                 "light",
		NotificationTolerance: "normal",
		Signature:             "",
	}
}
