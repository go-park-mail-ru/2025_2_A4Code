package profile_repository

import (
	"2025_2_a4code/internal/domain"
	"database/sql"
	"fmt"
	"log"
)

const (
	ErrCodeUniqueViolation     = "23505"
	ErrCodeNotNullViolation    = "23503"
	ErrCodeForeignKeyViolation = "23502"
)

type PostgresProfileRepository struct {
	db *sql.DB
}

func New(storagePath string) (*PostgresProfileRepository, error) {
	const op = "storage.postgresql.profile-repository.New"

	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, fmt.Errorf(op+": %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf(op+": %w", err)
	}

	log.Println("Connected to postgresql successfully")

	return &PostgresProfileRepository{db: db}, nil
}

func (repo *PostgresProfileRepository) FindByID(id int64) (*domain.Profile, error) {
	const op = "storage.postgresql.profile-repository.FindByID"

	// TODO: реализовать логику

	return &domain.Profile{}, nil
}
