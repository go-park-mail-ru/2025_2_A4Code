package gateway_service

import (
	"2025_2_a4code/auth-service/pkg/authproto"
	"2025_2_a4code/internal/config"
	"2025_2_a4code/messages-service/pkg/messagesproto"
	"2025_2_a4code/profile-service/pkg/profileproto"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type MockAuthClient struct {
	mock.Mock
}

func (m *MockAuthClient) Login(ctx context.Context, in *authproto.LoginRequest, opts ...grpc.CallOption) (*authproto.LoginResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authproto.LoginResponse), args.Error(1)
}

func (m *MockAuthClient) Signup(ctx context.Context, in *authproto.SignupRequest, opts ...grpc.CallOption) (*authproto.SignupResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authproto.SignupResponse), args.Error(1)
}

func (m *MockAuthClient) Refresh(ctx context.Context, in *authproto.RefreshRequest, opts ...grpc.CallOption) (*authproto.RefreshResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authproto.RefreshResponse), args.Error(1)
}

func (m *MockAuthClient) Logout(ctx context.Context, in *authproto.LogoutRequest, opts ...grpc.CallOption) (*authproto.LogoutResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*authproto.LogoutResponse), args.Error(1)
}

type MockProfileClient struct {
	mock.Mock
}

func (m *MockProfileClient) GetProfile(ctx context.Context, in *profileproto.GetProfileRequest, opts ...grpc.CallOption) (*profileproto.GetProfileResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profileproto.GetProfileResponse), args.Error(1)
}

func (m *MockProfileClient) UpdateProfile(ctx context.Context, in *profileproto.UpdateProfileRequest, opts ...grpc.CallOption) (*profileproto.UpdateProfileResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profileproto.UpdateProfileResponse), args.Error(1)
}

func (m *MockProfileClient) Settings(ctx context.Context, in *profileproto.SettingsRequest, opts ...grpc.CallOption) (*profileproto.SettingsResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profileproto.SettingsResponse), args.Error(1)
}

func (m *MockProfileClient) UploadAvatar(ctx context.Context, in *profileproto.UploadAvatarRequest, opts ...grpc.CallOption) (*profileproto.UploadAvatarResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*profileproto.UploadAvatarResponse), args.Error(1)
}

type MockMessageClient struct {
	mock.Mock
}

func (m *MockMessageClient) Inbox(ctx context.Context, in *messagesproto.InboxRequest, opts ...grpc.CallOption) (*messagesproto.InboxResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.InboxResponse), args.Error(1)
}

func (m *MockMessageClient) MessagePage(ctx context.Context, in *messagesproto.MessagePageRequest, opts ...grpc.CallOption) (*messagesproto.MessagePageResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.MessagePageResponse), args.Error(1)
}

func (m *MockMessageClient) Reply(ctx context.Context, in *messagesproto.ReplyRequest, opts ...grpc.CallOption) (*messagesproto.ReplyResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.ReplyResponse), args.Error(1)
}

func (m *MockMessageClient) Send(ctx context.Context, in *messagesproto.SendRequest, opts ...grpc.CallOption) (*messagesproto.SendResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.SendResponse), args.Error(1)
}

func (m *MockMessageClient) Sent(ctx context.Context, in *messagesproto.SentRequest, opts ...grpc.CallOption) (*messagesproto.SentResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.SentResponse), args.Error(1)
}

func (m *MockMessageClient) MarkAsSpam(ctx context.Context, in *messagesproto.MarkAsSpamRequest, opts ...grpc.CallOption) (*messagesproto.MarkAsSpamResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.MarkAsSpamResponse), args.Error(1)
}

func (m *MockMessageClient) MoveToFolder(ctx context.Context, in *messagesproto.MoveToFolderRequest, opts ...grpc.CallOption) (*messagesproto.MoveToFolderResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.MoveToFolderResponse), args.Error(1)
}

func (m *MockMessageClient) CreateFolder(ctx context.Context, in *messagesproto.CreateFolderRequest, opts ...grpc.CallOption) (*messagesproto.CreateFolderResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.CreateFolderResponse), args.Error(1)
}

func (m *MockMessageClient) GetFolder(ctx context.Context, in *messagesproto.GetFolderRequest, opts ...grpc.CallOption) (*messagesproto.GetFolderResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.GetFolderResponse), args.Error(1)
}

func (m *MockMessageClient) GetFolders(ctx context.Context, in *messagesproto.GetFoldersRequest, opts ...grpc.CallOption) (*messagesproto.GetFoldersResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.GetFoldersResponse), args.Error(1)
}

func (m *MockMessageClient) RenameFolder(ctx context.Context, in *messagesproto.RenameFolderRequest, opts ...grpc.CallOption) (*messagesproto.RenameFolderResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.RenameFolderResponse), args.Error(1)
}

