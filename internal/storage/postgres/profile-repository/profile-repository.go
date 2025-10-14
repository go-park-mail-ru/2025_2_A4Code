package profile_repository

import (
	"2025_2_a4code/internal/domain"
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
