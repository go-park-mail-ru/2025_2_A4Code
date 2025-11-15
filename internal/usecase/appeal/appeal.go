package appeal

import (
	"2025_2_a4code/internal/domain"
	"context"
	"time"
)

type AppealRepository interface {
	SaveAppeal(ctx context.Context, profileID int64, topic, text string) error
	FindByProfileIDWithKeysetPagination(ctx context.Context, profileID, lastAppealID int64, lastDatetime time.Time, limit int) ([]domain.Appeal, error)
	FindAppealByProfileID(ctx context.Context, profileID int64) (domain.Appeal, error)
}

type AppealUsecase struct {
	repo AppealRepository
}

func New(repo AppealRepository) *AppealUsecase {
	return &AppealUsecase{repo: repo}
}

func (uc *AppealUsecase) FindByProfileIDWithKeysetPagination(
	ctx context.Context,
	profileID, lastAppealID int64,
	lastDatetime time.Time,
	limit int,
) ([]domain.Appeal, error) {
	return uc.repo.FindByProfileIDWithKeysetPagination(ctx, profileID, lastAppealID, lastDatetime, limit)
}

func (uc *AppealUsecase) SaveAppeal(
	ctx context.Context,
	profileID int64, topic, text string,
) error {
	return uc.repo.SaveAppeal(ctx, profileID, topic, text)
}

func FindAppealByProfileID(ctx context.Context, profileID int64) (domain.Appeal, error) {
	return domain.Appeal{}, nil
}