func (m *MockMessageClient) DeleteFolder(ctx context.Context, in *messagesproto.DeleteFolderRequest, opts ...grpc.CallOption) (*messagesproto.DeleteFolderResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.DeleteFolderResponse), args.Error(1)
}

func (m *MockMessageClient) DeleteMessageFromFolder(ctx context.Context, in *messagesproto.DeleteMessageFromFolderRequest, opts ...grpc.CallOption) (*messagesproto.DeleteMessageFromFolderResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.DeleteMessageFromFolderResponse), args.Error(1)
}

func (m *MockMessageClient) SaveDraft(ctx context.Context, in *messagesproto.SaveDraftRequest, opts ...grpc.CallOption) (*messagesproto.SaveDraftResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.SaveDraftResponse), args.Error(1)
}

func (m *MockMessageClient) DeleteDraft(ctx context.Context, in *messagesproto.DeleteDraftRequest, opts ...grpc.CallOption) (*messagesproto.DeleteDraftResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.DeleteDraftResponse), args.Error(1)
}

func (m *MockMessageClient) SendDraft(ctx context.Context, in *messagesproto.SendDraftRequest, opts ...grpc.CallOption) (*messagesproto.SendDraftResponse, error) {
	args := m.Called(ctx, in)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*messagesproto.SendDraftResponse), args.Error(1)
}

type MockMinioClient struct {
	mock.Mock
}

func (m *MockMinioClient) BucketExists(ctx context.Context, bucketName string) (bool, error) {
	args := m.Called(ctx, bucketName)
	return args.Bool(0), args.Error(1)
}

func (m *MockMinioClient) MakeBucket(ctx context.Context, bucketName string, opts minio.MakeBucketOptions) error {
	args := m.Called(ctx, bucketName, opts)
	return args.Error(0)
}

func (m *MockMinioClient) GetObject(ctx context.Context, bucketName, objectName string, opts minio.GetObjectOptions) (*minio.Object, error) {
	args := m.Called(ctx, bucketName, objectName, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*minio.Object), args.Error(1)
}

func (m *MockMinioClient) PutObject(ctx context.Context, bucketName, objectName string, reader io.Reader, objectSize int64, opts minio.PutObjectOptions) (minio.UploadInfo, error) {
	args := m.Called(ctx, bucketName, objectName, reader, objectSize, opts)
	return args.Get(0).(minio.UploadInfo), args.Error(1)
}

func setupTestServer() (*Server, *MockAuthClient, *MockProfileClient, *MockMessageClient, *MockMinioClient) {
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			GatewayPort:        "8080",
			GatewayMetricsPort: "9090",
			Host:               "localhost",
			AuthPort:           "50051",
			ProfilePort:        "50052",
			MessagesPort:       "50053",
		},
	}

	mockAuth := &MockAuthClient{}
	mockProfile := &MockProfileClient{}
	mockMessage := &MockMessageClient{}
	mockMinio := &MockMinioClient{}

	server := &Server{
		cfg:               cfg,
		authClient:        mockAuth,
		profileClient:     mockProfile,
		messageClient:     mockMessage,
		minioBucket:       "test-bucket",
		attachmentsBucket: "attachments",
	}

	return server, mockAuth, mockProfile, mockMessage, mockMinio
}

func createRequestWithToken(method, url string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "test-access-token",
	})
	return req
}

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		AppConfig: &config.AppConfig{
			GatewayPort:        "8080",
			GatewayMetricsPort: "9090",
			Host:               "localhost",
			AuthPort:           "50051",
			ProfilePort:        "50052",
			MessagesPort:       "50053",
		},
	}

	server, err := NewServer(cfg)
	assert.Nil(t, err)
	assert.NotNil(t, server)
	assert.NotNil(t, server.cfg)
	assert.NotNil(t, server.authClient)
	assert.NotNil(t, server.profileClient)
	assert.NotNil(t, server.messageClient)
}

func TestServer_AddTokenToContext(t *testing.T) {
	server, _, _, _, _ := setupTestServer()
	ctx := context.Background()

	newCtx := server.addTokenToContext(ctx, "test-token")

	md, ok := metadata.FromOutgoingContext(newCtx)
	assert.True(t, ok, "metadata should be present in context")
	assert.Equal(t, []string{"Bearer test-token"}, md.Get("authorization"))
}

