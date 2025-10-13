package usecase

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/storage/postgres/profile-repository"
)

type ProfileRepository interface {
	GetByID(id int64) (*domain.Profile, error)
}

type ProfileUcase struct {
	repo profile_repository.PostgresProfileRepository
}

func New(repo profile_repository.PostgresProfileRepository) *ProfileUcase {
	return &ProfileUcase{}
}

func (uc *ProfileUcase) GetByID(id int64) (*domain.Profile, error) {
	return uc.repo.FindByID(int64(id))
}
