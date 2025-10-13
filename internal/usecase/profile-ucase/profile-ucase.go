package profile_ucase

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/storage/postgres/base-profile-repository"
)

type BaseProfileRepository interface {
	FindByID(id int64) (*domain.BaseProfile, error)
}

type BaseProfileUcase struct {
	repo base_profile_repository.PostgresBaseProfileRepository
}

func New(repo base_profile_repository.PostgresBaseProfileRepository) *BaseProfileUcase {
	return &BaseProfileUcase{repo: repo}
}

func (uc *BaseProfileUcase) FindByID(id int64) (*domain.BaseProfile, error) {
	return uc.repo.FindByID(int64(id))
}
