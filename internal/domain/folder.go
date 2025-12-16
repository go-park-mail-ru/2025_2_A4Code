package domain

import "time"

//go:generate go run github.com/mailru/easyjson/easyjson@v0.9.1 -all folder.go

type FolderType string

const (
	FolderInbox  FolderType = "inbox"
	FolderSent   FolderType = "sent"
	FolderDrafts FolderType = "drafts"
	FolderTrash  FolderType = "trash"
	FolderSpam   FolderType = "spam"
	FolderCustom FolderType = "custom"
)

type Folder struct {
	ID        int64      `json:"id"`
	ProfileID int64      `json:"profile_id"`
	Name      string     `json:"name"`
	Type      FolderType `json:"type"`
	CreatedAt time.Time  `json:"created_at"`
}
