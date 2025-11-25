package message

import (
	"2025_2_a4code/internal/domain"
	"context"
	"time"
)

type MessageRepository interface {
	// базовые методы для сообщений
	FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error)
	FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error)
	SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error)
	SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error)

	// методы для тредов
	SaveThread(ctx context.Context, messageID int64) (threadID int64, err error)
	SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error
	FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error)

	// методы для работы с сообщениями
	MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error
	MarkMessageAsSpam(ctx context.Context, messageID int64, profileID int64) error
	IsUsersMessage(ctx context.Context, messageID int64, profileID int64) (bool, error)

	// методы для черновиков
	SaveDraft(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error)
	IsDraftBelongsToUser(ctx context.Context, draftID, profileID int64) (bool, error)
	DeleteDraft(ctx context.Context, draftID, profileID int64) error
	SendDraft(ctx context.Context, draftID, profileID int64) error
	GetDraft(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error)

	// методы для папок
	MoveToFolder(ctx context.Context, profileID, messageID, folderID int64) error
	GetFolderByType(ctx context.Context, profileID int64, folderType string) (int64, error)
	ShouldMarkAsRead(ctx context.Context, messageID, profileID int64) (bool, error)
	CreateFolder(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error)
	GetUserFolders(ctx context.Context, profileID int64) ([]domain.Folder, error)
	RenameFolder(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error)
	DeleteFolder(ctx context.Context, profileID, folderID int64) error
	DeleteMessageFromFolder(ctx context.Context, profileID, messageID, folderID int64) error
	GetFolderMessagesWithKeysetPagination(ctx context.Context, profileID, folderID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error)
	GetFolderMessagesInfo(ctx context.Context, profileID, folderID int64) (domain.Messages, error)

	// методы для отправки сообщений с автоматическим распределением по папкам
	SaveMessageWithFolderDistribution(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error)
	ReplyToMessageWithFolderDistribution(ctx context.Context, receiverEmail string, senderProfileID int64, threadRoot int64, topic, text string) (int64, error)
}

type MessageUcase struct {
	repo MessageRepository
}

func New(repo MessageRepository) *MessageUcase {
	return &MessageUcase{repo: repo}
}

// базовые методы для сообщений
func (uc *MessageUcase) FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error) {
	return uc.repo.FindByMessageID(ctx, messageID)
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

// методы для тредов
func (uc *MessageUcase) SaveThread(ctx context.Context, messageID int64) (threadID int64, err error) {
	return uc.repo.SaveThread(ctx, messageID)
}

func (uc *MessageUcase) SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error {
	return uc.repo.SaveThreadIdToMessage(ctx, messageID, threadID)
}

func (uc *MessageUcase) FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error) {
	return uc.repo.FindThreadsByProfileID(ctx, profileID)
}

// методы для работы с сообщениями
func (uc *MessageUcase) MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error {
	return uc.repo.MarkMessageAsRead(ctx, messageID, profileID)
}

func (uc *MessageUcase) MarkMessageAsSpam(ctx context.Context, messageID int64, profileID int64) error {
	return uc.repo.MarkMessageAsSpam(ctx, messageID, profileID)
}

// методы для черновиков
func (uc *MessageUcase) SaveDraft(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error) {
	return uc.repo.SaveDraft(ctx, profileID, draftID, receiverEmail, topic, text)
}

func (uc *MessageUcase) IsDraftBelongsToUser(ctx context.Context, draftID, profileID int64) (bool, error) {
	return uc.repo.IsDraftBelongsToUser(ctx, draftID, profileID)
}

func (uc *MessageUcase) DeleteDraft(ctx context.Context, draftID, profileID int64) error {
	return uc.repo.DeleteDraft(ctx, draftID, profileID)
}

func (uc *MessageUcase) SendDraft(ctx context.Context, draftID, profileID int64) error {
	return uc.repo.SendDraft(ctx, draftID, profileID)
}

func (uc *MessageUcase) GetDraft(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error) {
	return uc.repo.GetDraft(ctx, draftID, profileID)
}

// методы для папок
func (uc *MessageUcase) MoveToFolder(ctx context.Context, profileID, messageID, folderID int64) error {
	return uc.repo.MoveToFolder(ctx, profileID, messageID, folderID)
}

func (uc *MessageUcase) GetFolderByType(ctx context.Context, profileID int64, folderType string) (int64, error) {
	return uc.repo.GetFolderByType(ctx, profileID, folderType)
}

func (uc *MessageUcase) ShouldMarkAsRead(ctx context.Context, messageID, profileID int64) (bool, error) {
	return uc.repo.ShouldMarkAsRead(ctx, messageID, profileID)
}

func (uc *MessageUcase) CreateFolder(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error) {
	return uc.repo.CreateFolder(ctx, profileID, folderName)
}

func (uc *MessageUcase) GetUserFolders(ctx context.Context, profileID int64) ([]domain.Folder, error) {
	return uc.repo.GetUserFolders(ctx, profileID)
}

func (uc *MessageUcase) RenameFolder(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error) {
	return uc.repo.RenameFolder(ctx, profileID, folderID, newName)
}

func (uc *MessageUcase) DeleteFolder(ctx context.Context, profileID, folderID int64) error {
	return uc.repo.DeleteFolder(ctx, profileID, folderID)
}

func (uc *MessageUcase) DeleteMessageFromFolder(ctx context.Context, profileID, messageID, folderID int64) error {
	return uc.repo.DeleteMessageFromFolder(ctx, profileID, messageID, folderID)
}

func (uc *MessageUcase) GetFolderMessagesWithKeysetPagination(ctx context.Context, profileID, folderID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error) {
	return uc.repo.GetFolderMessagesWithKeysetPagination(ctx, profileID, folderID, lastMessageID, lastDatetime, limit)
}

func (uc *MessageUcase) GetFolderMessagesInfo(ctx context.Context, profileID, folderID int64) (domain.Messages, error) {
	return uc.repo.GetFolderMessagesInfo(ctx, profileID, folderID)
}

func (uc *MessageUcase) SendMessage(ctx context.Context, receiverEmail string, senderProfileID int64, topic, text string) (int64, error) {
	return uc.repo.SaveMessageWithFolderDistribution(ctx, receiverEmail, senderProfileID, topic, text)
}

func (uc *MessageUcase) ReplyToMessage(ctx context.Context, receiverEmail string, senderProfileID int64, threadRoot int64, topic, text string) (int64, error) {
	return uc.repo.ReplyToMessageWithFolderDistribution(ctx, receiverEmail, senderProfileID, threadRoot, topic, text)
}

func (uc *MessageUcase) IsUsersMessage(ctx context.Context, messageID int64, profileID int64) (bool, error) {
	return uc.repo.IsUsersMessage(ctx, messageID, profileID)
}
