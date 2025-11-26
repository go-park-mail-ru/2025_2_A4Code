package message

import (
	"2025_2_a4code/internal/domain"
	"context"
	"errors"
	"reflect"
	"testing"
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

func TestNew(t *testing.T) {
	mockRepo := &MockMessageRepository{}
	uc := New(mockRepo)

	if uc == nil {
		t.Error("New() returned nil")
	}

	if uc.repo != mockRepo {
		t.Error("New() didn't set repository correctly")
	}
}

func TestMessageUcase_FindFullByMessageID(t *testing.T) {
	expectedFullMessage := domain.FullMessage{
		ID:    "1",
		Topic: "Test",
		Text:  "Test message text",
	}

	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		messageID int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    domain.FullMessage
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					FindFullByMessageIDFn: func(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error) {
						return expectedFullMessage, nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			want:    expectedFullMessage,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					FindFullByMessageIDFn: func(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error) {
						return domain.FullMessage{}, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			want:    domain.FullMessage{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.FindFullByMessageID(tt.args.ctx, tt.args.messageID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindFullByMessageID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindFullByMessageID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_SaveMessage(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx                  context.Context
		receiverProfileEmail string
		senderBaseProfileID  int64
		topic                string
		text                 string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					SaveMessageFn: func(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error) {
						return 123, nil
					},
				},
			},
			args:    args{ctx: context.Background(), receiverProfileEmail: "test@test.com", senderBaseProfileID: 1, topic: "Test", text: "Hello"},
			want:    123,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					SaveMessageFn: func(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error) {
						return 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), receiverProfileEmail: "test@test.com", senderBaseProfileID: 1, topic: "Test", text: "Hello"},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.SaveMessage(tt.args.ctx, tt.args.receiverProfileEmail, tt.args.senderBaseProfileID, tt.args.topic, tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SaveMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_SaveFile(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx         context.Context
		messageID   int64
		fileName    string
		fileType    string
		storagePath string
		size        int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					SaveFileFn: func(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error) {
						return 456, nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, fileName: "test.txt", fileType: "text/plain", storagePath: "/path/to/file", size: 1024},
			want:    456,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					SaveFileFn: func(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error) {
						return 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, fileName: "test.txt", fileType: "text/plain", storagePath: "/path/to/file", size: 1024},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.SaveFile(tt.args.ctx, tt.args.messageID, tt.args.fileName, tt.args.fileType, tt.args.storagePath, tt.args.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SaveFile() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_SaveThread(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		messageID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					SaveThreadFn: func(ctx context.Context, messageID int64) (threadID int64, err error) {
						return 789, nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1},
			want:    789,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					SaveThreadFn: func(ctx context.Context, messageID int64) (threadID int64, err error) {
						return 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.SaveThread(tt.args.ctx, tt.args.messageID)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveThread() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SaveThread() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_SaveThreadIdToMessage(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		messageID int64
		threadID  int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					SaveThreadIdToMessageFn: func(ctx context.Context, messageID int64, threadID int64) error {
						return nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, threadID: 1},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					SaveThreadIdToMessageFn: func(ctx context.Context, messageID int64, threadID int64) error {
						return mockError
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, threadID: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			err := uc.SaveThreadIdToMessage(tt.args.ctx, tt.args.messageID, tt.args.threadID)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveThreadIdToMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUcase_FindThreadsByProfileID(t *testing.T) {
	expectedThreads := []domain.ThreadInfo{
		{ID: 1, RootMessage: 1, LastActivity: time.Now()},
		{ID: 2, RootMessage: 2, LastActivity: time.Now()},
	}

	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domain.ThreadInfo
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					FindThreadsByProfileIDFn: func(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error) {
						return expectedThreads, nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1},
			want:    expectedThreads,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					FindThreadsByProfileIDFn: func(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error) {
						return nil, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.FindThreadsByProfileID(tt.args.ctx, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("FindThreadsByProfileID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FindThreadsByProfileID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_MarkMessageAsRead(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		messageID int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					MarkMessageAsReadFn: func(ctx context.Context, messageID int64, profileID int64) error {
						return nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					MarkMessageAsReadFn: func(ctx context.Context, messageID int64, profileID int64) error {
						return mockError
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			err := uc.MarkMessageAsRead(tt.args.ctx, tt.args.messageID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarkMessageAsRead() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUcase_MarkMessageAsSpam(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		messageID int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					MarkMessageAsSpamFn: func(ctx context.Context, messageID int64, profileID int64) error {
						return nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					MarkMessageAsSpamFn: func(ctx context.Context, messageID int64, profileID int64) error {
						return mockError
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			err := uc.MarkMessageAsSpam(tt.args.ctx, tt.args.messageID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarkMessageAsSpam() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUcase_SaveDraft(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx           context.Context
		profileID     int64
		draftID       string
		receiverEmail string
		topic         string
		text          string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					SaveDraftFn: func(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error) {
						return 111, nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, draftID: "draft1", receiverEmail: "test@test.com", topic: "Draft", text: "Draft text"},
			want:    111,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					SaveDraftFn: func(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error) {
						return 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, draftID: "draft1", receiverEmail: "test@test.com", topic: "Draft", text: "Draft text"},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.SaveDraft(tt.args.ctx, tt.args.profileID, tt.args.draftID, tt.args.receiverEmail, tt.args.topic, tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveDraft() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SaveDraft() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_IsDraftBelongsToUser(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		draftID   int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Success_True",
			fields: fields{
				repo: &MockMessageRepository{
					IsDraftBelongsToUserFn: func(ctx context.Context, draftID, profileID int64) (bool, error) {
						return true, nil
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			want:    true,
			wantErr: false,
		},
		{
			name: "Success_False",
			fields: fields{
				repo: &MockMessageRepository{
					IsDraftBelongsToUserFn: func(ctx context.Context, draftID, profileID int64) (bool, error) {
						return false, nil
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			want:    false,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					IsDraftBelongsToUserFn: func(ctx context.Context, draftID, profileID int64) (bool, error) {
						return false, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.IsDraftBelongsToUser(tt.args.ctx, tt.args.draftID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsDraftBelongsToUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsDraftBelongsToUser() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_DeleteDraft(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		draftID   int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					DeleteDraftFn: func(ctx context.Context, draftID, profileID int64) error {
						return nil
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					DeleteDraftFn: func(ctx context.Context, draftID, profileID int64) error {
						return mockError
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			err := uc.DeleteDraft(tt.args.ctx, tt.args.draftID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteDraft() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUcase_SendDraft(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		draftID   int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					SendDraftFn: func(ctx context.Context, draftID, profileID int64) error {
						return nil
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					SendDraftFn: func(ctx context.Context, draftID, profileID int64) error {
						return mockError
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			err := uc.SendDraft(tt.args.ctx, tt.args.draftID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendDraft() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUcase_GetDraft(t *testing.T) {
	expectedDraft := domain.FullMessage{
		ID:    "draft1",
		Topic: "Draft",
		Text:  "Draft text",
	}

	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		draftID   int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    domain.FullMessage
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					GetDraftFn: func(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error) {
						return expectedDraft, nil
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			want:    expectedDraft,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					GetDraftFn: func(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error) {
						return domain.FullMessage{}, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), draftID: 1, profileID: 1},
			want:    domain.FullMessage{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.GetDraft(tt.args.ctx, tt.args.draftID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDraft() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDraft() got = %v, want %v", got, tt.want)
			}
		})
	}
}
func TestMessageUcase_MoveToFolder(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		profileID int64
		messageID int64
		folderID  int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					MoveToFolderFn: func(ctx context.Context, profileID, messageID, folderID int64) error {
						return nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, messageID: 1, folderID: 1},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					MoveToFolderFn: func(ctx context.Context, profileID, messageID, folderID int64) error {
						return mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, messageID: 1, folderID: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			err := uc.MoveToFolder(tt.args.ctx, tt.args.profileID, tt.args.messageID, tt.args.folderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("MoveToFolder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUcase_GetFolderByType(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx        context.Context
		profileID  int64
		folderType string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					GetFolderByTypeFn: func(ctx context.Context, profileID int64, folderType string) (int64, error) {
						return 123, nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderType: "inbox"},
			want:    123,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					GetFolderByTypeFn: func(ctx context.Context, profileID int64, folderType string) (int64, error) {
						return 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderType: "inbox"},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.GetFolderByType(tt.args.ctx, tt.args.profileID, tt.args.folderType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFolderByType() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetFolderByType() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_ShouldMarkAsRead(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		messageID int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Success_True",
			fields: fields{
				repo: &MockMessageRepository{
					ShouldMarkAsReadFn: func(ctx context.Context, messageID, profileID int64) (bool, error) {
						return true, nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			want:    true,
			wantErr: false,
		},
		{
			name: "Success_False",
			fields: fields{
				repo: &MockMessageRepository{
					ShouldMarkAsReadFn: func(ctx context.Context, messageID, profileID int64) (bool, error) {
						return false, nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			want:    false,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					ShouldMarkAsReadFn: func(ctx context.Context, messageID, profileID int64) (bool, error) {
						return false, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.ShouldMarkAsRead(tt.args.ctx, tt.args.messageID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ShouldMarkAsRead() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ShouldMarkAsRead() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_CreateFolder(t *testing.T) {
	expectedFolder := &domain.Folder{ID: 1, Name: "Test Folder"}

	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx        context.Context
		profileID  int64
		folderName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.Folder
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					CreateFolderFn: func(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error) {
						return expectedFolder, nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderName: "Test Folder"},
			want:    expectedFolder,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					CreateFolderFn: func(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error) {
						return nil, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderName: "Test Folder"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.CreateFolder(tt.args.ctx, tt.args.profileID, tt.args.folderName)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFolder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateFolder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_GetUserFolders(t *testing.T) {
	expectedFolders := []domain.Folder{
		{ID: 1, Name: "Inbox"},
		{ID: 2, Name: "Sent"},
	}

	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domain.Folder
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					GetUserFoldersFn: func(ctx context.Context, profileID int64) ([]domain.Folder, error) {
						return expectedFolders, nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1},
			want:    expectedFolders,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					GetUserFoldersFn: func(ctx context.Context, profileID int64) ([]domain.Folder, error) {
						return nil, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.GetUserFolders(tt.args.ctx, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserFolders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUserFolders() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_RenameFolder(t *testing.T) {
	expectedFolder := &domain.Folder{ID: 1, Name: "Renamed Folder"}

	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		profileID int64
		folderID  int64
		newName   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.Folder
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					RenameFolderFn: func(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error) {
						return expectedFolder, nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderID: 1, newName: "Renamed Folder"},
			want:    expectedFolder,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					RenameFolderFn: func(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error) {
						return nil, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderID: 1, newName: "Renamed Folder"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.RenameFolder(tt.args.ctx, tt.args.profileID, tt.args.folderID, tt.args.newName)
			if (err != nil) != tt.wantErr {
				t.Errorf("RenameFolder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RenameFolder() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_DeleteFolder(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		profileID int64
		folderID  int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					DeleteFolderFn: func(ctx context.Context, profileID, folderID int64) error {
						return nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderID: 1},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					DeleteFolderFn: func(ctx context.Context, profileID, folderID int64) error {
						return mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderID: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			err := uc.DeleteFolder(tt.args.ctx, tt.args.profileID, tt.args.folderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteFolder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUcase_DeleteMessageFromFolder(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		profileID int64
		messageID int64
		folderID  int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					DeleteMessageFromFolderFn: func(ctx context.Context, profileID, messageID, folderID int64) error {
						return nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, messageID: 1, folderID: 1},
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					DeleteMessageFromFolderFn: func(ctx context.Context, profileID, messageID, folderID int64) error {
						return mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, messageID: 1, folderID: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			err := uc.DeleteMessageFromFolder(tt.args.ctx, tt.args.profileID, tt.args.messageID, tt.args.folderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteMessageFromFolder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageUcase_GetFolderMessagesWithKeysetPagination(t *testing.T) {
	expectedMessages := []domain.Message{
		{ID: "1", Topic: "Message 1"},
		{ID: "2", Topic: "Message 2"},
	}

	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx           context.Context
		profileID     int64
		folderID      int64
		lastMessageID int64
		lastDatetime  time.Time
		limit         int
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []domain.Message
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					GetFolderMessagesWithKeysetPaginationFn: func(ctx context.Context, profileID, folderID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error) {
						return expectedMessages, nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderID: 1, lastMessageID: 0, lastDatetime: time.Time{}, limit: 10},
			want:    expectedMessages,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					GetFolderMessagesWithKeysetPaginationFn: func(ctx context.Context, profileID, folderID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error) {
						return nil, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderID: 1, lastMessageID: 0, lastDatetime: time.Time{}, limit: 10},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.GetFolderMessagesWithKeysetPagination(tt.args.ctx, tt.args.profileID, tt.args.folderID, tt.args.lastMessageID, tt.args.lastDatetime, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFolderMessagesWithKeysetPagination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFolderMessagesWithKeysetPagination() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_GetFolderMessagesInfo(t *testing.T) {
	expectedMessagesInfo := domain.Messages{
		MessageTotal:  5,
		MessageUnread: 2,
		Messages:      []domain.Message{},
	}

	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		profileID int64
		folderID  int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    domain.Messages
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					GetFolderMessagesInfoFn: func(ctx context.Context, profileID, folderID int64) (domain.Messages, error) {
						return expectedMessagesInfo, nil
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderID: 1},
			want:    expectedMessagesInfo,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					GetFolderMessagesInfoFn: func(ctx context.Context, profileID, folderID int64) (domain.Messages, error) {
						return domain.Messages{}, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1, folderID: 1},
			want:    domain.Messages{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.GetFolderMessagesInfo(tt.args.ctx, tt.args.profileID, tt.args.folderID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFolderMessagesInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFolderMessagesInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_SendMessage(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx             context.Context
		receiverEmail   string
		senderProfileID int64
		topic           string
		text            string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					SaveMessageWithFolderDistributionFn: func(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error) {
						return 999, nil
					},
				},
			},
			args:    args{ctx: context.Background(), receiverEmail: "test@test.com", senderProfileID: 1, topic: "Test", text: "Hello"},
			want:    999,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					SaveMessageWithFolderDistributionFn: func(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error) {
						return 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), receiverEmail: "test@test.com", senderProfileID: 1, topic: "Test", text: "Hello"},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.SendMessage(tt.args.ctx, tt.args.receiverEmail, tt.args.senderProfileID, tt.args.topic, tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("SendMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SendMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_ReplyToMessage(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx             context.Context
		receiverEmail   string
		senderProfileID int64
		threadRoot      int64
		topic           string
		text            string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		{
			name: "Success",
			fields: fields{
				repo: &MockMessageRepository{
					ReplyToMessageWithFolderDistributionFn: func(ctx context.Context, receiverEmail string, senderProfileID int64, threadRoot int64, topic, text string) (int64, error) {
						return 888, nil
					},
				},
			},
			args:    args{ctx: context.Background(), receiverEmail: "test@test.com", senderProfileID: 1, threadRoot: 1, topic: "Re: Test", text: "Reply"},
			want:    888,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					ReplyToMessageWithFolderDistributionFn: func(ctx context.Context, receiverEmail string, senderProfileID int64, threadRoot int64, topic, text string) (int64, error) {
						return 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), receiverEmail: "test@test.com", senderProfileID: 1, threadRoot: 1, topic: "Re: Test", text: "Reply"},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.ReplyToMessage(tt.args.ctx, tt.args.receiverEmail, tt.args.senderProfileID, tt.args.threadRoot, tt.args.topic, tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplyToMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReplyToMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_IsUsersMessage(t *testing.T) {
	type fields struct {
		repo MessageRepository
	}
	type args struct {
		ctx       context.Context
		messageID int64
		profileID int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Success_True",
			fields: fields{
				repo: &MockMessageRepository{
					IsUsersMessageFn: func(ctx context.Context, messageID int64, profileID int64) (bool, error) {
						return true, nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			want:    true,
			wantErr: false,
		},
		{
			name: "Success_False",
			fields: fields{
				repo: &MockMessageRepository{
					IsUsersMessageFn: func(ctx context.Context, messageID int64, profileID int64) (bool, error) {
						return false, nil
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			want:    false,
			wantErr: false,
		},
		{
			name: "Failure",
			fields: fields{
				repo: &MockMessageRepository{
					IsUsersMessageFn: func(ctx context.Context, messageID int64, profileID int64) (bool, error) {
						return false, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), messageID: 1, profileID: 1},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.IsUsersMessage(tt.args.ctx, tt.args.messageID, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsUsersMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsUsersMessage() got = %v, want %v", got, tt.want)
			}
		})
	}
}
