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

func TestMessageUcase_GetMessagesInfo(t *testing.T) {
	testMessages := []domain.Message{
		{ID: "1", IsRead: true},
		{ID: "2", IsRead: false},
		{ID: "3", IsRead: false},
		{ID: "4", IsRead: true},
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
		want    domain.Messages
		wantErr bool
	}{
		{
			name: "Success_MixedMessages",
			fields: fields{
				repo: &MockMessageRepository{
					FindByProfileIDFn: func(ctx context.Context, profileID int64) ([]domain.Message, error) {
						return testMessages, nil
					},
				},
			},
			args: args{ctx: context.Background(), profileID: 1},
			want: domain.Messages{
				MessageTotal:  4,
				MessageUnread: 2,
				Messages:      testMessages,
			},
			wantErr: false,
		},
		{
			name: "Success_AllReadMessages",
			fields: fields{
				repo: &MockMessageRepository{
					FindByProfileIDFn: func(ctx context.Context, profileID int64) ([]domain.Message, error) {
						return []domain.Message{{ID: "1", IsRead: true}, {ID: "2", IsRead: true}}, nil
					},
				},
			},
			args: args{ctx: context.Background(), profileID: 1},
			want: domain.Messages{
				MessageTotal:  2,
				MessageUnread: 0,
				Messages:      []domain.Message{{ID: "1", IsRead: true}, {ID: "2", IsRead: true}},
			},
			wantErr: false,
		},
		{
			name: "Success_NoMessages",
			fields: fields{
				repo: &MockMessageRepository{
					FindByProfileIDFn: func(ctx context.Context, profileID int64) ([]domain.Message, error) {
						return []domain.Message{}, nil
					},
				},
			},
			args: args{ctx: context.Background(), profileID: 1},
			want: domain.Messages{
				MessageTotal:  0,
				MessageUnread: 0,
				Messages:      []domain.Message{},
			},
			wantErr: false,
		},
		{
			name: "Failure_RepoError",
			fields: fields{
				repo: &MockMessageRepository{
					FindByProfileIDFn: func(ctx context.Context, profileID int64) ([]domain.Message, error) {
						return nil, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1},
			want:    domain.Messages{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.GetMessagesInfo(tt.args.ctx, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMessagesInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMessagesInfo() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_GetMessagesInfoWithPagination(t *testing.T) {
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
		want    domain.Messages
		wantErr bool
	}{
		{
			name: "Success_WithStats",
			fields: fields{
				repo: &MockMessageRepository{
					GetMessagesStatsFn: func(ctx context.Context, profileID int64) (int, int, error) {
						return 100, 5, nil // 100 total, 5 unread
					},
				},
			},
			args: args{ctx: context.Background(), profileID: 1},
			want: domain.Messages{
				MessageTotal:  100,
				MessageUnread: 5,
				Messages:      nil,
			},
			wantErr: false,
		},
		{
			name: "Success_ZeroStats",
			fields: fields{
				repo: &MockMessageRepository{
					GetMessagesStatsFn: func(ctx context.Context, profileID int64) (int, int, error) {
						return 0, 0, nil
					},
				},
			},
			args: args{ctx: context.Background(), profileID: 1},
			want: domain.Messages{
				MessageTotal:  0,
				MessageUnread: 0,
				Messages:      nil,
			},
			wantErr: false,
		},
		{
			name: "Failure_RepoError",
			fields: fields{
				repo: &MockMessageRepository{
					GetMessagesStatsFn: func(ctx context.Context, profileID int64) (int, int, error) {
						return 0, 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1},
			want:    domain.Messages{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.GetMessagesInfoWithPagination(tt.args.ctx, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetMessagesInfoWithPagination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMessagesInfoWithPagination() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageUcase_GetSentMessagesInfoWithPagination(t *testing.T) {
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
		want    domain.Messages
		wantErr bool
	}{
		{
			name: "Success_WithStats",
			fields: fields{
				repo: &MockMessageRepository{
					GetSentMessagesStatsFn: func(ctx context.Context, profileID int64) (int, int, error) {
						return 50, 0, nil
					},
				},
			},
			args: args{ctx: context.Background(), profileID: 1},
			want: domain.Messages{
				MessageTotal:  50,
				MessageUnread: 0,
				Messages:      nil,
			},
			wantErr: false,
		},
		{
			name: "Success_ZeroStats",
			fields: fields{
				repo: &MockMessageRepository{
					GetSentMessagesStatsFn: func(ctx context.Context, profileID int64) (int, int, error) {
						return 0, 0, nil
					},
				},
			},
			args: args{ctx: context.Background(), profileID: 1},
			want: domain.Messages{
				MessageTotal:  0,
				MessageUnread: 0,
				Messages:      nil,
			},
			wantErr: false,
		},
		{
			name: "Failure_RepoError",
			fields: fields{
				repo: &MockMessageRepository{
					GetSentMessagesStatsFn: func(ctx context.Context, profileID int64) (int, int, error) {
						return 0, 0, mockError
					},
				},
			},
			args:    args{ctx: context.Background(), profileID: 1},
			want:    domain.Messages{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &MessageUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.GetSentMessagesInfoWithPagination(tt.args.ctx, tt.args.profileID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSentMessagesInfoWithPagination() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSentMessagesInfoWithPagination() got = %v, want %v", got, tt.want)
			}
		})
	}
}
