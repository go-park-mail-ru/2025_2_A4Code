package avatar_repository

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

type AvatarRepository struct {
	Client              *minio.Client
	publicPresignClient *minio.Client
	BucketName          string
}

func New(client *minio.Client, publicClient *minio.Client, bucketName string) *AvatarRepository {
	return &AvatarRepository{
		Client:              client,
		publicPresignClient: publicClient,
		BucketName:          bucketName,
	}
}

func (repo *AvatarRepository) UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {

	_, err := repo.Client.PutObject(ctx, repo.BucketName, objectName, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})

	return err
}

func (repo *AvatarRepository) GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
	client := repo.Client
	if repo.publicPresignClient != nil {
		client = repo.publicPresignClient
	}

	presignedURL, err := client.PresignedGetObject(ctx, repo.BucketName, objectName, duration, nil)
	if err != nil {
		return nil, err
	}

	return presignedURL, nil
}

func (repo *AvatarRepository) DeleteAvatar(ctx context.Context, objectName string) error {
	return repo.Client.RemoveObject(ctx, repo.BucketName, objectName, minio.RemoveObjectOptions{})
}
