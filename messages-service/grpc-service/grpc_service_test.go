package messages_service

import (
	"2025_2_a4code/internal/domain"
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	pb "2025_2_a4code/messages-service/pkg/messagesproto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
)

type MockMessageUsecase struct {
	mock.Mock
}

func (m *MockMessageUsecase) FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error) {
	args := m.Called(ctx, messageID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Message), args.Error(1)
}

func (m *MockMessageUsecase) FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error) {
	args := m.Called(ctx, messageID, profileID)
	return args.Get(0).(domain.FullMessage), args.Error(1)
}

func (m *MockMessageUsecase) SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (int64, error) {
	args := m.Called(ctx, receiverProfileEmail, senderBaseProfileID, topic, text)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUsecase) SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (int64, error) {
	args := m.Called(ctx, messageID, fileName, fileType, storagePath, size)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUsecase) SaveThread(ctx context.Context, messageID int64) (int64, error) {
	args := m.Called(ctx, messageID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUsecase) SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error {
	args := m.Called(ctx, messageID, threadID)
	return args.Error(0)
}

func (m *MockMessageUsecase) FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error) {
	args := m.Called(ctx, profileID)
	return args.Get(0).([]domain.ThreadInfo), args.Error(1)
}

func (m *MockMessageUsecase) MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error {
	args := m.Called(ctx, messageID, profileID)
	return args.Error(0)
}

func (m *MockMessageUsecase) MarkMessageAsSpam(ctx context.Context, messageID int64, profileID int64) error {
	args := m.Called(ctx, messageID, profileID)
	return args.Error(0)
}

func (m *MockMessageUsecase) IsUsersMessage(ctx context.Context, messageID int64, profileID int64) (bool, error) {
	args := m.Called(ctx, messageID, profileID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMessageUsecase) SaveDraft(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error) {
	args := m.Called(ctx, profileID, draftID, receiverEmail, topic, text)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUsecase) IsDraftBelongsToUser(ctx context.Context, draftID, profileID int64) (bool, error) {
	args := m.Called(ctx, draftID, profileID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMessageUsecase) DeleteDraft(ctx context.Context, draftID, profileID int64) error {
	args := m.Called(ctx, draftID, profileID)
	return args.Error(0)
}

func (m *MockMessageUsecase) SendDraft(ctx context.Context, draftID, profileID int64) error {
	args := m.Called(ctx, draftID, profileID)
	return args.Error(0)
}

func (m *MockMessageUsecase) GetDraft(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error) {
	args := m.Called(ctx, draftID, profileID)
	return args.Get(0).(domain.FullMessage), args.Error(1)
}

func (m *MockMessageUsecase) MoveToFolder(ctx context.Context, profileID, messageID, folderID int64) error {
	args := m.Called(ctx, profileID, messageID, folderID)
	return args.Error(0)
}

func (m *MockMessageUsecase) GetFolderByType(ctx context.Context, profileID int64, folderType string) (int64, error) {
	args := m.Called(ctx, profileID, folderType)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUsecase) ShouldMarkAsRead(ctx context.Context, messageID, profileID int64) (bool, error) {
	args := m.Called(ctx, messageID, profileID)
	return args.Bool(0), args.Error(1)
}

func (m *MockMessageUsecase) CreateFolder(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error) {
	args := m.Called(ctx, profileID, folderName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Folder), args.Error(1)
}

func (m *MockMessageUsecase) GetUserFolders(ctx context.Context, profileID int64) ([]domain.Folder, error) {
	args := m.Called(ctx, profileID)
	return args.Get(0).([]domain.Folder), args.Error(1)
}

func (m *MockMessageUsecase) RenameFolder(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error) {
	args := m.Called(ctx, profileID, folderID, newName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Folder), args.Error(1)
}

func (m *MockMessageUsecase) DeleteFolder(ctx context.Context, profileID, folderID int64) error {
	args := m.Called(ctx, profileID, folderID)
	return args.Error(0)
}

func (m *MockMessageUsecase) DeleteMessageFromFolder(ctx context.Context, profileID, messageID, folderID int64) error {
	args := m.Called(ctx, profileID, messageID, folderID)
	return args.Error(0)
}

func (m *MockMessageUsecase) GetFolderMessagesWithKeysetPagination(ctx context.Context, profileID, folderID, lastMessageID int64, lastDatetime time.Time, limit int) ([]domain.Message, error) {
	args := m.Called(ctx, profileID, folderID, lastMessageID, lastDatetime, limit)
	return args.Get(0).([]domain.Message), args.Error(1)
}

func (m *MockMessageUsecase) GetFolderMessagesInfo(ctx context.Context, profileID, folderID int64) (domain.Messages, error) {
	args := m.Called(ctx, profileID, folderID)
	return args.Get(0).(domain.Messages), args.Error(1)
}

func (m *MockMessageUsecase) SendMessage(ctx context.Context, receiverEmail string, senderProfileID int64, topic, text string) (int64, error) {
	args := m.Called(ctx, receiverEmail, senderProfileID, topic, text)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUsecase) ReplyToMessage(ctx context.Context, receiverEmail string, senderProfileID int64, threadRoot int64, topic, text string) (int64, error) {
	args := m.Called(ctx, receiverEmail, senderProfileID, threadRoot, topic, text)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUsecase) SaveOutgoingExternalMessage(ctx context.Context, senderProfileID int64, topic, text string) (int64, error) {
	args := m.Called(ctx, senderProfileID, topic, text)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockMessageUsecase) GetProfileEmail(ctx context.Context, profileID int64) (string, error) {
	args := m.Called(ctx, profileID)
	return args.String(0), args.Error(1)
}

type MockAvatarUsecase struct {
	mock.Mock
}

func (m *MockAvatarUsecase) GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
	args := m.Called(ctx, objectName, duration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*url.URL), args.Error(1)
}

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrFolderExists    = errors.New("folder already exists")
	ErrFolderNotFound  = errors.New("folder not found")
	ErrFolderSystem    = errors.New("system folder cannot be deleted")
)

