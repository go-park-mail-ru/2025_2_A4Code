package avatar_repository

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

type AvatarRepository struct {
	Client        *minio.Client
	BucketName    string
	publicBaseURL *url.URL
}

func New(client *minio.Client, bucketName string, publicEndpoint string, publicUseSSL bool) (*AvatarRepository, error) {
	publicBaseURL, err := buildPublicBaseURL(publicEndpoint, publicUseSSL)
	if err != nil {
		return nil, err
	}

	return &AvatarRepository{
		Client:        client,
		BucketName:    bucketName,
		publicBaseURL: publicBaseURL,
	}, nil
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

	if repo.publicBaseURL == nil {
		return presignedURL, nil
	}

	publicURL := *presignedURL
	publicURL.Scheme = repo.publicBaseURL.Scheme
	publicURL.Host = repo.publicBaseURL.Host

	return &publicURL, nil
}

func (repo *AvatarRepository) DeleteAvatar(ctx context.Context, objectName string) error {
	return repo.Client.RemoveObject(ctx, repo.BucketName, objectName, minio.RemoveObjectOptions{})
}

func buildPublicBaseURL(endpoint string, useSSL bool) (*url.URL, error) {
	if strings.TrimSpace(endpoint) == "" {
		return nil, nil
	}

	candidate := endpoint
	if !strings.Contains(endpoint, "://") {
		scheme := "http"
		if useSSL {
			scheme = "https"
		}
		candidate = fmt.Sprintf("%s://%s", scheme, endpoint)
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return nil, fmt.Errorf("avatar repository: invalid public endpoint: %w", err)
	}

	if parsed.Host == "" {
		return nil, fmt.Errorf("avatar repository: public endpoint must include host, got %q", endpoint)
	}

	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return parsed, nil
}
