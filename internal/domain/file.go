package domain

import "time"

type File struct {
	ID          int64     `json:"id" db:"id"`
	FileType    string    `json:"file_type" db:"file_type"`
	Size        int64     `json:"size" db:"size"`
	StoragePath string    `json:"storage_path" db:"storage_path"`
	MessageID   int64     `json:"message_id" db:"message_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
