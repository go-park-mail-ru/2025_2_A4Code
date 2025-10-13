package profilemessage_ucase

import "2025_2_a4code/internal/domain"

type ProfileMessageRepository interface {
	FindByProfileID(profileID int64) ([]domain.ProfileMessage, error)
	FindByMessageID(messageID int64) (*domain.ProfileMessage, error)
}

type ProfileMessageUcase struct {
	repo ProfileMessageRepository
}

func New(repo ProfileMessageRepository) *ProfileMessageUcase {
	return &ProfileMessageUcase{repo: repo}
}

func (uc *ProfileMessageUcase) FindByMessageID(messageID int64) (*domain.ProfileMessage, error) {
	return uc.repo.FindByMessageID(messageID)
}

func (uc *ProfileMessageUcase) FindByProfileID(profileID int64) ([]domain.ProfileMessage, error) {
	return uc.repo.FindByProfileID(profileID)
}
