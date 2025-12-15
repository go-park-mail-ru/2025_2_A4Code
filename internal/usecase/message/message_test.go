package message

import (
	"2025_2_a4code/internal/domain"
	"context"
	"errors"
	"time"
)

var mockError = errors.New("mock repository error")

type MockMessageRepository struct {
	FindByMessageIDFn                                 func(ctx context.Context, messageID int64) (*domain.Message, error)
	FindByProfileIDFn                                 func(ctx context.Context, profileID int64) ([]domain.Message, error)
	FindFullByMessageIDFn                             func(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error)
	SaveMessageFn                                     func(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error)
	SaveFileFn                                        func(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error)
	SaveThreadFn                                      func(ctx context.Context, messageID int64) (threadID int64, err error)
	SaveThreadIdToMessageFn                           func(ctx context.Context, messageID int64, threadID int64) error
	FindByProfileIDWithKeysetPaginationFn             func(ctx context.Context, profileID int64, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetMessagesStatsFn                                func(ctx context.Context, profileID int64) (int, int, error)
	FindThreadsByProfileIDFn                          func(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error)
	MarkMessageAsReadFn                               func(ctx context.Context, messageID int64, profileID int64) error
	FindSentMessagesByProfileIDWithKeysetPaginationFn func(ctx context.Context, profileID int64, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetSentMessagesStatsFn                            func(ctx context.Context, profileID int64) (int, int, error)
	MarkMessageAsSpamFn                               func(ctx context.Context, messageID int64, profileID int64) error
	IsUsersMessageFn                                  func(ctx context.Context, messageID int64, profileID int64) (bool, error)
	SaveDraftFn                                       func(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error)
	IsDraftBelongsToUserFn                            func(ctx context.Context, draftID, profileID int64) (bool, error)
	DeleteDraftFn                                     func(ctx context.Context, draftID, profileID int64) error
	SendDraftFn                                       func(ctx context.Context, draftID, profileID int64) error
	GetDraftFn                                        func(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error)
	MoveToFolderFn                                    func(ctx context.Context, profileID, messageID, folderID int64) error
	GetFolderByTypeFn                                 func(ctx context.Context, profileID int64, folderType string) (int64, error)
	ShouldMarkAsReadFn                                func(ctx context.Context, messageID, profileID int64) (bool, error)
	CreateFolderFn                                    func(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error)
	GetUserFoldersFn                                  func(ctx context.Context, profileID int64) ([]domain.Folder, error)
	RenameFolderFn                                    func(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error)
	DeleteFolderFn                                    func(ctx context.Context, profileID, folderID int64) error
	DeleteMessageFromFolderFn                         func(ctx context.Context, profileID, messageID, folderID int64) error
	GetFolderMessagesWithKeysetPaginationFn           func(ctx context.Context, profileID, folderID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetFolderMessagesInfoFn                           func(ctx context.Context, profileID, folderID int64) (domain.Messages, error)
	SaveMessageWithFolderDistributionFn               func(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error)
	ReplyToMessageWithFolderDistributionFn            func(ctx context.Context, receiverEmail string, senderProfileID int64, threadRoot int64, topic, text string) (int64, error)
}

func (m *MockMessageRepository) FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error) {
	if m.FindByMessageIDFn != nil {
		return m.FindByMessageIDFn(ctx, messageID)
	}
	return nil, nil
}
func (m *MockMessageRepository) FindByProfileID(ctx context.Context, profileID int64) ([]domain.Message, error) {
	if m.FindByProfileIDFn != nil {
		return m.FindByProfileIDFn(ctx, profileID)
	}
	return nil, nil
}
func (m *MockMessageRepository) FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error) {
	if m.FindFullByMessageIDFn != nil {
		return m.FindFullByMessageIDFn(ctx, messageID, profileID)
	}
	return domain.FullMessage{}, nil
}
func (m *MockMessageRepository) SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error) {
	if m.SaveMessageFn != nil {
		return m.SaveMessageFn(ctx, receiverProfileEmail, senderBaseProfileID, topic, text)
	}
	return 0, nil
}
func (m *MockMessageRepository) SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error) {
	if m.SaveFileFn != nil {
		return m.SaveFileFn(ctx, messageID, fileName, fileType, storagePath, size)
	}
	return 0, nil
}
func (m *MockMessageRepository) SaveThread(ctx context.Context, messageID int64) (threadID int64, err error) {
	if m.SaveThreadFn != nil {
		return m.SaveThreadFn(ctx, messageID)
	}
	return 0, nil
}
func (m *MockMessageRepository) SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error {
	if m.SaveThreadIdToMessageFn != nil {
		return m.SaveThreadIdToMessageFn(ctx, messageID, threadID)
	}
	return nil
}
func (m *MockMessageRepository) FindByProfileIDWithKeysetPagination(ctx context.Context, profileID int64, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error) {
	if m.FindByProfileIDWithKeysetPaginationFn != nil {
		return m.FindByProfileIDWithKeysetPaginationFn(ctx, profileID, lastMessageID, lastDatetime, limit)
	}
	return nil, nil
}
func (m *MockMessageRepository) GetMessagesStats(ctx context.Context, profileID int64) (int, int, error) {
	if m.GetMessagesStatsFn != nil {
		return m.GetMessagesStatsFn(ctx, profileID)
	}
	return 0, 0, nil
}
func (m *MockMessageRepository) FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error) {
	if m.FindThreadsByProfileIDFn != nil {
		return m.FindThreadsByProfileIDFn(ctx, profileID)
	}
	return nil, nil
}
func (m *MockMessageRepository) MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error {
	if m.MarkMessageAsReadFn != nil {
		return m.MarkMessageAsReadFn(ctx, messageID, profileID)
	}
	return nil
}
func (m *MockMessageRepository) FindSentMessagesByProfileIDWithKeysetPagination(ctx context.Context, profileID int64, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error) {
	if m.FindSentMessagesByProfileIDWithKeysetPaginationFn != nil {
		return m.FindSentMessagesByProfileIDWithKeysetPaginationFn(ctx, profileID, lastMessageID, lastDatetime, limit)
	}
	return nil, nil
}
func (m *MockMessageRepository) GetSentMessagesStats(ctx context.Context, profileID int64) (int, int, error) {
	if m.GetSentMessagesStatsFn != nil {
		return m.GetSentMessagesStatsFn(ctx, profileID)
	}
	return 0, 0, nil
}
func (m *MockMessageRepository) MarkMessageAsSpam(ctx context.Context, messageID int64, profileID int64) error {
	if m.MarkMessageAsSpamFn != nil {
		return m.MarkMessageAsSpamFn(ctx, messageID, profileID)
	}
	return nil
}