func setupTestServer() (*Server, *MockMessageUsecase, *MockAvatarUsecase) {
	mockMessageUsecase := &MockMessageUsecase{}
	mockAvatarUsecase := &MockAvatarUsecase{}
	jwtSecret := []byte("test-secret-key-very-long-for-testing")
	server := New(mockMessageUsecase, mockAvatarUsecase, jwtSecret, nil, "attachments")
	return server, mockMessageUsecase, mockAvatarUsecase
}

func generateTestToken(userID int64, secret []byte, tokenType string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    tokenType,
	})

	return token.SignedString(secret)
}

func createTestContextWithToken(userID int64, secret []byte) context.Context {
	token, err := generateTestToken(userID, secret, "access")
	if err != nil {
		panic(err)
	}
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	return metadata.NewIncomingContext(context.Background(), md)
}

func createTestContextWithoutAuth() context.Context {
	return context.Background()
}

func createTestContextWithInvalidToken() context.Context {
	md := metadata.New(map[string]string{"authorization": "Bearer invalid-token"})
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestServer_validateFolderName(t *testing.T) {
	server, _, _ := setupTestServer()

	tests := []struct {
		name        string
		folderName  string
		expectError bool
	}{
		{
			name:        "ValidFolderName",
			folderName:  "Test Folder",
			expectError: false,
		},
		{
			name:        "EmptyFolderName",
			folderName:  "",
			expectError: true,
		},
		{
			name:        "FolderNameTooLong",
			folderName:  "This is a very long folder name that exceeds the maximum allowed length of fifty characters",
			expectError: true,
		},
		{
			name:        "SystemFolderNameInbox",
			folderName:  "inbox",
			expectError: true,
		},
		{
			name:        "SystemFolderNameSent",
			folderName:  "sent",
			expectError: true,
		},
		{
			name:        "SystemFolderNameDraft",
			folderName:  "draft",
			expectError: true,
		},
		{
			name:        "SystemFolderNameSpam",
			folderName:  "spam",
			expectError: true,
		},
		{
			name:        "SystemFolderNameTrash",
			folderName:  "trash",
			expectError: true,
		},
		{
			name:        "SystemFolderNameCustom",
			folderName:  "custom",
			expectError: true,
		},
		{
			name:        "FolderNameWithDangerousCharacters",
			folderName:  "folder<script>",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := server.validateFolderName(tt.folderName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestServer_enrichSenderAvatar(t *testing.T) {
	server, _, mockAvatar := setupTestServer()

	tests := []struct {
		name          string
		sender        *domain.Sender
		mockSetup     func()
		expectedError bool
	}{
		{
			name: "Success",
			sender: &domain.Sender{
				Avatar: "avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
					Scheme: "https",
					Host:   "example.com",
					Path:   "/avatar.jpg",
				}, nil)
			},
			expectedError: false,
		},
		{
			name: "EmptyAvatar",
			sender: &domain.Sender{
				Avatar: "",
			},
			mockSetup:     func() {},
			expectedError: false,
		},
		{
			name:          "NilSender",
			sender:        nil,
			mockSetup:     func() {},
			expectedError: false,
		},
		{
			name: "AvatarWithHTTPPrefix",
			sender: &domain.Sender{
				Avatar: "http://example.com/avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
					Scheme: "https",
					Host:   "example.com",
					Path:   "/avatar.jpg",
				}, nil)
			},
			expectedError: false,
		},
		{
			name: "AvatarWithHTTPSPrefix",
			sender: &domain.Sender{
				Avatar: "https://example.com/avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
					Scheme: "https",
					Host:   "example.com",
					Path:   "/avatar.jpg",
				}, nil)
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := server.enrichSenderAvatar(context.Background(), tt.sender)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.sender != nil && tt.sender.Avatar != "" && !tt.expectedError {
					assert.Equal(t, "https://example.com/avatar.jpg", tt.sender.Avatar)
				}
			}

			mockAvatar.AssertExpectations(t)
		})
	}
}

