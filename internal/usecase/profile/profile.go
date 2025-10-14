package profile

import (
	"2025_2_a4code/internal/domain"
	profile "2025_2_a4code/internal/storage/postgres/profile-repository"
)

type ProfileRepository interface {
	FindByID(id int64) (*domain.Profile, error)
}

type ProfileUcase struct {
	repo profile.ProfileRepository
}

func New(repo profile.ProfileRepository) *ProfileUcase {
	return &ProfileUcase{repo: repo}
}

func (uc *ProfileUcase) FindByID(id int64) (*domain.Profile, error) {
	return uc.repo.FindByID(int64(id))
}
