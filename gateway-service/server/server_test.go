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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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

func setupTestServer() (*Server, *MockAuthClient, *MockProfileClient, *MockMessageClient) {
	cfg := &config.AppConfig{
		GatewayPort:        "8080",
		GatewayMetricsPort: "9090",
	}

	mockAuth := &MockAuthClient{}
	mockProfile := &MockProfileClient{}
	mockMessage := &MockMessageClient{}

	server := &Server{
		cfg:           cfg,
		authClient:    mockAuth,
		profileClient: mockProfile,
		messageClient: mockMessage,
	}

	return server, mockAuth, mockProfile, mockMessage
}

func createRequestWithToken(method, url string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, url, body)
	req.AddCookie(&http.Cookie{
		Name:  "access_token",
		Value: "test-access-token",
	})
	return req
}

func TestServer_LoginHandler(t *testing.T) {
	server, mockAuth, _, _ := setupTestServer()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func()
		expectedStatus int
		expectedBody   interface{}
	}{
		{
			name: "Success",
			requestBody: map[string]interface{}{
				"login":    "test@example.com",
				"password": "password",
			},
			mockSetup: func() {
				mockAuth.On("Login", mock.Anything, mock.AnythingOfType("*authproto.LoginRequest")).
					Return(&authproto.LoginResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					}, nil)
			},
			expectedStatus: http.StatusOK,
			expectedBody: map[string]interface{}{
				"status":  float64(200),
				"message": "success",
				"body": map[string]interface{}{
					"accessToken":  "access-token",
					"refreshToken": "refresh-token",
				},
			},
		},
		{
			name: "InvalidRequestBody",
			requestBody: map[string]interface{}{
				"invalid": "data",
			},
			mockSetup:      func() {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: map[string]interface{}{
				"status":  float64(400),
				"message": "Invalid request body",
			},
		},
		{
			name: "LoginFailed",
			requestBody: map[string]interface{}{
				"login":    "test@example.com",
				"password": "wrong-password",
			},
			mockSetup: func() {
				mockAuth.On("Login", mock.Anything, mock.AnythingOfType("*authproto.LoginRequest")).
					Return(nil, errors.New("login failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			expectedBody: map[string]interface{}{
				"status":  float64(500),
				"message": "Login failed",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var body bytes.Buffer
			json.NewEncoder(&body).Encode(tt.requestBody)

			req := httptest.NewRequest("POST", "/auth/login", &body)
			w := httptest.NewRecorder()

			server.loginHandler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			var responseBody map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&responseBody)
			assert.Equal(t, tt.expectedBody, responseBody)

			mockAuth.AssertExpectations(t)
		})
	}
}

