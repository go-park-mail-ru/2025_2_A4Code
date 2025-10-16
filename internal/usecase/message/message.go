package message

import (
	"2025_2_a4code/internal/domain"
	"fmt"
)

type MessageRepository interface {
	FindByMessageID(messageID int64) (*domain.Message, error)
	FindByProfileID(profileID int64) ([]domain.Message, error)
}

type MessageUcase struct {
	repo MessageRepository
}

func New(repo MessageRepository) *MessageUcase {
	return &MessageUcase{repo: repo}
}

func (uc *MessageUcase) FindByMessageID(messageID int64) (*domain.Message, error) {
	return uc.repo.FindByMessageID(messageID)
}

func (uc *MessageUcase) FindByProfileID(profileID int64) ([]domain.Message, error) {
	return uc.repo.FindByProfileID(profileID)
}

func (uc *MessageUcase) GetMessagesInfo(profileID int64) (domain.Messages, error) {
	const op = "usecase.message.GetMessagesInfo"

	messages, err := uc.FindByProfileID(profileID)
	if err != nil {
		return domain.Messages{}, fmt.Errorf("%s: %w", op, err)
	}

	unread := len(messages)
	for _, message := range messages {
		if message.IsRead {
			unread--
		}
	}

	return domain.Messages{
		MessageTotal:  len(messages),
		MessageUnread: unread,
		Messages:      messages,
	}, nil
}
