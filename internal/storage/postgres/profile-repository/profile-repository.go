package profile_repository

import (
	"2025_2_a4code/internal/domain"
	"context"
	"database/sql"
)

const (
// TODO: вынести коды ошибок бд в константы
)

type ProfileRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (repo *ProfileRepository) FindByID(id int64) (*domain.Profile, error) {
	const op = "storage.postgresql.base-profile.FindByID"

	// TODO: реализовать логику

	return &domain.Profile{}, nil
}

func (repo *ProfileRepository) FindSenderByID(id int64) (*domain.Sender, error) {
	const op = "storage.postgresql.base-profile.FindSenderByID"

	// TODO: реализовать логику

	return &domain.Sender{}, nil
}

func (repo *ProfileRepository) UserExists(ctx context.Context, username string) (bool, error) {
	const op = "storage.postgresql.base-profile.UserExists"

	// TODO: реализовать логику

	return true, nil
}

func (repo *ProfileRepository) CreateUser(ctx context.Context, profile domain.Profile) (int64, error) {
	const op = "storage.postgresql.base-profile.CreateUser"

	// TODO: реализовать логику

	return *new(int64), nil
}

func (repo *ProfileRepository) FindByUsernameAndDomain(ctx context.Context, username string, emailDomain string) (*domain.Profile, error) {
	const op = "storage.postgresql.base-profile.FindByUsernameAndDomain"

	// TODO: реализовать логику

	return &domain.Profile{}, nil
}

func (repo *ProfileRepository) FindInfoByID(profileID int64) (domain.ProfileInfo, error) {
	const op = "storage.postgresql.base-profile.FindInfoByID"

	// TODO: реализовать логику

	return domain.ProfileInfo{}, nil
}
