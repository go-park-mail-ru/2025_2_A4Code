package appeal

import (
	"2025_2_a4code/internal/domain"
	"context"
	"time"
)

type AppealRepository interface {
	SaveAppeal(ctx context.Context, profileID int64, topic, text string) error
	FindByProfileIDWithKeysetPagination(ctx context.Context, profileID, lastAppealID int64, lastDatetime time.Time, limit int) ([]domain.Appeal, error)
	FindLastAppealByProfileID(ctx context.Context, profileID int64) (domain.Appeal, error)
	FindAppealsStatsByProfileID(ctx context.Context, profileID int64) (domain.AppealsInfo, error)
	FindAllAppealsStats(ctx context.Context) (domain.AppealsInfo, error)
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

func (uc *AppealUsecase) FindLastAppealByProfileID(ctx context.Context, profileID int64) (domain.Appeal, error) {
	return uc.repo.FindLastAppealByProfileID(ctx, profileID)
}

func (uc *AppealUsecase) FindAppealsStatsByProfileID(ctx context.Context, profileID int64) (domain.AppealsInfo, error) {
	return uc.repo.FindAppealsStatsByProfileID(ctx, profileID)
}

func (uc *AppealUsecase) FindLastAppealsInfo(ctx context.Context) (domain.AppealsInfo, error) {
	return uc.repo.FindAllAppealsStats(ctx)
}

func (uc *AppealUsecase) FindAllAppealsStats(ctx context.Context) (domain.AppealsInfo, error) {
	return uc.repo.FindAllAppealsStats(ctx)
}
