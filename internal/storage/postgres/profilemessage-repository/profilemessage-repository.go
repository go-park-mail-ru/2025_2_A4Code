package profilemessage_repository

import (
	"2025_2_a4code/internal/domain"
	"database/sql"
)

type ProfileMessageRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *ProfileMessageRepository {
	return &ProfileMessageRepository{db: db}
}

func (repo *ProfileMessageRepository) FindByProfileID(profileID int64) ([]domain.ProfileMessage, error) {
	const op = "storage.postgresql.base-profile.FindByProfileID"

	// TODO: реализовать логику

	return []domain.ProfileMessage{}, nil
}

func (repo *ProfileMessageRepository) FindByMessageID(messageID int64) (*domain.ProfileMessage, error) {
	const op = "storage.postgresql.base-profile.FindByProfileID"

	// TODO: реализовать логику

	return &domain.ProfileMessage{}, nil
}
