package domain

type File struct {
	ID          int64
	Name        string
	FileType    string
	Size        int64
	StoragePath string
	MessageID   int64
}
