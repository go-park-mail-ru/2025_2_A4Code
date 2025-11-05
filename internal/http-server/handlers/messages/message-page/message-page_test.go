package message_page_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"2025_2_a4code/internal/domain"
	message_page "2025_2_a4code/internal/http-server/handlers/messages/message-page"
	"2025_2_a4code/internal/http-server/handlers/messages/message-page/mocks"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang/mock/gomock"
)

func createTestToken(secret []byte, userID int64) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour).Unix(),
		"type":    "access",
	})
	tokenString, _ := token.SignedString(secret)
	return tokenString
}

func TestHandlerMessagePage_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := message_page.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()
	testFullMessage := domain.FullMessage{
		Topic:    "Test Topic",
		Text:     "This is a test message content",
		Datetime: testTime,
		Sender: domain.Sender{
			Email:    "sender@example.com",
			Username: "sender",
			Avatar:   "avatar.jpg",
		},
		ThreadRoot: "thread-123",
		Files: []domain.File{
			{
				Name:     "document.pdf",
				FileType: "application/pdf",
				Size:     1024,
			},
			{
				Name:     "image.jpg",
				FileType: "image/jpeg",
				Size:     2048,
			},
		},
	}

	tests := []struct {
		name             string
		method           string
		messageID        string
		setupRequest     func(messageID string) *http.Request
		setupMocks       func(messageID int)
		expectedStatus   int
		validateResponse func(t *testing.T, body string)
	}{
		{
			name:      "GET - Success get message",
			method:    http.MethodGet,
			messageID: "123",
			setupRequest: func(messageID string) *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/"+messageID, nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func(messageID int) {
				mockMessageUsecase.EXPECT().
					FindFullByMessageID(gomock.Any(), int64(messageID), int64(1)).
					Return(testFullMessage, nil)

				mockMessageUsecase.EXPECT().
					MarkMessageAsRead(gomock.Any(), int64(messageID), int64(1)).
					Return(nil)

				presignedURL, _ := url.Parse("https://storage.example.com/avatar.jpg?signature=abc")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar.jpg", 15*time.Minute).
					Return(presignedURL, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Status  int                  `json:"status"`
					Message string               `json:"message"`
					Body    message_page.Message `json:"body"`
				}

				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, body)
				}

				if response.Status != http.StatusOK {
					t.Errorf("Response status = %d, want %d", response.Status, http.StatusOK)
				}

				if response.Message != "success" {
					t.Errorf("Response message = %s, want 'success'", response.Message)
				}

				if response.Body.Topic != testFullMessage.Topic {
					t.Errorf("Expected topic %s, got %s", testFullMessage.Topic, response.Body.Topic)
				}

				if response.Body.Text != testFullMessage.Text {
					t.Errorf("Expected text %s, got %s", testFullMessage.Text, response.Body.Text)
				}

				if !response.Body.Datetime.Equal(testFullMessage.Datetime) {
					t.Errorf("Expected datetime %v, got %v", testFullMessage.Datetime, response.Body.Datetime)
				}

				if response.Body.ThreadId != testFullMessage.ThreadRoot {
					t.Errorf("Expected thread ID %s, got %s", testFullMessage.ThreadRoot, response.Body.ThreadId)
				}

				if response.Body.Sender.Email != testFullMessage.Email {
					t.Errorf("Expected sender email %s, got %s", testFullMessage.Email, response.Body.Sender.Email)
				}

				if response.Body.Sender.Username != testFullMessage.Username {
					t.Errorf("Expected sender username %s, got %s", testFullMessage.Username, response.Body.Sender.Username)
				}

				if response.Body.Sender.Avatar != "https://storage.example.com/avatar.jpg?signature=abc" {
					t.Errorf("Expected avatar URL %s, got %s", "https://storage.example.com/avatar.jpg?signature=abc", response.Body.Sender.Avatar)
				}

				if len(response.Body.Files) != len(testFullMessage.Files) {
					t.Errorf("Expected %d files, got %d", len(testFullMessage.Files), len(response.Body.Files))
				}
			},
		},
		{
			name:      "GET - Message not found",
			method:    http.MethodGet,
			messageID: "999",
			setupRequest: func(messageID string) *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/"+messageID, nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func(messageID int) {
				mockMessageUsecase.EXPECT().
					FindFullByMessageID(gomock.Any(), int64(messageID), int64(1)).
					Return(domain.FullMessage{}, errors.New("message not found"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "GET - Unauthorized",
			method:    http.MethodGet,
			messageID: "123",
			setupRequest: func(messageID string) *http.Request {
				return httptest.NewRequest("GET", "/messages/"+messageID, nil)
			},
			setupMocks:     func(messageID int) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:      "GET - Invalid message ID",
			method:    http.MethodGet,
			messageID: "invalid",
			setupRequest: func(messageID string) *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/"+messageID, nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func(messageID int) {},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:      "POST - Method not allowed",
			method:    http.MethodPost,
			messageID: "123",
			setupRequest: func(messageID string) *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("POST", "/messages/"+messageID, nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func(messageID int) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:      "PUT - Method not allowed",
			method:    http.MethodPut,
			messageID: "123",
			setupRequest: func(messageID string) *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("PUT", "/messages/"+messageID, nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func(messageID int) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:      "DELETE - Method not allowed",
			method:    http.MethodDelete,
			messageID: "123",
			setupRequest: func(messageID string) *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("DELETE", "/messages/"+messageID, nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func(messageID int) {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:      "GET - Mark as read error",
			method:    http.MethodGet,
			messageID: "123",
			setupRequest: func(messageID string) *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/"+messageID, nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func(messageID int) {
				mockMessageUsecase.EXPECT().
					FindFullByMessageID(gomock.Any(), int64(messageID), int64(1)).
					Return(testFullMessage, nil)

				mockMessageUsecase.EXPECT().
					MarkMessageAsRead(gomock.Any(), int64(messageID), int64(1)).
					Return(errors.New("failed to mark as read"))

				presignedURL, _ := url.Parse("https://storage.example.com/avatar.jpg?signature=abc")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar.jpg", 15*time.Minute).
					Return(presignedURL, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageID, _ := strconv.Atoi(tt.messageID)
			tt.setupMocks(messageID)
			req := tt.setupRequest(tt.messageID)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if tt.validateResponse != nil && rr.Body.Len() > 0 {
				tt.validateResponse(t, rr.Body.String())
			}
		})
	}
}

func TestHandlerMessagePage_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := message_page.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()

	t.Run("GET - Message with full URL avatar", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/123", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		fullMessage := domain.FullMessage{
			Topic:    "Test Topic",
			Text:     "Test content",
			Datetime: testTime,
			Sender: domain.Sender{
				Email:    "sender@example.com",
				Username: "sender",
				Avatar:   "https://example.com/avatars/user123.jpg",
			},
			ThreadRoot: "thread-123",
			Files:      []domain.File{},
		}

		mockMessageUsecase.EXPECT().
			FindFullByMessageID(gomock.Any(), int64(123), int64(1)).
			Return(fullMessage, nil)

		mockMessageUsecase.EXPECT().
			MarkMessageAsRead(gomock.Any(), int64(123), int64(1)).
			Return(nil)

		presignedURL, _ := url.Parse("https://storage.example.com/user123.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "user123.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GET - Message without avatar", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/123", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		fullMessage := domain.FullMessage{
			Topic:    "Test Topic",
			Text:     "Test content",
			Datetime: testTime,
			Sender: domain.Sender{
				Email:    "sender@example.com",
				Username: "sender",
				Avatar:   "",
			},
			ThreadRoot: "thread-123",
			Files:      []domain.File{},
		}

		mockMessageUsecase.EXPECT().
			FindFullByMessageID(gomock.Any(), int64(123), int64(1)).
			Return(fullMessage, nil)

		mockMessageUsecase.EXPECT().
			MarkMessageAsRead(gomock.Any(), int64(123), int64(1)).
			Return(nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var response struct {
			Body message_page.Message `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Body.Sender.Avatar != "" {
			t.Errorf("Expected empty avatar, got %s", response.Body.Sender.Avatar)
		}
	})

	t.Run("GET - Avatar enrichment error", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/123", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		fullMessage := domain.FullMessage{
			Topic:    "Test Topic",
			Text:     "Test content",
			Datetime: testTime,
			Sender: domain.Sender{
				Email:    "sender@example.com",
				Username: "sender",
				Avatar:   "avatar.jpg",
			},
			ThreadRoot: "thread-123",
			Files:      []domain.File{},
		}

		mockMessageUsecase.EXPECT().
			FindFullByMessageID(gomock.Any(), int64(123), int64(1)).
			Return(fullMessage, nil)

		mockMessageUsecase.EXPECT().
			MarkMessageAsRead(gomock.Any(), int64(123), int64(1)).
			Return(nil)

		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar.jpg", 15*time.Minute).
			Return(nil, errors.New("avatar not found"))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d when avatar enrichment fails, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GET - Message with trailing slash", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/123/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		fullMessage := domain.FullMessage{
			Topic:    "Test Topic",
			Text:     "Test content",
			Datetime: testTime,
			Sender: domain.Sender{
				Email:    "sender@example.com",
				Username: "sender",
				Avatar:   "avatar.jpg",
			},
			ThreadRoot: "thread-123",
			Files:      []domain.File{},
		}

		mockMessageUsecase.EXPECT().
			FindFullByMessageID(gomock.Any(), int64(123), int64(1)).
			Return(fullMessage, nil)

		mockMessageUsecase.EXPECT().
			MarkMessageAsRead(gomock.Any(), int64(123), int64(1)).
			Return(nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for URL with trailing slash, got %d", http.StatusOK, rr.Code)
		}
	})
}

func TestHandlerMessagePage_FileHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := message_page.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()

	t.Run("GET - Message with multiple files", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/123", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		fullMessage := domain.FullMessage{
			Topic:    "Test Topic",
			Text:     "Test content with files",
			Datetime: testTime,
			Sender: domain.Sender{
				Email:    "sender@example.com",
				Username: "sender",
				Avatar:   "avatar.jpg",
			},
			ThreadRoot: "thread-123",
			Files: []domain.File{
				{
					Name:        "document.pdf",
					FileType:    "application/pdf",
					Size:        1024,
					StoragePath: "/storage/docs/doc1.pdf",
				},
				{
					Name:        "presentation.pptx",
					FileType:    "application/vnd.ms-powerpoint",
					Size:        2048,
					StoragePath: "/storage/docs/pres1.pptx",
				},
				{
					Name:        "spreadsheet.xlsx",
					FileType:    "application/vnd.ms-excel",
					Size:        3072,
					StoragePath: "/storage/docs/sheet1.xlsx",
				},
			},
		}

		mockMessageUsecase.EXPECT().
			FindFullByMessageID(gomock.Any(), int64(123), int64(1)).
			Return(fullMessage, nil)

		mockMessageUsecase.EXPECT().
			MarkMessageAsRead(gomock.Any(), int64(123), int64(1)).
			Return(nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var response struct {
			Body message_page.Message `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Body.Files) != 3 {
			t.Errorf("Expected 3 files, got %d", len(response.Body.Files))
		}

		for i, file := range response.Body.Files {
			if file.Name != fullMessage.Files[i].Name {
				t.Errorf("File %d: expected name %s, got %s", i, fullMessage.Files[i].Name, file.Name)
			}
			if file.FileType != fullMessage.Files[i].FileType {
				t.Errorf("File %d: expected type %s, got %s", i, fullMessage.Files[i].FileType, file.FileType)
			}
			if file.Size != fullMessage.Files[i].Size {
				t.Errorf("File %d: expected size %d, got %d", i, fullMessage.Files[i].Size, file.Size)
			}
			if file.StoragePath != "" {
				t.Errorf("File %d: expected empty StoragePath, got %s", i, file.StoragePath)
			}
		}
	})

	t.Run("GET - Message without files", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/123", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		fullMessage := domain.FullMessage{
			Topic:    "Test Topic",
			Text:     "Test content without files",
			Datetime: testTime,
			Sender: domain.Sender{
				Email:    "sender@example.com",
				Username: "sender",
				Avatar:   "avatar.jpg",
			},
			ThreadRoot: "thread-123",
			Files:      []domain.File{},
		}

		mockMessageUsecase.EXPECT().
			FindFullByMessageID(gomock.Any(), int64(123), int64(1)).
			Return(fullMessage, nil)

		mockMessageUsecase.EXPECT().
			MarkMessageAsRead(gomock.Any(), int64(123), int64(1)).
			Return(nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var response struct {
			Body message_page.Message `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Body.Files) != 0 {
			t.Errorf("Expected 0 files, got %d", len(response.Body.Files))
		}
	})
}
