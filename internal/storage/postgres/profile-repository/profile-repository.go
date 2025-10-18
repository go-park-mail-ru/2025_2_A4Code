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
	// TODO: реализовать логику
	return true, nil
}

func (repo *ProfileRepository) CreateUser(ctx context.Context, profile domain.Profile) (int64, error) {
	// TODO: реализовать логику
	var variable int64
	return variable, nil
}

func (repo *ProfileRepository) FindByUsernameAndDomain(ctx context.Context, username string, emailDomain string) (*domain.Profile, error) {
	const op = "storage.postgresql.base-profile.FindByUsernameAndDomain"
	// TODO: реализовать логику
	return &domain.Profile{}, nil
}
