package profilemessage_repository

import (
	"2025_2_a4code/internal/domain"
	"database/sql"
)

type PostgresProfileMessageRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *PostgresProfileMessageRepository {
	return &PostgresProfileMessageRepository{db: db}
}

func (repo *PostgresProfileMessageRepository) FindByProfileID(profileID int64) ([]domain.ProfileMessage, error) {
	const op = "storage.postgresql.base-profile-repository.FindByProfileID"

	// TODO: реализовать логику

	return []domain.ProfileMessage{}, nil
}

func (repo *PostgresProfileMessageRepository) FindByMessageID(messageID int64) (*domain.ProfileMessage, error) {
	const op = "storage.postgresql.base-profile-repository.FindByProfileID"

	// TODO: реализовать логику

	return &domain.ProfileMessage{}, nil
}
