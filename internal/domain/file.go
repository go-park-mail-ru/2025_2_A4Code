package domain

type File struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FileType    string `json:"file_type"`
	Size        int64  `json:"size"`
	StoragePath string `json:"storage_path"`
	MessageID   int64  `json:"message_id"`
}

type Files []File
