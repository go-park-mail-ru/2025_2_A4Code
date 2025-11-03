package avatar

import (
	"2025_2_a4code/internal/lib/rand"
	e "2025_2_a4code/internal/lib/wrapper"
	"2025_2_a4code/internal/usecase/profile"
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strconv"
	"time"
)

type AvatarRepository interface {
	UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error
	GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error)
	DeleteAvatar(ctx context.Context, objectName string) error
}

type AvatarUcase struct {
	avatarRepo  AvatarRepository
	profileRepo profile.ProfileRepository
}

func New(storage AvatarRepository, profileRepo profile.ProfileRepository) *AvatarUcase {
	return &AvatarUcase{
		avatarRepo:  storage,
		profileRepo: profileRepo,
	}
}

func (uc *AvatarUcase) UploadAvatar(ctx context.Context, userID string, file io.Reader, size int64, originalFilename string) (string, error) {
	const op = "usecase.avatar.UploadAvatar"
	ext := filepath.Ext(originalFilename)
	if ext == "" {
		ext = ".jpg"
	}

	randId, err := rand.GenerateRandID()
	if err != nil {
		return "", e.Wrap(op, err)
	}
	objectName := fmt.Sprintf("avatar/%s/%s%s", userID, randId, ext)
	contentType := "application/octet-stream"

	err = uc.avatarRepo.UploadFile(ctx, objectName, file, size, contentType)
	if err != nil {
		return "", e.Wrap(op+" could not upload avatar to storage: ", err)
	}

	url, err := uc.avatarRepo.GetAvatarPresignedURL(ctx, objectName, 15*time.Minute)
	if err != nil {
		return "", e.Wrap(op+" could not get presigned URL: ", err)
	}

	intID, err := strconv.Atoi(userID)
	if err != nil {
		return "", e.Wrap(op, err)
	}
	stringURL := url.String()
	err = uc.profileRepo.InsertProfileAvatar(ctx, int64(intID), stringURL)

	return stringURL, nil
}

func (uc *AvatarUcase) DeleteAvatar(ctx context.Context, objectName string) error {
	return uc.avatarRepo.DeleteAvatar(ctx, objectName)
}

func (uc *AvatarUcase) GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
	return uc.avatarRepo.GetAvatarPresignedURL(ctx, objectName, duration)
}

func (uc *AvatarUcase) UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
	return uc.avatarRepo.UploadFile(ctx, objectName, data, size, contentType)
}
