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

	const query = `
		SELECT
            a.id, a.topic, a.text, a.appeal_status,
			a.created_at, a.updated_at
        FROM
            appeal a
        LEFT JOIN
            folder f ON fpm.folder_id = f.id AND f.profile_id = recipient_profile.id
        JOIN
            base_profile bp ON m.sender_base_profile_id = bp.id
        LEFT JOIN
            profile sender_profile ON bp.id = sender_profile.base_profile_id
        WHERE
            recipient_profile.base_profile_id = $1
			AND (($2 = 0 AND $3 = 0) OR (m.date_of_dispatch, m.id) < (to_timestamp($3), $2))
        GROUP BY 
            m.id, 
            m.topic, 
            m.text, 
            m.date_of_dispatch,
            pm.read_status,
            bp.id, 
            bp.username, 
            bp.domain,
            sender_profile.name, 
            sender_profile.surname, 
            sender_profile.image_path
        ORDER BY
            m.date_of_dispatch DESC, m.id DESC
		FETCH FIRST $4 ROWS ONLY`

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
	const op = "storage.appealRepository.SaveAppeal"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		INSERT INTO appeal (topic, text, profile_id)
		VALUES ($1, $2, $3)`

	log.Debug("Execute SaveAppeal query...")

	_, err := repo.db.ExecContext(ctx, query, topic, text, profileID)
	if err != nil {
		return e.Wrap(op, err)
	}

	return nil
}
