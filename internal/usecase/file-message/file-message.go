package file_message

import "context"

type FileMessageRepository interface {
	DeleteFile(ctx context.Context, fileId int64) error
	InsertFile(ctx context.Context, messageId string, size int64, fileType, storagePath string) error
}

type FileMessageUsecase struct {
	fileRepo FileMessageRepository
}

func New(fileRepo FileMessageRepository) *FileMessageUsecase {
	return &FileMessageUsecase{
		fileRepo: fileRepo,
	}
}

func (uc *FileMessageUsecase) DeleteFile(ctx context.Context, fileId int64) error {
	return uc.fileRepo.DeleteFile(ctx, fileId)
}

func (uc *FileMessageUsecase) InsertFile(ctx context.Context, messageId string, size int64, fileType, storagePath string) error {
	return uc.fileRepo.InsertFile(ctx, messageId, size, fileType, storagePath)
}
