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

func (repo *MessageRepository) FindByID(id int64) (*domain.Message, error) {
	const op = "storage.postgresql.message.FindByMessageID"

	// TODO: реализовать логику (возможно этот метод будет использоваться в ProfileMessageRepository)

	return &domain.Message{}, nil
}