func (m *MockMessageRepository) IsUsersMessage(ctx context.Context, messageID int64, profileID int64) (bool, error) {
	if m.IsUsersMessageFn != nil {
		return m.IsUsersMessageFn(ctx, messageID, profileID)
	}
	return false, nil
}

func (m *MockMessageRepository) SaveDraft(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error) {
	if m.SaveDraftFn != nil {
		return m.SaveDraftFn(ctx, profileID, draftID, receiverEmail, topic, text)
	}
	return 0, nil
}

func (m *MockMessageRepository) IsDraftBelongsToUser(ctx context.Context, draftID, profileID int64) (bool, error) {
	if m.IsDraftBelongsToUserFn != nil {
		return m.IsDraftBelongsToUserFn(ctx, draftID, profileID)
	}
	return false, nil
}

func (m *MockMessageRepository) DeleteDraft(ctx context.Context, draftID, profileID int64) error {
	if m.DeleteDraftFn != nil {
		return m.DeleteDraftFn(ctx, draftID, profileID)
	}
	return nil
}

func (m *MockMessageRepository) SendDraft(ctx context.Context, draftID, profileID int64) error {
	if m.SendDraftFn != nil {
		return m.SendDraftFn(ctx, draftID, profileID)
	}
	return nil
}

func (m *MockMessageRepository) GetDraft(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error) {
	if m.GetDraftFn != nil {
		return m.GetDraftFn(ctx, draftID, profileID)
	}
	return domain.FullMessage{}, nil
}

