package app

import (
	"database/sql"
	"fmt"
	"log"
)

// TODO: подключение хэндлеров, бд и тд

type Storage struct {
	db *sql.DB
}

func Init() {}

func newDbConnection(storagePath string) (*Storage, error) {
	const op = "app.newDbConnection"

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
