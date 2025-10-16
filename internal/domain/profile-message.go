package domain

type ProfileMessage struct {
	ProfileID     int64
	MessageID     int64
	ReadStatus    bool
	DeletedStatus bool
	DraftStatus   bool
	FolderID      string
}
