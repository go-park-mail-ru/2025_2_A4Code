package profile

import (
	"2025_2_a4code/internal/domain"
)

type ProfileRepository interface {
	FindByID(id int64) (*domain.Profile, error)
	FindSenderByID(id int64) (*domain.Sender, error)
}

type ProfileUcase struct {
	repo ProfileRepository
}

func New(repo ProfileRepository) *ProfileUcase {
	return &ProfileUcase{repo: repo}
}

func (uc *ProfileUcase) FindByID(id int64) (*domain.Profile, error) {
	return uc.repo.FindByID(int64(id))
}

func (uc *ProfileUcase) FindSenderByID(id int64) (*domain.Sender, error) {
	return uc.repo.FindSenderByID(int64(id))
}
