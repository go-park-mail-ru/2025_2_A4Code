package domain

import "time"

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
	ID        int64
	ProfileID int64
	Name      string
	Type      FolderType
	CreatedAt time.Time
	UpdatedAt time.Time
}
