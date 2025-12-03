package files_repository

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	e "2025_2_a4code/internal/lib/wrapper"
	"context"
	"database/sql"
	"log/slog"
	"time"
)

type FilesRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *FilesRepository {
	return &FilesRepository{db: db}
}

func (repo *FilesRepository) InsertFile(ctx context.Context, messageId string, size int64, fileType, storagePath string) error {
	const op = "files-repository.InsertFile"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		INSERT INTO file(file_type, size, storage_path, message_id, created_at, updated_at) 
		VALUES ($1, $2, $3, $4, $5)`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		log.Debug(err.Error())
		return e.Wrap(op, err)
	}
	defer stmt.Close()

	log.Debug("Executing InsertFile query")
	_, err = stmt.ExecContext(ctx, fileType, size, storagePath, messageId, time.Now(), time.Now())
	if err != nil {
		log.Debug(err.Error())
		return e.Wrap(op, err)
	}

	return nil
}

func (repo *FilesRepository) DeleteFile(ctx context.Context, fileId int64) error {
	const op = "files-repository.DeleteFile"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	stmt, err := repo.db.PrepareContext(ctx, `DELETE FROM file WHERE id = $1`)
	if err != nil {
		log.Debug(err.Error())
		return e.Wrap(op, err)
	}
	defer stmt.Close()

	log.Debug("Executing DeleteFile query")
	_, err = stmt.Exec(fileId)
	if err != nil {
		log.Debug(err.Error())
		return e.Wrap(op, err)
	}

	return nil
}
