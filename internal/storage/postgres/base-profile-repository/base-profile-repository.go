package base_profile_repository

import (
	"2025_2_a4code/internal/domain"
	"database/sql"
)

const (
// TODO: вынести коды ошибок бд в константы
)

type PostgresBaseProfileRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *PostgresBaseProfileRepository {
	return &PostgresBaseProfileRepository{db: db}
}

func (repo *PostgresBaseProfileRepository) FindByID(id int64) (*domain.BaseProfile, error) {
	const op = "storage.postgresql.base-profile-repository.FindByID"

	// TODO: реализовать логику

	return &domain.BaseProfile{}, nil
}
