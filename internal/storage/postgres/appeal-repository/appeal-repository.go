package appeal_repository

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	e "2025_2_a4code/internal/lib/wrapper"
	"context"
	"database/sql"
	"log/slog"
	"time"
)

type AppealRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *AppealRepository {
	return &AppealRepository{db: db}
}

func (repo *AppealRepository) FindByProfileIDWithKeysetPagination(
	ctx context.Context,
	profileID, lastAppealID int64,
	lastDatetime time.Time,
	limit int,
) ([]domain.Appeal, error) {
	const op = "storage.appealRepository.FindByProfileIDWithKeysetPagination"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = ``

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	var lastDatetimeUnix int64
	if !lastDatetime.IsZero() {
		lastDatetimeUnix = lastDatetime.Unix()
	}

	log.Debug("Executing FindAppeals query...")
	rows, err := stmt.QueryContext(ctx, profileID, lastAppealID, lastDatetimeUnix, limit)
	if err != nil {
		return nil, e.Wrap(op, err)
	}

	var appeals []domain.Appeal
	for rows.Next() {
		var appeal domain.Appeal
		err := rows.Scan(&appeal.Id, &appeal.Topic, &appeal.Text, &appeal.Status, &appeal.CreatedAt, &appeal.UpdatedAt)
		if err != nil {
			return nil, e.Wrap(op, err)
		}
		appeals = append(appeals, appeal)
	}

	if err := rows.Err(); err != nil {
		return nil, e.Wrap(op, err)
	}

	return appeals, nil
}

func (repo *AppealRepository) SaveAppeal(ctx context.Context, profileID int64, topic, text string) error {
	return nil
}
