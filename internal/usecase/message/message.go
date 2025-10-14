package message

import "2025_2_a4code/internal/domain"

type MessageRepository struct {
	FindByID func(messageID int64) (*domain.Message, error)
}

type MessageUcase struct {
	repo MessageRepository
}

func New(repo MessageRepository) *MessageUcase {
	return &MessageUcase{repo: repo}
}

func (uc *MessageUcase) FindByID(messageID int64) (*domain.Message, error) {
	return uc.repo.FindByID(messageID)
}
