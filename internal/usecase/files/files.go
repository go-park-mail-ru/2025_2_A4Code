package files

import (
	"2025_2_a4code/internal/lib/rand"
	e "2025_2_a4code/internal/lib/wrapper"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"time"
)

type FileRepository interface {
	UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error
	GetFilePresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error)
	DeleteFile(ctx context.Context, objectName string) error
}

type FileUcase struct {
	fileRepo FileRepository
}

func New(fileRepo FileRepository) *FileUcase {
	return &FileUcase{
		fileRepo: fileRepo,
	}
}

func (uc *FileUcase) UploadFileMain(ctx context.Context, messageID string, file io.Reader, size int64, originalFilename string) (string, string, error) {
	const op = "usecase.files.UploadFileMain"
	ext := filepath.Ext(originalFilename)

	randId, err := rand.GenerateRandID()
	if err != nil {
		return "", "", e.Wrap(op, err)
	}
	objectName := fmt.Sprintf("file/%s/%s%s", messageID, randId, ext)
	contentType := "application/octet-stream"

	err = uc.fileRepo.UploadFile(ctx, objectName, file, size, contentType)
	if err != nil {
		return "", "", e.Wrap(op+" could not upload file to storage: ", err)
	}

	url, err := uc.fileRepo.GetFilePresignedURL(ctx, objectName, 15*time.Minute)
	if err != nil {
		return "", "", e.Wrap(op+" could not get presigned URL: ", err)
	}

	return objectName, url.String(), nil
}

func (uc *FileUcase) DeleteFile(ctx context.Context, objectName string) error {
	return uc.fileRepo.DeleteFile(ctx, objectName)
}

func (uc *FileUcase) GetFilePresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
	return uc.fileRepo.GetFilePresignedURL(ctx, objectName, duration)
}

func (uc *FileUcase) UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
	return uc.fileRepo.UploadFile(ctx, objectName, data, size, contentType)
}
