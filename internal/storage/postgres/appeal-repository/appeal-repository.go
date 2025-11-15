package appeal_repository

import (
	"2025_2_a4code/internal/domain"
	"context"
	"database/sql"
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
	return make([]domain.Appeal, 0), nil
}