func (m *MockMessageRepository) MoveToFolder(ctx context.Context, profileID, messageID, folderID int64) error {
	if m.MoveToFolderFn != nil {
		return m.MoveToFolderFn(ctx, profileID, messageID, folderID)
	}
	return nil
}

func (m *MockMessageRepository) GetFolderByType(ctx context.Context, profileID int64, folderType string) (int64, error) {
	if m.GetFolderByTypeFn != nil {
		return m.GetFolderByTypeFn(ctx, profileID, folderType)
	}
	return 0, nil
}

func (m *MockMessageRepository) ShouldMarkAsRead(ctx context.Context, messageID, profileID int64) (bool, error) {
	if m.ShouldMarkAsReadFn != nil {
		return m.ShouldMarkAsReadFn(ctx, messageID, profileID)
	}
	return false, nil
}

func (m *MockMessageRepository) CreateFolder(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error) {
	if m.CreateFolderFn != nil {
		return m.CreateFolderFn(ctx, profileID, folderName)
	}
	return nil, nil
}

func (m *MockMessageRepository) GetUserFolders(ctx context.Context, profileID int64) ([]domain.Folder, error) {
	if m.GetUserFoldersFn != nil {
		return m.GetUserFoldersFn(ctx, profileID)
	}
	return nil, nil
}

func (m *MockMessageRepository) RenameFolder(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error) {
	if m.RenameFolderFn != nil {
		return m.RenameFolderFn(ctx, profileID, folderID, newName)
	}
	return nil, nil
}

func (m *MockMessageRepository) DeleteFolder(ctx context.Context, profileID, folderID int64) error {
	if m.DeleteFolderFn != nil {
		return m.DeleteFolderFn(ctx, profileID, folderID)
	}
	return nil
}

func (m *MockMessageRepository) DeleteMessageFromFolder(ctx context.Context, profileID, messageID, folderID int64) error {
	if m.DeleteMessageFromFolderFn != nil {
		return m.DeleteMessageFromFolderFn(ctx, profileID, messageID, folderID)
	}
	return nil
}

func (m *MockMessageRepository) GetFolderMessagesWithKeysetPagination(ctx context.Context, profileID, folderID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error) {
	if m.GetFolderMessagesWithKeysetPaginationFn != nil {
		return m.GetFolderMessagesWithKeysetPaginationFn(ctx, profileID, folderID, lastMessageID, lastDatetime, limit)
	}
	return nil, nil
}

func (m *MockMessageRepository) GetFolderMessagesInfo(ctx context.Context, profileID, folderID int64) (domain.Messages, error) {
	if m.GetFolderMessagesInfoFn != nil {
		return m.GetFolderMessagesInfoFn(ctx, profileID, folderID)
	}
	return domain.Messages{}, nil
}

func (m *MockMessageRepository) SaveMessageWithFolderDistribution(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error) {
	if m.SaveMessageWithFolderDistributionFn != nil {
		return m.SaveMessageWithFolderDistributionFn(ctx, receiverProfileEmail, senderBaseProfileID, topic, text)
	}
	return 0, nil
}

func (m *MockMessageRepository) ReplyToMessageWithFolderDistribution(ctx context.Context, receiverEmail string, senderProfileID int64, threadRoot int64, topic, text string) (int64, error) {
	if m.ReplyToMessageWithFolderDistributionFn != nil {
		return m.ReplyToMessageWithFolderDistributionFn(ctx, receiverEmail, senderProfileID, threadRoot, topic, text)
	}
	return 0, nil
}