func BenchmarkServer_MessagePage(b *testing.B) {
	server, mockMessage, mockAvatar := setupTestServer()

	token, _ := generateTestToken(1, server.JWTSecret, "access")
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	mockMessage.On("IsUsersMessage", mock.Anything, int64(123), int64(1)).Return(true, nil)
	mockMessage.On("FindFullByMessageID", mock.Anything, int64(123), int64(1)).Return(domain.FullMessage{
		Topic:      "Test Topic",
		Text:       "Test Text",
		Datetime:   time.Now(),
		ThreadRoot: "456",
		Sender: domain.Sender{
			Email:    "sender@example.com",
			Username: "sender",
			Avatar:   "avatar.jpg",
		},
		Files: []domain.File{},
	}, nil)
	mockMessage.On("MarkMessageAsRead", mock.Anything, int64(123), int64(1)).Return(nil)
	mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
		Scheme: "https",
		Host:   "example.com",
		Path:   "/avatar.jpg",
	}, nil)

	req := &pb.MessagePageRequest{
		MessageId: "123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.MessagePage(ctx, req)
	}
}

func BenchmarkServer_Send(b *testing.B) {
	server, mockMessage, _ := setupTestServer()

	token, _ := generateTestToken(1, server.JWTSecret, "access")
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	mockMessage.On("GetProfileEmail", mock.Anything, int64(1)).Return("sender@example.com", nil)
	mockMessage.On("SendMessage", mock.Anything, "test@example.com", int64(1), "Test Topic", "Test Message").Return(int64(123), nil)
	mockMessage.On("SaveThread", mock.Anything, int64(123)).Return(int64(456), nil)
	mockMessage.On("SaveThreadIdToMessage", mock.Anything, int64(123), int64(456)).Return(nil)

	req := &pb.SendRequest{
		Topic: "Test Topic",
		Text:  "Test Message",
		Receivers: []*pb.Receiver{
			{Email: "test@example.com"},
		},
		Files: []*pb.File{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.Send(ctx, req)
	}
}
