package appeal_repository

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	e "2025_2_a4code/internal/lib/wrapper"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
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

func (repo *AppealRepository) FindAllAppeals(
	ctx context.Context,
	lastAppealID int64,
	lastDatetime time.Time,
	limit int,
) ([]domain.AdminAppeal, error) {
	const op = "storage.appealRepository.FindAllAppeals"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		SELECT
			a.id,
			a.topic,
			a.text,
			a.status,
			a.created_at,
			a.updated_at,
			bp.id as profile_id,
			bp.username,
			bp.domain,
			COALESCE(p.name, '') AS first_name,
			COALESCE(p.surname, '') AS last_name
		FROM
			appeal a
		JOIN
			base_profile bp ON bp.id = a.base_profile_id
		LEFT JOIN
			profile p ON p.base_profile_id = bp.id
		WHERE
			(($1 = 0 AND $2 = 0) OR (a.created_at, a.id) < (to_timestamp($2), $1))
		ORDER BY
			a.created_at DESC,
			a.id DESC
		FETCH FIRST $3 ROWS ONLY`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	var lastDatetimeUnix int64
	if !lastDatetime.IsZero() {
		lastDatetimeUnix = lastDatetime.Unix()
	}

	log.Debug("Executing FindAllAppeals query...")
	rows, err := stmt.QueryContext(ctx, lastAppealID, lastDatetimeUnix, limit)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer rows.Close()

	var appeals []domain.AdminAppeal
	for rows.Next() {
		var appeal domain.AdminAppeal
		var username, domainPart, firstName, lastName string
		if err := rows.Scan(
			&appeal.ID,
			&appeal.Topic,
			&appeal.Text,
			&appeal.Status,
			&appeal.CreatedAt,
			&appeal.UpdatedAt,
			&appeal.ProfileID,
			&username,
			&domainPart,
			&firstName,
			&lastName,
		); err != nil {
			return nil, e.Wrap(op, err)
		}

		appeal.AuthorEmail = fmt.Sprintf("%s@%s", username, domainPart)
		nameParts := []string{}
		if strings.TrimSpace(firstName) != "" {
			nameParts = append(nameParts, strings.TrimSpace(firstName))
		}
		if strings.TrimSpace(lastName) != "" {
			nameParts = append(nameParts, strings.TrimSpace(lastName))
		}
		if len(nameParts) > 0 {
			appeal.AuthorName = strings.Join(nameParts, " ")
		} else {
			appeal.AuthorName = username
		}

		appeals = append(appeals, appeal)
	}

	if err := rows.Err(); err != nil {
		return nil, e.Wrap(op, err)
	}

	return appeals, nil
}

func (repo *AppealRepository) UpdateAppealStatus(ctx context.Context, appealID int64, status string) error {
	const op = "storage.appealRepository.UpdateAppealStatus"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		UPDATE appeal
		SET status = $2
		WHERE id = $1`

	log.Debug("Execute UpdateAppealStatus query...")
	_, err := repo.db.ExecContext(ctx, query, appealID, status)
	if err != nil {
		return e.Wrap(op, err)
	}
	return nil
}

func (repo *AppealRepository) GetAppealsStats(ctx context.Context) (domain.SupportStats, error) {
	const op = "storage.appealRepository.GetAppealsStats"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		SELECT
			COUNT(*) AS total,
			COUNT(*) FILTER (WHERE status = 'open') AS open,
			COUNT(*) FILTER (WHERE status = 'in progress') AS in_progress,
			COUNT(*) FILTER (WHERE status = 'closed') AS closed
		FROM appeal`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return domain.SupportStats{}, e.Wrap(op, err)
	}
	defer stmt.Close()

	var stats domain.SupportStats

	log.Debug("Executing GetAppealsStats query...")
	if err := stmt.QueryRowContext(ctx).Scan(
		&stats.TotalAppeals,
		&stats.OpenAppeals,
		&stats.InProgressAppeals,
		&stats.ClosedAppeals,
	); err != nil {
		return domain.SupportStats{}, e.Wrap(op, err)
	}

	return stats, nil
}