func TestServer_LoginHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, mockAuth, _, _, _ := setupTestServer()

		mockAuth.On("Login", mock.Anything, mock.AnythingOfType("*authproto.LoginRequest")).
			Return(&authproto.LoginResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			}, nil)

		body := map[string]interface{}{
			"login":    "test@example.com",
			"password": "password",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(body)

		req := httptest.NewRequest("POST", "/auth/login", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.loginHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var responseBody map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&responseBody)
		expectedBody := map[string]interface{}{
			"status":  float64(200),
			"message": "success",
			"body": map[string]interface{}{
				"access_token":  "access-token",
				"refresh_token": "refresh-token",
			},
		}
		assert.Equal(t, expectedBody, responseBody)
		mockAuth.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/auth/login", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.loginHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})

	t.Run("LoginFailed", func(t *testing.T) {
		server, mockAuth, _, _, _ := setupTestServer()

		mockAuth.On("Login", mock.Anything, mock.AnythingOfType("*authproto.LoginRequest")).
			Return(nil, errors.New("login failed"))

		body := map[string]interface{}{
			"login":    "test@example.com",
			"password": "wrong-password",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(body)

		req := httptest.NewRequest("POST", "/auth/login", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.loginHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockAuth.AssertExpectations(t)
	})
}

func TestServer_SignupHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, mockAuth, _, _, _ := setupTestServer()

		mockAuth.On("Signup", mock.Anything, mock.AnythingOfType("*authproto.SignupRequest")).
			Return(&authproto.SignupResponse{
				AccessToken:  "access-token",
				RefreshToken: "refresh-token",
			}, nil)

		body := map[string]interface{}{
			"name":     "New User",
			"username": "newuser",
			"birthday": "1990-01-01",
			"gender":   "male",
			"password": "password",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(body)

		req := httptest.NewRequest("POST", "/auth/signup", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.signupHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockAuth.AssertExpectations(t)
	})

	t.Run("AlreadyExists", func(t *testing.T) {
		server, mockAuth, _, _, _ := setupTestServer()

		mockAuth.On("Signup", mock.Anything, mock.AnythingOfType("*authproto.SignupRequest")).
			Return(nil, status.Error(codes.AlreadyExists, "user already exists"))

		body := map[string]interface{}{
			"name":     "Existing User",
			"username": "existinguser",
			"birthday": "1990-01-01",
			"gender":   "male",
			"password": "password",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(body)

		req := httptest.NewRequest("POST", "/auth/signup", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.signupHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		mockAuth.AssertExpectations(t)
	})
}

func TestServer_GetProfileHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("GetProfile", mock.Anything, mock.AnythingOfType("*profileproto.GetProfileRequest")).
			Return(&profileproto.GetProfileResponse{
				Profile: &profileproto.Profile{
					Username:   "testuser",
					Name:       "Test",
					Surname:    "User",
					AvatarPath: "/avatars/test.jpg",
				},
			}, nil)

		req := httptest.NewRequest("GET", "/user/profile", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getProfileHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})

	t.Run("NoAccessToken", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("GET", "/user/profile", nil)
		w := httptest.NewRecorder()

		server.getProfileHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("GRPCError", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("GetProfile", mock.Anything, mock.AnythingOfType("*profileproto.GetProfileRequest")).
			Return(nil, status.Error(codes.NotFound, "profile not found"))

		req := httptest.NewRequest("GET", "/user/profile", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getProfileHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})
}

func TestServer_UpdateProfileHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("UpdateProfile", mock.Anything, mock.AnythingOfType("*profileproto.UpdateProfileRequest")).
			Return(&profileproto.UpdateProfileResponse{
				Profile: &profileproto.Profile{
					Username: "testuser",
					Name:     "Updated",
					Surname:  "User",
				},
			}, nil)

		updateRequest := map[string]interface{}{
			"name":       "Updated",
			"surname":    "User",
			"patronymic": "Middle",
			"gender":     "male",
			"birthday":   "1990-01-01",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(updateRequest)

		req := httptest.NewRequest("PUT", "/user/profile", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.updateProfileHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("PUT", "/user/profile", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.updateProfileHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_MessagePageHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("MessagePage", mock.Anything, mock.AnythingOfType("*messagesproto.MessagePageRequest")).
			Return(&messagesproto.MessagePageResponse{
				Message: &messagesproto.FullMessage{
					Topic: "Test Message",
					Text:  "Test content",
				},
			}, nil)

		req := httptest.NewRequest("GET", "/messages/123", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.messagePageHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("MessageNotFound", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("MessagePage", mock.Anything, mock.AnythingOfType("*messagesproto.MessagePageRequest")).
			Return(nil, status.Error(codes.NotFound, "message not found"))

		req := httptest.NewRequest("GET", "/messages/999", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.messagePageHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InternalError", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("MessagePage", mock.Anything, mock.AnythingOfType("*messagesproto.MessagePageRequest")).
			Return(nil, errors.New("internal error"))

		req := httptest.NewRequest("GET", "/messages/123", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.messagePageHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_SendHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("Send", mock.Anything, mock.AnythingOfType("*messagesproto.SendRequest")).
			Return(&messagesproto.SendResponse{
				MessageId: "123",
			}, nil)

		sendRequest := map[string]interface{}{
			"topic": "Test Topic",
			"text":  "Test Message",
			"receivers": []map[string]interface{}{
				{"email": "receiver@example.com"},
			},
			"files": []map[string]interface{}{},
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(sendRequest)

		req := httptest.NewRequest("POST", "/messages/send", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.sendHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.Equal(t, "success", response["message"])
		mockMessage.AssertExpectations(t)
	})

	t.Run("SendFailed", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("Send", mock.Anything, mock.AnythingOfType("*messagesproto.SendRequest")).
			Return(nil, errors.New("send failed"))

		sendRequest := map[string]interface{}{
			"topic": "Test Topic",
			"text":  "Test Message",
			"receivers": []map[string]interface{}{
				{"email": "receiver@example.com"},
			},
			"files": []map[string]interface{}{},
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(sendRequest)

		req := httptest.NewRequest("POST", "/messages/send", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.sendHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/messages/send", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.sendHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_GetFoldersHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("GetFolders", mock.Anything, mock.AnythingOfType("*messagesproto.GetFoldersRequest")).
			Return(&messagesproto.GetFoldersResponse{
				Folders: []*messagesproto.Folder{
					{
						FolderId:   "1",
						FolderName: "Inbox",
						FolderType: "inbox",
					},
					{
						FolderId:   "2",
						FolderName: "Sent",
						FolderType: "sent",
					},
				},
			}, nil)

		req := httptest.NewRequest("GET", "/messages/get-folders", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getFoldersHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("GetFoldersFailed", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("GetFolders", mock.Anything, mock.AnythingOfType("*messagesproto.GetFoldersRequest")).
			Return(nil, errors.New("get folders failed"))

		req := httptest.NewRequest("GET", "/messages/get-folders", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getFoldersHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_CreateFolderHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("CreateFolder", mock.Anything, mock.AnythingOfType("*messagesproto.CreateFolderRequest")).
			Return(&messagesproto.CreateFolderResponse{
				FolderId:   "3",
				FolderName: "New Folder",
				FolderType: "custom",
			}, nil)

		createRequest := map[string]interface{}{
			"folder_name": "New Folder",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(createRequest)

		req := httptest.NewRequest("POST", "/messages/create-folder", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.createFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("FolderAlreadyExists", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("CreateFolder", mock.Anything, mock.AnythingOfType("*messagesproto.CreateFolderRequest")).
			Return(nil, status.Error(codes.AlreadyExists, "folder already exists"))

		createRequest := map[string]interface{}{
			"folder_name": "New Folder",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(createRequest)

		req := httptest.NewRequest("POST", "/messages/create-folder", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.createFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/messages/create-folder", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.createFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_RenameFolderHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("RenameFolder", mock.Anything, mock.AnythingOfType("*messagesproto.RenameFolderRequest")).
			Return(&messagesproto.RenameFolderResponse{
				FolderId:   "1",
				FolderName: "Renamed Folder",
				FolderType: "custom",
			}, nil)

		renameRequest := map[string]interface{}{
			"folder_id":       "1",
			"new_folder_name": "Renamed Folder",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(renameRequest)

		req := httptest.NewRequest("PUT", "/messages/rename-folder", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.renameFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("PUT", "/messages/rename-folder", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.renameFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_SaveDraftHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("SaveDraft", mock.Anything, mock.AnythingOfType("*messagesproto.SaveDraftRequest")).
			Return(&messagesproto.SaveDraftResponse{
				DraftId: "123",
			}, nil)

		saveDraftRequest := map[string]interface{}{
			"draft_id":  "",
			"thread_id": "thread123",
			"topic":     "Draft Topic",
			"text":      "Draft Text",
			"receivers": []map[string]interface{}{
				{"email": "test@example.com"},
			},
			"files": []map[string]interface{}{},
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(saveDraftRequest)

		req := httptest.NewRequest("POST", "/messages/save-draft", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.saveDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/messages/save-draft", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.saveDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_DeleteDraftHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("DeleteDraft", mock.Anything, mock.AnythingOfType("*messagesproto.DeleteDraftRequest")).
			Return(&messagesproto.DeleteDraftResponse{
				Success: true,
			}, nil)

		deleteDraftRequest := map[string]interface{}{
			"draft_id": "123",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(deleteDraftRequest)

		req := httptest.NewRequest("DELETE", "/messages/delete-draft", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.deleteDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("DELETE", "/messages/delete-draft", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.deleteDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_MarkAsSpamHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("MarkAsSpam", mock.Anything, mock.AnythingOfType("*messagesproto.MarkAsSpamRequest")).
			Return(&messagesproto.MarkAsSpamResponse{}, nil)

		spamRequest := map[string]interface{}{
			"message_id": "123",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(spamRequest)

		req := httptest.NewRequest("POST", "/messages/mark-as-spam", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.markAsSpamHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/messages/mark-as-spam", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.markAsSpamHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_MoveToFolderHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("MoveToFolder", mock.Anything, mock.AnythingOfType("*messagesproto.MoveToFolderRequest")).
			Return(&messagesproto.MoveToFolderResponse{}, nil)

		moveRequest := map[string]interface{}{
			"message_id": "123",
			"folder_id":  "2",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(moveRequest)

		req := httptest.NewRequest("POST", "/messages/move-to-folder", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.moveToFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/messages/move-to-folder", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.moveToFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_UploadAvatarHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("UploadAvatar", mock.Anything, mock.AnythingOfType("*profileproto.UploadAvatarRequest")).
			Return(&profileproto.UploadAvatarResponse{
				AvatarPath: "/avatars/new-avatar.jpg",
			}, nil)

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("avatar", "test.jpg")
		part.Write([]byte("fake image data"))
		writer.Close()

		req := httptest.NewRequest("POST", "/user/upload/avatar", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.uploadAvatarHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})

	t.Run("NoFileProvided", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/user/upload/avatar", strings.NewReader(""))
		req.Header.Set("Content-Type", "multipart/form-data")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.uploadAvatarHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	// Тест FileTooLarge удален
}

func TestNormalizeAvatarURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{
			name:     "ValidURL",
			input:    "http://example.com/avatar.jpg",
			expected: "http://example.com/avatar.jpg",
			hasError: false,
		},
		{
			name:     "LocalhostToMinio",
			input:    "http://localhost:9000/avatar.jpg",
			expected: "http://minio:9000/avatar.jpg",
			hasError: false,
		},
		{
			name:     "127.0.0.1ToMinio",
			input:    "http://127.0.0.1:9000/avatar.jpg",
			expected: "http://minio:9000/avatar.jpg",
			hasError: false,
		},
		{
			name:     "InvalidURL",
			input:    "://invalid",
			expected: "",
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizeAvatarURL(tt.input)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestResolveFolderID(t *testing.T) {
	folders := []*messagesproto.Folder{
		{FolderId: "1", FolderName: "Inbox", FolderType: "inbox"},
		{FolderId: "2", FolderName: "Sent", FolderType: "sent"},
		{FolderId: "3", FolderName: "Custom", FolderType: "custom"},
	}

	tests := []struct {
		name     string
		target   string
		expected string
	}{
		{
			name:     "ByFolderID",
			target:   "1",
			expected: "1",
		},
		{
			name:     "ByFolderName",
			target:   "Inbox",
			expected: "1",
		},
		{
			name:     "ByFolderType",
			target:   "inbox",
			expected: "1",
		},
		{
			name:     "NotFound",
			target:   "nonexistent",
			expected: "",
		},
		{
			name:     "EmptyTarget",
			target:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveFolderID(folders, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapProfile(t *testing.T) {
	t.Run("WithProfile", func(t *testing.T) {
		profile := &profileproto.Profile{
			Username:   "testuser",
			CreatedAt:  "2023-01-01",
			Name:       "Test",
			Surname:    "User",
			Patronymic: "Middle",
			Gender:     "male",
			Birthday:   "1990-01-01",
			AvatarPath: "/avatars/test.jpg",
		}

		result := mapProfile(profile)

		assert.Equal(t, "testuser", result.Username)
		assert.Equal(t, "Test", result.Name)
		assert.Equal(t, "User", result.Surname)
		assert.Equal(t, "Middle", result.Patronymic)
		assert.Equal(t, "male", result.Gender)
		assert.Equal(t, "1990-01-01", result.DateOfBirth)
		assert.Equal(t, "/avatars/test.jpg", result.AvatarPath)
		assert.Equal(t, "user", result.Role)
	})

	t.Run("NilProfile", func(t *testing.T) {
		result := mapProfile(nil)

		assert.Equal(t, "", result.Username)
		assert.Equal(t, "", result.Name)
		assert.Equal(t, "", result.Surname)
	})
}

func TestServer_RefreshHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, mockAuth, _, _, _ := setupTestServer()

		mockAuth.On("Refresh", mock.Anything, mock.AnythingOfType("*authproto.RefreshRequest")).
			Return(&authproto.RefreshResponse{
				AccessToken: "new-access-token",
			}, nil)

		req := httptest.NewRequest("POST", "/auth/refresh", nil)
		req.AddCookie(&http.Cookie{
			Name:  "refresh_token",
			Value: "old-refresh-token",
		})
		w := httptest.NewRecorder()

		server.refreshHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockAuth.AssertExpectations(t)
	})

	t.Run("NoRefreshToken", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/auth/refresh", nil)
		w := httptest.NewRecorder()

		server.refreshHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("RefreshInBody", func(t *testing.T) {
		server, mockAuth, _, _, _ := setupTestServer()

		mockAuth.On("Refresh", mock.Anything, mock.AnythingOfType("*authproto.RefreshRequest")).
			Return(&authproto.RefreshResponse{
				AccessToken: "new-access-token",
			}, nil)

		body := map[string]string{"refresh_token": "refresh-from-body"}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(body)

		req := httptest.NewRequest("POST", "/auth/refresh", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		server.refreshHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockAuth.AssertExpectations(t)
	})
}

func TestServer_LogoutHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, mockAuth, _, _, _ := setupTestServer()

		mockAuth.On("Logout", mock.Anything, mock.AnythingOfType("*authproto.LogoutRequest")).
			Return(&authproto.LogoutResponse{}, nil)

		req := httptest.NewRequest("POST", "/auth/logout", nil)
		w := httptest.NewRecorder()

		server.logoutHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		cookies := resp.Cookies()
		var accessCookie, refreshCookie *http.Cookie
		for _, cookie := range cookies {
			if cookie.Name == "access_token" {
				accessCookie = cookie
			} else if cookie.Name == "refresh_token" {
				refreshCookie = cookie
			}
		}
		assert.NotNil(t, accessCookie)
		assert.NotNil(t, refreshCookie)
		assert.Equal(t, -1, accessCookie.MaxAge)
		assert.Equal(t, -1, refreshCookie.MaxAge)

		mockAuth.AssertExpectations(t)
	})

	t.Run("LogoutFailed", func(t *testing.T) {
		server, mockAuth, _, _, _ := setupTestServer()

		mockAuth.On("Logout", mock.Anything, mock.AnythingOfType("*authproto.LogoutRequest")).
			Return(nil, errors.New("logout failed"))

		req := httptest.NewRequest("POST", "/auth/logout", nil)
		w := httptest.NewRecorder()

		server.logoutHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockAuth.AssertExpectations(t)
	})
}

func TestWriteGrpcAwareError(t *testing.T) {
	t.Run("Unauthenticated", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeGrpcAwareError(w, status.Error(codes.Unauthenticated, "invalid token"), "Authentication failed")

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("InvalidArgument", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeGrpcAwareError(w, status.Error(codes.InvalidArgument, "invalid input"), "Invalid request")

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("NotFound", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeGrpcAwareError(w, status.Error(codes.NotFound, "not found"), "Resource not found")

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeGrpcAwareError(w, status.Error(codes.PermissionDenied, "access denied"), "Access denied")

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("AlreadyExists", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeGrpcAwareError(w, status.Error(codes.AlreadyExists, "already exists"), "Conflict")

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
	})

	t.Run("GenericError", func(t *testing.T) {
		w := httptest.NewRecorder()
		writeGrpcAwareError(w, errors.New("generic error"), "Internal error")

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	})
}

func TestServer_ReplyHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("Reply", mock.Anything, mock.AnythingOfType("*messagesproto.ReplyRequest")).
			Return(&messagesproto.ReplyResponse{
				MessageId: "456",
			}, nil)

		replyRequest := map[string]interface{}{
			"root_message_id": "123",
			"topic":           "Re: Test",
			"text":            "Reply text",
			"thread_root":     "thread123",
			"receivers": []map[string]interface{}{
				{"email": "reply@example.com"},
			},
			"files": []map[string]interface{}{},
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(replyRequest)

		req := httptest.NewRequest("POST", "/messages/reply", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.replyHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/messages/reply", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.replyHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_SettingsHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("Settings", mock.Anything, mock.AnythingOfType("*profileproto.SettingsRequest")).
			Return(&profileproto.SettingsResponse{
				Settings: &profileproto.Settings{
					Theme: "dark",
				},
			}, nil)

		req := httptest.NewRequest("GET", "/user/settings", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.settingsHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})

	t.Run("GRPCError", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("Settings", mock.Anything, mock.AnythingOfType("*profileproto.SettingsRequest")).
			Return(nil, status.Error(codes.Internal, "internal error"))

		req := httptest.NewRequest("GET", "/user/settings", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.settingsHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})
}

func TestServer_SendDraftHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("SendDraft", mock.Anything, mock.AnythingOfType("*messagesproto.SendDraftRequest")).
			Return(&messagesproto.SendDraftResponse{
				Success:   true,
				MessageId: "456",
			}, nil)

		sendDraftRequest := map[string]interface{}{
			"draft_id": "123",
		}
		var bodyBytes bytes.Buffer
		json.NewEncoder(&bodyBytes).Encode(sendDraftRequest)

		req := httptest.NewRequest("POST", "/messages/send-draft", &bodyBytes)
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.sendDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("POST", "/messages/send-draft", strings.NewReader("invalid json"))
		req.Header.Set("Content-Type", "application/json")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.sendDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_InboxHandler(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("GetFolders", mock.Anything, mock.AnythingOfType("*messagesproto.GetFoldersRequest")).
			Return(&messagesproto.GetFoldersResponse{
				Folders: []*messagesproto.Folder{
					{FolderId: "1", FolderName: "Inbox", FolderType: "inbox"},
				},
			}, nil)

		mockMessage.On("GetFolder", mock.Anything, mock.AnythingOfType("*messagesproto.GetFolderRequest")).
			Return(&messagesproto.GetFolderResponse{
				MessageTotal:  "10",
				MessageUnread: "3",
				Messages:      []*messagesproto.Message{},
				Pagination:    &messagesproto.PaginationInfo{HasNext: "false"},
			}, nil)

		req := httptest.NewRequest("GET", "/messages/inbox", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.inboxHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("GetFoldersError", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("GetFolders", mock.Anything, mock.AnythingOfType("*messagesproto.GetFoldersRequest")).
			Return(nil, errors.New("get folders failed"))

		req := httptest.NewRequest("GET", "/messages/inbox", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.inboxHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestSanitizeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "ValidFileName",
			input:    "test.jpg",
			expected: "test.jpg",
		},
		{
			name:     "WithPath",
			input:    "path/to/file.jpg",
			expected: "file.jpg",
		},
		{
			name:     "Empty",
			input:    "",
			expected: "file.bin",
		},
		{
			name:     "Dot",
			input:    ".",
			expected: "file.bin",
		},
		{
			name:     "Slash",
			input:    "/",
			expected: "file.bin",
		},
		{
			name:     "WithSpaces",
			input:    "  test file.jpg  ",
			expected: "test file.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFileName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSetAuthCookies(t *testing.T) {
	w := httptest.NewRecorder()
	setAuthCookies(w, "access-token", "refresh-token")

	resp := w.Result()
	cookies := resp.Cookies()

	var accessCookie, refreshCookie, wsCookie *http.Cookie
	for _, cookie := range cookies {
		switch cookie.Name {
		case "access_token":
			accessCookie = cookie
		case "refresh_token":
			refreshCookie = cookie
		case "ws_token":
			wsCookie = cookie
		}
	}

	assert.NotNil(t, accessCookie)
	assert.Equal(t, "access-token", accessCookie.Value)
	assert.Equal(t, 15*60, accessCookie.MaxAge)

	assert.NotNil(t, refreshCookie)
	assert.Equal(t, "refresh-token", refreshCookie.Value)
	assert.Equal(t, 30*24*60*60, refreshCookie.MaxAge)

	assert.NotNil(t, wsCookie)
	assert.Equal(t, "access-token", wsCookie.Value)
	assert.Equal(t, 15*60, wsCookie.MaxAge)
}

func TestServer_Stop(t *testing.T) {
	server, _, _, _, _ := setupTestServer()

	server.httpServer = &http.Server{}
	ctx := context.Background()

	err := server.Stop(ctx)
	assert.NoError(t, err)
}

func TestServer_GetAvatarHandler(t *testing.T) {
	t.Run("SuccessWithURL", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("GET", "/user/avatar?url=http://example.com/avatar.jpg", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getAvatarHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
	})

	t.Run("SuccessWithProfileAvatar", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("GetProfile", mock.Anything, mock.AnythingOfType("*profileproto.GetProfileRequest")).
			Return(&profileproto.GetProfileResponse{
				Profile: &profileproto.Profile{
					AvatarPath: "http://example.com/avatar.jpg",
				},
			}, nil)

		req := httptest.NewRequest("GET", "/user/avatar", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getAvatarHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})

	t.Run("NoAvatarPath", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("GetProfile", mock.Anything, mock.AnythingOfType("*profileproto.GetProfileRequest")).
			Return(&profileproto.GetProfileResponse{
				Profile: &profileproto.Profile{
					AvatarPath: "",
				},
			}, nil)

		req := httptest.NewRequest("GET", "/user/avatar", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getAvatarHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})

	t.Run("GetProfileError", func(t *testing.T) {
		server, _, mockProfile, _, _ := setupTestServer()

		mockProfile.On("GetProfile", mock.Anything, mock.AnythingOfType("*profileproto.GetProfileRequest")).
			Return(nil, errors.New("profile error"))

		req := httptest.NewRequest("GET", "/user/avatar", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getAvatarHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})
}

func TestWriteResponse(t *testing.T) {
	t.Run("SuccessResponse", func(t *testing.T) {
		w := httptest.NewRecorder()
		body := map[string]string{"key": "value"}

		writeResponse(w, http.StatusOK, "success", body)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.Equal(t, float64(200), response["status"])
		assert.Equal(t, "success", response["message"])
		assert.NotNil(t, response["body"])
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		w := httptest.NewRecorder()

		writeResponse(w, http.StatusBadRequest, "error", nil)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)
		assert.Equal(t, float64(400), response["status"])
		assert.Equal(t, "error", response["message"])
	})
}

func TestRespondSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	body := map[string]string{"test": "data"}

	respondSuccess(w, body)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, "success", response["message"])
}

func TestRespondError(t *testing.T) {
	w := httptest.NewRecorder()

	respondError(w, "error message")

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	var response map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&response)
	assert.Equal(t, "error message", response["message"])
}

func TestServer_GetFolderHandler(t *testing.T) {
	t.Run("FolderNotFound", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("GetFolders", mock.Anything, mock.AnythingOfType("*messagesproto.GetFoldersRequest")).
			Return(&messagesproto.GetFoldersResponse{
				Folders: []*messagesproto.Folder{},
			}, nil)

		req := httptest.NewRequest("GET", "/folder/NonExistent", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("GetFoldersError", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("GetFolders", mock.Anything, mock.AnythingOfType("*messagesproto.GetFoldersRequest")).
			Return(nil, status.Error(codes.Internal, "get folders error"))

		req := httptest.NewRequest("GET", "/folder/Test", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.getFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("NoAccessToken", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("GET", "/folder/Test", nil)
		w := httptest.NewRecorder()

		server.getFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestServer_DeleteFolderHandler(t *testing.T) {
	t.Run("SuccessfulDeleteFolder", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("DeleteFolder", mock.Anything, mock.AnythingOfType("*messagesproto.DeleteFolderRequest")).
			Return(&messagesproto.DeleteFolderResponse{}, nil)

		req := httptest.NewRequest("DELETE", "/folder?folder_id=folder-1", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.deleteFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("DeleteFolderError", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("DeleteFolder", mock.Anything, mock.AnythingOfType("*messagesproto.DeleteFolderRequest")).
			Return(nil, errors.New("delete folder error"))

		req := httptest.NewRequest("DELETE", "/folder?folder_id=folder-1", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.deleteFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("NoAccessToken", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		req := httptest.NewRequest("DELETE", "/folder?folder_id=folder-1", nil)
		w := httptest.NewRecorder()

		server.deleteFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestServer_DeleteMessageFromFolderHandler(t *testing.T) {
	t.Run("SuccessfulDeleteMessage", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("DeleteMessageFromFolder", mock.Anything, mock.AnythingOfType("*messagesproto.DeleteMessageFromFolderRequest")).
			Return(&messagesproto.DeleteMessageFromFolderResponse{}, nil)

		body := bytes.NewBufferString(`{"message_id": "msg-1", "folder_id": "folder-1"}`)
		req := httptest.NewRequest("DELETE", "/folder/message", body)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.deleteMessageFromFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidBodyFormat", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		body := bytes.NewBufferString(`invalid json`)
		req := httptest.NewRequest("DELETE", "/folder/message", body)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.deleteMessageFromFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("DeleteMessageError", func(t *testing.T) {
		server, _, _, mockMessage, _ := setupTestServer()

		mockMessage.On("DeleteMessageFromFolder", mock.Anything, mock.AnythingOfType("*messagesproto.DeleteMessageFromFolderRequest")).
			Return(nil, errors.New("delete message error"))

		body := bytes.NewBufferString(`{"message_id": "msg-1", "folder_id": "folder-1"}`)
		req := httptest.NewRequest("DELETE", "/folder/message", body)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: "test-token",
		})
		w := httptest.NewRecorder()

		server.deleteMessageFromFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("NoAccessToken", func(t *testing.T) {
		server, _, _, _, _ := setupTestServer()

		body := bytes.NewBufferString(`{"message_id": "msg-1", "folder_id": "folder-1"}`)
		req := httptest.NewRequest("DELETE", "/folder/message", body)
		w := httptest.NewRecorder()

		server.deleteMessageFromFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestGetAccessToken(t *testing.T) {
	t.Run("FromAuthorizationHeader", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		token, err := getAccessToken(req)
		assert.NoError(t, err)
		assert.Equal(t, "test-token", token)
	})

	t.Run("FromQueryToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?token=query-token", nil)

		token, err := getAccessToken(req)
		assert.NoError(t, err)
		assert.Equal(t, "query-token", token)
	})

	t.Run("FromQueryAccessToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test?access_token=query-access-token", nil)

		token, err := getAccessToken(req)
		assert.NoError(t, err)
		assert.Equal(t, "query-access-token", token)
	})

	t.Run("FromAccessTokenCookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "access_token", Value: "cookie-token"})

		token, err := getAccessToken(req)
		assert.NoError(t, err)
		assert.Equal(t, "cookie-token", token)
	})

	t.Run("FromWsTokenCookie", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.AddCookie(&http.Cookie{Name: "ws_token", Value: "ws-token"})

		token, err := getAccessToken(req)
		assert.NoError(t, err)
		assert.Equal(t, "ws-token", token)
	})

	t.Run("NoToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)

		token, err := getAccessToken(req)
		assert.Error(t, err)
		assert.Equal(t, "", token)
	})
}

func TestEmptyFolderResponse(t *testing.T) {
	response := emptyFolderResponse()

	assert.Equal(t, "0", response.MessageTotal)
	assert.Equal(t, "0", response.MessageUnread)
	assert.Empty(t, response.Messages)
	assert.Equal(t, "false", response.Pagination.HasNext)
}
