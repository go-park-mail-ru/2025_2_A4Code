package message

import (
	"2025_2_a4code/internal/domain"
	e "2025_2_a4code/internal/lib/wrapper"
	"context"
	"time"
)

type MessageRepository interface {
	FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error)
	FindByProfileID(ctx context.Context, profileID int64) ([]domain.Message, error)
	FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error)
	SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error)
	SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error)
	SaveThread(ctx context.Context, messageID int64) (threadID int64, err error)
	SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error
	FindByProfileIDWithKeysetPagination(ctx context.Context, profileID int64, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetMessagesStats(ctx context.Context, profileID int64) (int, int, error)
	FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error)
	MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error
	FindSentMessagesByProfileIDWithKeysetPagination(ctx context.Context, profileID int64, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetSentMessagesStats(ctx context.Context, profileID int64) (int, int, error)
}

type MessageUcase struct {
	repo MessageRepository
}

func New(repo MessageRepository) *MessageUcase {
	return &MessageUcase{repo: repo}
}

func (uc *MessageUcase) FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error) {
	return uc.repo.FindByMessageID(ctx, messageID)
}

func (uc *MessageUcase) FindByProfileID(ctx context.Context, profileID int64) ([]domain.Message, error) {
	return uc.repo.FindByProfileID(ctx, profileID)
}

func (uc *MessageUcase) FindByProfileIDWithKeysetPagination(
	ctx context.Context,
	profileID int64,
	lastMessageID int64,
	lastDatetime time.Time,
	limit int,
) ([]domain.Message, error) {
	return uc.repo.FindByProfileIDWithKeysetPagination(ctx, profileID, lastMessageID, lastDatetime, limit)
}

func (uc *MessageUcase) GetMessagesInfo(ctx context.Context, profileID int64) (domain.Messages, error) {
	const op = "usecase.message.GetMessagesInfo"

	messages, err := uc.FindByProfileID(ctx, profileID)
	if err != nil {
		return domain.Messages{}, e.Wrap(op, err)
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

func (uc *MessageUcase) GetMessagesInfoWithPagination(ctx context.Context, profileID int64) (domain.Messages, error) {
	const op = "usecase.message.GetMessagesInfoWithPagination"

	messageTotal, messageUnread, err := uc.repo.GetMessagesStats(ctx, profileID)
	if err != nil {
		return domain.Messages{}, e.Wrap(op, err)
	}

	return domain.Messages{
		MessageTotal:  messageTotal,
		MessageUnread: messageUnread,
		Messages:      nil,
	}, nil
}

func (uc *MessageUcase) FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error) {
	return uc.repo.FindFullByMessageID(ctx, messageID, profileID)
}

func (uc *MessageUcase) SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (messageID int64, err error) {
	return uc.repo.SaveMessage(ctx, receiverProfileEmail, senderBaseProfileID, topic, text)
}

func (uc *MessageUcase) SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error) {
	return uc.repo.SaveFile(ctx, messageID, fileName, fileType, storagePath, size)
}

func (uc *MessageUcase) SaveThread(ctx context.Context, messageID int64) (threadID int64, err error) {
	return uc.repo.SaveThread(ctx, messageID)
}

func (uc *MessageUcase) SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error {
	return uc.repo.SaveThreadIdToMessage(ctx, messageID, threadID)
}

func (uc *MessageUcase) GetMessagesStats(ctx context.Context, profileID int64) (int, int, error) {
	return uc.repo.GetMessagesStats(ctx, profileID)
}

func (uc *MessageUcase) FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error) {
	return uc.repo.FindThreadsByProfileID(ctx, profileID)
}

func (uc *MessageUcase) MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error {
	return uc.repo.MarkMessageAsRead(ctx, messageID, profileID)
}

func (uc *MessageUcase) FindSentMessagesByProfileIDWithKeysetPagination(ctx context.Context, profileID int64, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error) {
	return uc.repo.FindSentMessagesByProfileIDWithKeysetPagination(ctx, profileID, lastMessageID, lastDatetime, limit)
}

func (uc *MessageUcase) GetSentMessagesStats(ctx context.Context, profileID int64) (int, int, error) {
	return uc.repo.GetSentMessagesStats(ctx, profileID)
}

func (uc *MessageUcase) GetSentMessagesInfoWithPagination(ctx context.Context, profileID int64) (domain.Messages, error) {
	const op = "usecase.message.GetSentMessagesInfoWithPagination"

	messageTotal, messageUnread, err := uc.repo.GetSentMessagesStats(ctx, profileID)
	if err != nil {
		return domain.Messages{}, e.Wrap(op, err)
	}

	return domain.Messages{
		MessageTotal:  messageTotal,
		MessageUnread: messageUnread,
		Messages:      nil,
	}, nil
}
