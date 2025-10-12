package postgres

import (
	"database/sql"
	"fmt"
	"log"
)

const (
	ErrCodeUniqueViolation     = "23505"
	ErrCodeNotNullViolation    = "23503"
	ErrCodeForeignKeyViolation = "23502"
)

type Storage struct {
	db *sql.DB
}

func New(storagePath string) (*Storage, error) {
	const op = "storage.postgresql.New"

	db, err := sql.Open("postgres", storagePath)
	if err != nil {
		return nil, fmt.Errorf(op+": %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf(op+": %w", err)
	}

	log.Println("Connected to postgresql successfully")

	return &Storage{db: db}, nil
}
