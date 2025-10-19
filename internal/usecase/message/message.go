package message

import (
	"2025_2_a4code/internal/domain"
	"fmt"
)

type MessageRepository interface {
	FindByMessageID(messageID int64) (*domain.Message, error)
	FindByProfileID(profileID int64) ([]domain.Message, error)
	FindFullByMessageID(messageID int64) (domain.FullMessage, error)
	SaveMessage(topic, receiver, text string, threadID int64) (int64, error)
	SaveFile(messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error)
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

func (uc *MessageUcase) FindFullByMessageID(messageID int64) (domain.FullMessage, error) {
	return uc.repo.FindFullByMessageID(messageID)
}

func (uc *MessageUcase) SaveMessage(topic, receiver, text string, threadID int64) (messageID int64, err error) {
	return uc.repo.SaveMessage(topic, receiver, text, threadID)
}

func (uc *MessageUcase) SaveFile(messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error) {
	return uc.repo.SaveFile(messageID, fileName, fileType, storagePath, size)
}
