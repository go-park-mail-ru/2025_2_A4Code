package domain

type Settings struct {
	ID                    int64    `json:"id"`
	ProfileID             int64    `json:"profile_id"`
	NotificationTolerance string   `json:"notification_tolerance"`
	Language              string   `json:"language"`
	Theme                 string   `json:"theme"`
	Signatures            []string `json:"signatures"`
}

type Signatures []string

func SetDefaultSettings(profileID int64) Settings {
	return Settings{
		ProfileID:             profileID,
		Language:              "ru",
		Theme:                 "light",
		NotificationTolerance: "normal",
		//Signature:             "",
	}
}
