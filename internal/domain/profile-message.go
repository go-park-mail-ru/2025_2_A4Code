package domain

import "time"

type ProfileMessage struct {
	ProfileID     int64     `json:"profile_id" db:"profile_id"`
	MessageID     int64     `json:"message_id" db:"message_id"`
	ReadStatus    bool      `json:"read_status" db:"read_status"`
	DeletedStatus bool      `json:"deleted_status" db:"deleted_status"`
	DraftStatus   bool      `json:"draft_status" db:"draft_status"`
	FolderName    string    `json:"folder_name" db:"folder_name"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}
