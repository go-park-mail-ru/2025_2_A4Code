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
            a.id, a.topic, a.text, a.status,
			a.created_at, a.updated_at
        FROM
            appeal a
        WHERE
            a.base_profile_id = $1
			AND (($2 = 0 AND $3 = 0) OR (a.created_at, a.id) < (to_timestamp($3), $2))
        ORDER BY
            a.created_at DESC, a.id DESC
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
		INSERT INTO appeal (topic, text, base_profile_id)
		VALUES ($1, $2, $3)`

	log.Debug("Execute SaveAppeal query...")

	_, err := repo.db.ExecContext(ctx, query, topic, text, profileID)
	if err != nil {
		return e.Wrap(op, err)
	}

	return nil
}

func (repo *AppealRepository) FindLastAppealByProfileID(ctx context.Context, profileID int64) (domain.Appeal, error) {
	const op = "storage.appealRepository.FindLastAppealByProfileID"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		SELECT
            a.id, a.topic, a.text, a.status,
			a.created_at, a.updated_at
        FROM
            appeal a
        WHERE
            a.base_profile_id = $1
        ORDER BY
            a.updated_at DESC, a.id DESC
		LIMIT 1`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return domain.Appeal{}, e.Wrap(op, err)
	}
	defer stmt.Close()

	log.Debug("Executing FindLastAppealByProfileID query...")

	row := stmt.QueryRowContext(ctx, profileID)

	var appeal domain.Appeal

	err = row.Scan(&appeal.Id, &appeal.Topic, &appeal.Text, &appeal.Status, &appeal.CreatedAt, &appeal.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Appeal{}, nil
		}
		return domain.Appeal{}, e.Wrap(op, err)
	}

	return appeal, nil
}

func (repo *AppealRepository) FindAppealsStatsByProfileID(
	ctx context.Context,
	profileID int64,
) (domain.AppealsInfo, error) {
	const op = "storage.appealRepository.FindAppealsStatsByProfileID"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        WITH stats AS (
            SELECT
                COUNT(*) as total_count,
                COUNT(CASE WHEN status = 'open' THEN 1 END) as open_count,
                COUNT(CASE WHEN status = 'in progress' THEN 1 END) as in_progress_count,
                COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed_count
            FROM appeal
            WHERE base_profile_id = $1
        ),
        last_appeal AS (
            SELECT
                id, topic, text, status, created_at, updated_at
            FROM appeal
            WHERE base_profile_id = $1
            ORDER BY updated_at DESC, id DESC
            LIMIT 1
        )
        SELECT 
            s.total_count, s.open_count, s.in_progress_count, s.closed_count,
            la.id, la.topic, la.text, la.status, la.created_at, la.updated_at
        FROM stats s
        LEFT JOIN last_appeal la ON true`

	log.Debug("Executing appeals stats query...")

	var stats domain.AppealsInfo
	var lastAppeal domain.Appeal
	var lastAppealID sql.NullInt64
	var lastAppealTopic, lastAppealText, lastAppealStatus sql.NullString
	var lastAppealCreatedAt, lastAppealUpdatedAt sql.NullTime

	row := repo.db.QueryRowContext(ctx, query, profileID)

	err := row.Scan(
		&stats.TotalAppeals, &stats.OpenAppeals, &stats.InProgressAppeals, &stats.ClosedAppeals,
		&lastAppealID, &lastAppealTopic, &lastAppealText, &lastAppealStatus,
		&lastAppealCreatedAt, &lastAppealUpdatedAt,
	)
	if err != nil {
		return domain.AppealsInfo{}, e.Wrap(op, err)
	}

	if lastAppealID.Valid {
		lastAppeal.Id = lastAppealID.Int64
		lastAppeal.Topic = lastAppealTopic.String
		lastAppeal.Text = lastAppealText.String
		lastAppeal.Status = lastAppealStatus.String
		lastAppeal.CreatedAt = lastAppealCreatedAt.Time
		lastAppeal.UpdatedAt = lastAppealUpdatedAt.Time
		stats.LastAppeal = lastAppeal
	} else {
		stats.LastAppeal = domain.Appeal{}
	}

	return stats, nil
}

func (repo *AppealRepository) FindAllAppealsStats(
	ctx context.Context,
) (domain.AppealsInfo, error) {
	const op = "storage.appealRepository.FindAllAppealsStats"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        WITH stats AS (
            SELECT
                COUNT(*) as total_count,
                COUNT(CASE WHEN status = 'open' THEN 1 END) as open_count,
                COUNT(CASE WHEN status = 'in progress' THEN 1 END) as in_progress_count,
                COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed_count
            FROM appeal
        ),
        last_appeal AS (
            SELECT
                id, topic, text, status, created_at, updated_at
            FROM appeal
            ORDER BY updated_at DESC, id DESC
            LIMIT 1
        )
        SELECT 
            s.total_count, s.open_count, s.in_progress_count, s.closed_count,
            la.id, la.topic, la.text, la.status, la.created_at, la.updated_at
        FROM stats s
        LEFT JOIN last_appeal la ON true`

	log.Debug("Executing all appeals stats query...")

	var stats domain.AppealsInfo
	var lastAppeal domain.Appeal
	var lastAppealID sql.NullInt64
	var lastAppealTopic, lastAppealText, lastAppealStatus sql.NullString
	var lastAppealCreatedAt, lastAppealUpdatedAt sql.NullTime

	row := repo.db.QueryRowContext(ctx, query)

	err := row.Scan(
		&stats.TotalAppeals, &stats.OpenAppeals, &stats.InProgressAppeals, &stats.ClosedAppeals,
		&lastAppealID, &lastAppealTopic, &lastAppealText, &lastAppealStatus,
		&lastAppealCreatedAt, &lastAppealUpdatedAt,
	)
	if err != nil {
		return domain.AppealsInfo{}, e.Wrap(op, err)
	}

	if lastAppealID.Valid {
		lastAppeal.Id = lastAppealID.Int64
		lastAppeal.Topic = lastAppealTopic.String
		lastAppeal.Text = lastAppealText.String
		lastAppeal.Status = lastAppealStatus.String
		lastAppeal.CreatedAt = lastAppealCreatedAt.Time
		lastAppeal.UpdatedAt = lastAppealUpdatedAt.Time
		stats.LastAppeal = lastAppeal
	} else {
		stats.LastAppeal = domain.Appeal{}
	}

	return stats, nil
}

func (repo *AppealRepository) UpdateAppeal(ctx context.Context, appeal_id int64, text, status string) error {
	const op = "storage.appealRepository.UpdateAppeal"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		UPDATE appeal
		SET text = $2,
			status = $3
		WHERE id = $1
	`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return e.Wrap(op, err)
	}
	defer stmt.Close()

	textNull := sql.NullString{String: text, Valid: text != ""}
	statusNull := sql.NullString{String: status, Valid: status != ""}

	log.Debug("Executing UpdateAppeal query...")
	_, err = stmt.ExecContext(ctx, appeal_id, textNull, statusNull)
	if err != nil {
		return e.Wrap(op, err)
	}

	return nil
}
