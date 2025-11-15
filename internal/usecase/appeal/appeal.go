package appeal

import (
	"2025_2_a4code/internal/domain"
	"context"
	"time"
)

type AppealRepository interface {
	SaveAppeal(ctx context.Context, profileID int64, topic, text string) error
	FindByProfileIDWithKeysetPagination(ctx context.Context, profileID, lastAppealID int64, lastDatetime time.Time, limit int) ([]domain.Appeal, error)
	FindAllAppeals(ctx context.Context, lastAppealID int64, lastDatetime time.Time, limit int) ([]domain.AdminAppeal, error)
	UpdateAppealStatus(ctx context.Context, appealID int64, status string) error
	GetAppealsStats(ctx context.Context) (domain.SupportStats, error)
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

func (uc *AppealUsecase) FindAllAppeals(
	ctx context.Context,
	lastAppealID int64,
	lastDatetime time.Time,
	limit int,
) ([]domain.AdminAppeal, error) {
	return uc.repo.FindAllAppeals(ctx, lastAppealID, lastDatetime, limit)
}

func (uc *AppealUsecase) UpdateAppealStatus(ctx context.Context, appealID int64, status string) error {
	return uc.repo.UpdateAppealStatus(ctx, appealID, status)
}

func (uc *AppealUsecase) GetAppealsStats(ctx context.Context) (domain.SupportStats, error) {
	return uc.repo.GetAppealsStats(ctx)
}
