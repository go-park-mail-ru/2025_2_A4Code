package message_repository

import (
	"2025_2_a4code/internal/domain"
	"database/sql"
)

type PostgresMessageRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *PostgresMessageRepository {
	return &PostgresMessageRepository{db: db}
}

func (repo *PostgresMessageRepository) FindByID(id int64) (*domain.Message, error) {
	const op = "storage.postgresql.message-repository.FindByMessageID"

	// TODO: реализовать логику (возможно этот метод будет использоваться в ProfileMessageRepository)

	return &domain.Message{}, nil
}
