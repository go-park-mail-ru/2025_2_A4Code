package message_repository

import (
	"2025_2_a4code/internal/domain"
	"database/sql"
)

type MessageRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func (repo *MessageRepository) FindByMessageID(messageID int64) (*domain.Message, error) {
	const op = "storage.postgresql.message.FindByMessageID"

	// TODO: реализовать логику

	return &domain.Message{}, nil
}

func (repo *MessageRepository) FindFullByMessageID(messageID int64) (domain.FullMessage, error) {
	const op = "storage.postgresql.message.FindByMessageID"

	// TODO: реализовать логику

	return domain.FullMessage{}, nil
}

func (repo *MessageRepository) FindByProfileID(profileID int64) ([]domain.Message, error) {
	const op = "storage.postgresql.message.FindByProfileID"

	// TODO: реализовать логику

	return []domain.Message{}, nil
}
