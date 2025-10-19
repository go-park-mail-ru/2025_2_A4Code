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

func (repo *MessageRepository) SaveMessage(topic, receiver, text string, threadID int64) (messageID int64, err error) {
	const op = "storage.postgresql.message.Save"

	// TODO: реализовать логику

	return *new(int64), nil
}

func (repo *MessageRepository) SaveFile(messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error) {
	const op = "storage.postgresql.message.SaveFile"

	// TODO: реализовать логику

	return *new(int64), nil
}