func TestServer_SignupHandler(t *testing.T) {
	server, mockAuth, _, _ := setupTestServer()

	tests := []struct {
		name           string
		requestBody    interface{}
		mockSetup      func()
		expectedStatus int
	}{
		{
			name: "Success",
			requestBody: map[string]interface{}{
				"name":     "New User",
				"username": "newuser",
				"birthday": "1990-01-01",
				"gender":   "male",
				"password": "password",
			},
			mockSetup: func() {
				mockAuth.On("Signup", mock.Anything, mock.AnythingOfType("*authproto.SignupRequest")).
					Return(&authproto.SignupResponse{
						AccessToken:  "access-token",
						RefreshToken: "refresh-token",
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "AlreadyExists",
			requestBody: map[string]interface{}{
				"name":     "Existing User",
				"username": "existinguser",
				"birthday": "1990-01-01",
				"gender":   "male",
				"password": "password",
			},
			mockSetup: func() {
				mockAuth.On("Signup", mock.Anything, mock.AnythingOfType("*authproto.SignupRequest")).
					Return(nil, status.Error(codes.AlreadyExists, "user already exists"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			var body bytes.Buffer
			json.NewEncoder(&body).Encode(tt.requestBody)

			req := httptest.NewRequest("POST", "/auth/signup", &body)
			w := httptest.NewRecorder()

			server.signupHandler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			mockAuth.AssertExpectations(t)
		})
	}
}

func TestServer_GetProfileHandler(t *testing.T) {
	server, _, mockProfile, _ := setupTestServer()

	tests := []struct {
		name           string
		setupCookies   bool
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:         "Success",
			setupCookies: true,
			mockSetup: func() {
				mockProfile.On("GetProfile", mock.Anything, mock.AnythingOfType("*profileproto.GetProfileRequest")).
					Return(&profileproto.GetProfileResponse{
						Profile: &profileproto.Profile{
							Username:   "testuser",
							Name:       "Test",
							Surname:    "User",
							AvatarPath: "/avatars/test.jpg",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "NoAccessToken",
			setupCookies:   false,
			mockSetup:      func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:         "GRPCError",
			setupCookies: true,
			mockSetup: func() {
				mockProfile.On("GetProfile", mock.Anything, mock.AnythingOfType("*profileproto.GetProfileRequest")).
					Return(nil, status.Error(codes.NotFound, "profile not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := httptest.NewRequest("GET", "/user/profile", nil)
			if tt.setupCookies {
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: "test-token",
				})
			}

			w := httptest.NewRecorder()

			server.getProfileHandler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			mockProfile.AssertExpectations(t)
		})
	}
}

func TestServer_UpdateProfileHandler(t *testing.T) {
	server, _, mockProfile, _ := setupTestServer()

	updateRequest := map[string]interface{}{
		"name":       "Updated",
		"surname":    "User",
		"patronymic": "Middle",
		"gender":     "male",
		"birthday":   "1990-01-01",
	}

	t.Run("Success", func(t *testing.T) {
		mockProfile.On("UpdateProfile", mock.Anything, mock.AnythingOfType("*profileproto.UpdateProfileRequest")).
			Return(&profileproto.UpdateProfileResponse{
				Profile: &profileproto.Profile{
					Username: "testuser",
					Name:     "Updated",
					Surname:  "User",
				},
			}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(updateRequest)

		req := createRequestWithToken("PUT", "/user/profile", &body)
		w := httptest.NewRecorder()

		server.updateProfileHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		req := createRequestWithToken("PUT", "/user/profile", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()

		server.updateProfileHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_MessagePageHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	tests := []struct {
		name           string
		messageID      string
		mockSetup      func()
		expectedStatus int
	}{
		{
			name:      "Success",
			messageID: "123",
			mockSetup: func() {
				mockMessage.On("MessagePage", mock.Anything, mock.AnythingOfType("*messagesproto.MessagePageRequest")).
					Return(&messagesproto.MessagePageResponse{
						Message: &messagesproto.FullMessage{
							Topic: "Test Message",
							Text:  "Test content",
						},
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:      "MessageNotFound",
			messageID: "999",
			mockSetup: func() {
				mockMessage.On("MessagePage", mock.Anything, mock.AnythingOfType("*messagesproto.MessagePageRequest")).
					Return(nil, status.Error(codes.NotFound, "message not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			req := createRequestWithToken("GET", "/messages/"+tt.messageID, nil)
			w := httptest.NewRecorder()

			server.messagePageHandler(w, req)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			mockMessage.AssertExpectations(t)
		})
	}
}

func TestServer_SendHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	sendRequest := map[string]interface{}{
		"topic": "Test Topic",
		"text":  "Test Message",
		"receivers": []map[string]interface{}{
			{"email": "receiver@example.com"},
		},
		"files": []map[string]interface{}{},
	}

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("Send", mock.Anything, mock.AnythingOfType("*messagesproto.SendRequest")).
			Return(&messagesproto.SendResponse{
				MessageId: "123",
			}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(sendRequest)

		req := createRequestWithToken("POST", "/messages/send", &body)
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
		mockMessage.On("Send", mock.Anything, mock.AnythingOfType("*messagesproto.SendRequest")).
			Return(nil, errors.New("send failed"))

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(sendRequest)

		req := createRequestWithToken("POST", "/messages/send", &body)
		w := httptest.NewRecorder()

		server.sendHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_GetFoldersHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	t.Run("Success", func(t *testing.T) {
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

		req := createRequestWithToken("GET", "/messages/get-folders", nil)
		w := httptest.NewRecorder()

		server.getFoldersHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("GetFoldersFailed", func(t *testing.T) {
		mockMessage.On("GetFolders", mock.Anything, mock.AnythingOfType("*messagesproto.GetFoldersRequest")).
			Return(nil, errors.New("get folders failed"))

		req := createRequestWithToken("GET", "/messages/get-folders", nil)
		w := httptest.NewRecorder()

		server.getFoldersHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_CreateFolderHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	createRequest := map[string]interface{}{
		"folder_name": "New Folder",
	}

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("CreateFolder", mock.Anything, mock.AnythingOfType("*messagesproto.CreateFolderRequest")).
			Return(&messagesproto.CreateFolderResponse{
				FolderId:   "3",
				FolderName: "New Folder",
				FolderType: "custom",
			}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(createRequest)

		req := createRequestWithToken("POST", "/messages/create-folder", &body)
		w := httptest.NewRecorder()

		server.createFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("FolderAlreadyExists", func(t *testing.T) {
		mockMessage.On("CreateFolder", mock.Anything, mock.AnythingOfType("*messagesproto.CreateFolderRequest")).
			Return(nil, status.Error(codes.AlreadyExists, "folder already exists"))

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(createRequest)

		req := createRequestWithToken("POST", "/messages/create-folder", &body)
		w := httptest.NewRecorder()

		server.createFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_RenameFolderHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	renameRequest := map[string]interface{}{
		"folder_id":       "1",
		"new_folder_name": "Renamed Folder",
	}

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("RenameFolder", mock.Anything, mock.AnythingOfType("*messagesproto.RenameFolderRequest")).
			Return(&messagesproto.RenameFolderResponse{
				FolderId:   "1",
				FolderName: "Renamed Folder",
				FolderType: "custom",
			}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(renameRequest)

		req := createRequestWithToken("PUT", "/messages/rename-folder", &body)
		w := httptest.NewRecorder()

		server.renameFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_SaveDraftHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

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

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("SaveDraft", mock.Anything, mock.AnythingOfType("*messagesproto.SaveDraftRequest")).
			Return(&messagesproto.SaveDraftResponse{
				DraftId: "123",
			}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(saveDraftRequest)

		req := createRequestWithToken("POST", "/messages/save-draft", &body)
		w := httptest.NewRecorder()

		server.saveDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_DeleteDraftHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	deleteDraftRequest := map[string]interface{}{
		"draft_id": "123",
	}

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("DeleteDraft", mock.Anything, mock.AnythingOfType("*messagesproto.DeleteDraftRequest")).
			Return(&messagesproto.DeleteDraftResponse{
				Success: true,
			}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(deleteDraftRequest)

		req := createRequestWithToken("DELETE", "/messages/delete-draft", &body)
		w := httptest.NewRecorder()

		server.deleteDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_MarkAsSpamHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	spamRequest := map[string]interface{}{
		"message_id": "123",
	}

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("MarkAsSpam", mock.Anything, mock.AnythingOfType("*messagesproto.MarkAsSpamRequest")).
			Return(&messagesproto.MarkAsSpamResponse{}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(spamRequest)

		req := createRequestWithToken("POST", "/messages/mark-as-spam", &body)
		w := httptest.NewRecorder()

		server.markAsSpamHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_MoveToFolderHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	moveRequest := map[string]interface{}{
		"message_id": "123",
		"folder_id":  "2",
	}

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("MoveToFolder", mock.Anything, mock.AnythingOfType("*messagesproto.MoveToFolderRequest")).
			Return(&messagesproto.MoveToFolderResponse{}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(moveRequest)

		req := createRequestWithToken("POST", "/messages/move-to-folder", &body)
		w := httptest.NewRecorder()

		server.moveToFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_UploadAvatarHandler(t *testing.T) {
	server, _, mockProfile, _ := setupTestServer()

	t.Run("Success", func(t *testing.T) {
		mockProfile.On("UploadAvatar", mock.Anything, mock.AnythingOfType("*profileproto.UploadAvatarRequest")).
			Return(&profileproto.UploadAvatarResponse{
				AvatarPath: "/avatars/new-avatar.jpg",
			}, nil)

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, _ := writer.CreateFormFile("avatar", "test.jpg")
		part.Write([]byte("fake image data"))
		writer.Close()

		req := createRequestWithToken("POST", "/user/upload/avatar", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		w := httptest.NewRecorder()

		server.uploadAvatarHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})

	t.Run("NoFileProvided", func(t *testing.T) {
		req := createRequestWithToken("POST", "/user/upload/avatar", strings.NewReader(""))
		req.Header.Set("Content-Type", "multipart/form-data")
		w := httptest.NewRecorder()

		server.uploadAvatarHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
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
}

func TestMapProfile_Nil(t *testing.T) {
	result := mapProfile(nil)

	assert.Equal(t, "", result.Username)
	assert.Equal(t, "", result.Name)
	assert.Equal(t, "", result.Surname)
}

func TestServer_RefreshHandler(t *testing.T) {
	server, mockAuth, _, _ := setupTestServer()

	t.Run("Success", func(t *testing.T) {
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
		req := httptest.NewRequest("POST", "/auth/refresh", nil)
		w := httptest.NewRecorder()

		server.refreshHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestServer_LogoutHandler(t *testing.T) {
	server, mockAuth, _, _ := setupTestServer()

	t.Run("Success", func(t *testing.T) {
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
}

func TestWriteGrpcAwareError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		defaultMessage string
		expectedStatus int
	}{
		{
			name:           "Unauthenticated",
			err:            status.Error(codes.Unauthenticated, "invalid token"),
			defaultMessage: "Authentication failed",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "InvalidArgument",
			err:            status.Error(codes.InvalidArgument, "invalid input"),
			defaultMessage: "Invalid request",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "NotFound",
			err:            status.Error(codes.NotFound, "not found"),
			defaultMessage: "Resource not found",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "PermissionDenied",
			err:            status.Error(codes.PermissionDenied, "access denied"),
			defaultMessage: "Access denied",
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "AlreadyExists",
			err:            status.Error(codes.AlreadyExists, "already exists"),
			defaultMessage: "Conflict",
			expectedStatus: http.StatusConflict,
		},
		{
			name:           "GenericError",
			err:            errors.New("generic error"),
			defaultMessage: "Internal error",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeGrpcAwareError(w, tt.err, tt.defaultMessage)

			resp := w.Result()
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

func TestServer_ReplyHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

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

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("Reply", mock.Anything, mock.AnythingOfType("*messagesproto.ReplyRequest")).
			Return(&messagesproto.ReplyResponse{
				MessageId: "456",
			}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(replyRequest)

		req := createRequestWithToken("POST", "/messages/reply", &body)
		w := httptest.NewRecorder()

		server.replyHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})

	t.Run("InvalidRequestBody", func(t *testing.T) {
		req := createRequestWithToken("POST", "/messages/reply", strings.NewReader("invalid json"))
		w := httptest.NewRecorder()

		server.replyHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestServer_SettingsHandler(t *testing.T) {
	server, _, mockProfile, _ := setupTestServer()

	t.Run("Success", func(t *testing.T) {
		mockProfile.On("Settings", mock.Anything, mock.AnythingOfType("*profileproto.SettingsRequest")).
			Return(&profileproto.SettingsResponse{
				Settings: &profileproto.Settings{
					Theme: "dark",
				},
			}, nil)

		req := createRequestWithToken("GET", "/user/settings", nil)
		w := httptest.NewRecorder()

		server.settingsHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockProfile.AssertExpectations(t)
	})
}

func TestServer_SendDraftHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	sendDraftRequest := map[string]interface{}{
		"draft_id": "123",
	}

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("SendDraft", mock.Anything, mock.AnythingOfType("*messagesproto.SendDraftRequest")).
			Return(&messagesproto.SendDraftResponse{
				Success:   true,
				MessageId: "456",
			}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(sendDraftRequest)

		req := createRequestWithToken("POST", "/messages/send-draft", &body)
		w := httptest.NewRecorder()

		server.sendDraftHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_DeleteFolderHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("DeleteFolder", mock.Anything, mock.AnythingOfType("*messagesproto.DeleteFolderRequest")).
			Return(&messagesproto.DeleteFolderResponse{}, nil)

		req := createRequestWithToken("DELETE", "/messages/delete-folder?folder_id=1", nil)
		w := httptest.NewRecorder()

		server.deleteFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}

func TestServer_DeleteMessageFromFolderHandler(t *testing.T) {
	server, _, _, mockMessage := setupTestServer()

	deleteRequest := map[string]interface{}{
		"message_id": "123",
		"folder_id":  "1",
	}

	t.Run("Success", func(t *testing.T) {
		mockMessage.On("DeleteMessageFromFolder", mock.Anything, mock.AnythingOfType("*messagesproto.DeleteMessageFromFolderRequest")).
			Return(&messagesproto.DeleteMessageFromFolderResponse{}, nil)

		var body bytes.Buffer
		json.NewEncoder(&body).Encode(deleteRequest)

		req := createRequestWithToken("DELETE", "/messages/delete-message-from-folder", &body)
		w := httptest.NewRecorder()

		server.deleteMessageFromFolderHandler(w, req)

		resp := w.Result()
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		mockMessage.AssertExpectations(t)
	})
}
