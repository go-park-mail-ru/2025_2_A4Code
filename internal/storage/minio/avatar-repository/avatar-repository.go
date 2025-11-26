package avatar_repository

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

type AvatarRepository struct {
	Client        *minio.Client
	BucketName    string
	publicBaseURL string
	publicUseSSL  bool
}

func New(client *minio.Client, bucketName string, publicBaseURL string, publicUseSSL bool) *AvatarRepository {
	return &AvatarRepository{
		Client:        client,
		BucketName:    bucketName,
		publicBaseURL: publicBaseURL,
		publicUseSSL:  publicUseSSL,
	}
}

func (repo *AvatarRepository) UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
	_, err := repo.Client.PutObject(ctx, repo.BucketName, objectName, data, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (repo *AvatarRepository) GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
	presignedURL, err := repo.Client.PresignedGetObject(ctx, repo.BucketName, objectName, duration, nil)
	if err != nil {
		return nil, err
	}

	if repo.publicBaseURL != "" {
		if override, err := url.Parse(repo.publicBaseURL); err == nil {
			host := override.Host
			if host == "" {
				host = repo.publicBaseURL
			}
			scheme := override.Scheme
			if scheme == "" {
				if repo.publicUseSSL {
					scheme = "https"
				} else {
					scheme = "http"
				}
			}

			cloned := *presignedURL
			cloned.Scheme = scheme
			cloned.Host = host
			presignedURL = &cloned
		}
	}

	return presignedURL, nil
}
func (repo *AvatarRepository) DeleteAvatar(ctx context.Context, objectName string) error {
	return repo.Client.RemoveObject(ctx, repo.BucketName, objectName, minio.RemoveObjectOptions{})
}
