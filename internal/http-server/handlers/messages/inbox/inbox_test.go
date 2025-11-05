package inbox_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/handlers/messages/inbox"
	"2025_2_a4code/internal/http-server/handlers/messages/inbox/mocks"

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

func TestHandlerInbox_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := inbox.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()
	testMessages := []domain.Message{
		{
			ID:       "1",
			Topic:    "Test Topic 1",
			Snippet:  "This is a test snippet 1",
			Datetime: testTime,
			IsRead:   false,
			Sender: domain.Sender{
				Email:    "sender1@example.com",
				Username: "sender1",
				Avatar:   "avatar1.jpg",
			},
		},
		{
			ID:       "2",
			Topic:    "Test Topic 2",
			Snippet:  "This is a test snippet 2",
			Datetime: testTime.Add(-time.Hour),
			IsRead:   true,
			Sender: domain.Sender{
				Email:    "sender2@example.com",
				Username: "sender2",
				Avatar:   "",
			},
		},
	}

	testMessagesInfo := domain.Messages{
		MessageTotal:  10,
		MessageUnread: 3,
	}

	tests := []struct {
		name             string
		method           string
		setupRequest     func() *http.Request
		setupMocks       func()
		expectedStatus   int
		validateResponse func(t *testing.T, body string)
	}{
		//	{
		//		name:   "GET - Success get inbox",
		//		method: http.MethodGet,
		//		setupRequest: func() *http.Request {
		//			token := createTestToken(secret, 1)
		//			req := httptest.NewRequest("GET", "/messages/inbox", nil)
		//			req.AddCookie(&http.Cookie{
		//				Name:  "access_token",
		//				Value: token,
		//			})
		//			return req
		//		},
		//		setupMocks: func() {
		//			mockMessageUsecase.EXPECT().
		//				FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
		//				Return(testMessages, nil)
		//
		//			mockMessageUsecase.EXPECT().
		//				GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
		//				Return(testMessagesInfo, nil)
		//
		//			presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		//			mockAvatarUsecase.EXPECT().
		//				GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
		//				Return(presignedURL, nil)
		//		},
		//		expectedStatus: http.StatusOK,
		//		validateResponse: func(t *testing.T, body string) {
		//			var response struct {
		//				Status  int                 `json:"status"`
		//				Message string              `json:"message"`
		//				Body    inbox.InboxResponse `json:"body"`
		//			}
		//
		//			if err := json.Unmarshal([]byte(body), &response); err != nil {
		//				t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, body)
		//			}
		//
		//			if response.Status != http.StatusOK {
		//				t.Errorf("Response status = %d, want %d", response.Status, http.StatusOK)
		//			}
		//
		//			if response.Message != "success" {
		//				t.Errorf("Response message = %s, want 'success'", response.Message)
		//			}
		//
		//			if response.Body.MessageTotal != 10 {
		//				t.Errorf("Expected message total 10, got %d", response.Body.MessageTotal)
		//			}
		//
		//			if response.Body.MessageUnread != 3 {
		//				t.Errorf("Expected message unread 3, got %d", response.Body.MessageUnread)
		//			}
		//
		//			if len(response.Body.Messages) != 2 {
		//				t.Errorf("Expected 2 messages, got %d", len(response.Body.Messages))
		//			}
		//
		//			if !response.Body.Pagination.HasNext {
		//				t.Error("Expected pagination to have next page")
		//			}
		//		},
		//	},
		{
			name:   "GET - Success with pagination params",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/inbox?last_message_id=100&last_datetime=2023-01-01T00:00:00Z&limit=10", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				expectedTime, _ := time.Parse(time.RFC3339, "2023-01-01T00:00:00Z")
				mockMessageUsecase.EXPECT().
					FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(100), expectedTime, 10).
					Return(testMessages[:1], nil)

				mockMessageUsecase.EXPECT().
					GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
					Return(testMessagesInfo, nil)

				presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
					Return(presignedURL, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Body inbox.InboxResponse `json:"body"`
				}

				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Body.Pagination.HasNext {
					t.Error("Expected no next page when returned messages < limit")
				}
			},
		},
		{
			name:   "GET - Unauthorized",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/messages/inbox", nil)
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "GET - Invalid limit parameter",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/inbox?limit=invalid", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
					Return(testMessages, nil)

				mockMessageUsecase.EXPECT().
					GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
					Return(testMessagesInfo, nil)

				presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
					Return(presignedURL, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "GET - Message usecase error",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/inbox", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
					Return(nil, errors.New("message not found"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "GET - Messages info error",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/inbox", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
					Return(testMessages, nil)

				mockMessageUsecase.EXPECT().
					GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
					Return(domain.Messages{}, errors.New("message not found"))

				presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
					Return(presignedURL, nil)
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "POST - Method not allowed",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("POST", "/messages/inbox", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "PUT - Method not allowed",
			method: http.MethodPut,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("PUT", "/messages/inbox", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "DELETE - Method not allowed",
			method: http.MethodDelete,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("DELETE", "/messages/inbox", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()
			req := tt.setupRequest()
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

func TestHandlerInbox_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := inbox.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()
	testMessagesInfo := domain.Messages{
		MessageTotal:  5,
		MessageUnread: 2,
	}

	t.Run("GET - Avatar with full URL", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/inbox", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		messagesWithFullURL := []domain.Message{
			{
				ID:       "1",
				Topic:    "Test",
				Snippet:  "Test",
				Datetime: testTime,
				IsRead:   false,
				Sender: domain.Sender{
					Email:    "sender@example.com",
					Username: "sender",
					Avatar:   "https://example.com/avatars/user123.jpg",
				},
			},
		}

		mockMessageUsecase.EXPECT().
			FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(messagesWithFullURL, nil)

		mockMessageUsecase.EXPECT().
			GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

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

	t.Run("GET - Empty messages", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/inbox", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		mockMessageUsecase.EXPECT().
			FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return([]domain.Message{}, nil)

		mockMessageUsecase.EXPECT().
			GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for empty messages, got %d", http.StatusOK, rr.Code)
		}

		var response struct {
			Body inbox.InboxResponse `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Body.Messages) != 0 {
			t.Errorf("Expected 0 messages, got %d", len(response.Body.Messages))
		}

		if response.Body.Pagination.HasNext {
			t.Error("Expected no next page for empty messages")
		}
	})

	t.Run("GET - Invalid datetime format", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/inbox?last_datetime=invalid", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		mockMessageUsecase.EXPECT().
			FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return([]domain.Message{
				{
					ID:       "1",
					Topic:    "Test Topic",
					Snippet:  "Test snippet",
					Datetime: testTime,
					IsRead:   false,
					Sender: domain.Sender{
						Email:    "sender@example.com",
						Username: "sender",
						Avatar:   "avatar1.jpg",
					},
				},
			}, nil)

		mockMessageUsecase.EXPECT().
			GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for invalid datetime, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GET - Limit out of range", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/inbox?limit=150", nil) // Больше максимума
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		mockMessageUsecase.EXPECT().
			FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return([]domain.Message{
				{
					ID:       "1",
					Topic:    "Test Topic",
					Snippet:  "Test snippet",
					Datetime: testTime,
					IsRead:   false,
					Sender: domain.Sender{
						Email:    "sender@example.com",
						Username: "sender",
						Avatar:   "avatar1.jpg",
					},
				},
			}, nil)

		mockMessageUsecase.EXPECT().
			GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for out of range limit, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GET - Avatar enrichment error", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/inbox", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		mockMessageUsecase.EXPECT().
			FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return([]domain.Message{
				{
					ID:       "1",
					Topic:    "Test Topic",
					Snippet:  "Test snippet",
					Datetime: testTime,
					IsRead:   false,
					Sender: domain.Sender{
						Email:    "sender@example.com",
						Username: "sender",
						Avatar:   "avatar1.jpg",
					},
				},
			}, nil)

		mockMessageUsecase.EXPECT().
			GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(nil, errors.New("avatar not found"))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d when avatar enrichment fails, got %d", http.StatusOK, rr.Code)
		}
	})
}

func TestHandlerInbox_PaginationLogic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := inbox.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()
	testMessagesInfo := domain.Messages{
		MessageTotal:  25,
		MessageUnread: 5,
	}

	t.Run("GET - HasNext true when full page", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/inbox?limit=5", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		fullPageMessages := make([]domain.Message, 5)
		for i := 0; i < 5; i++ {
			fullPageMessages[i] = domain.Message{
				ID:       string(rune(i + 1)),
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime.Add(-time.Duration(i) * time.Hour),
				IsRead:   false,
				Sender: domain.Sender{
					Email:    "sender@example.com",
					Username: "sender",
					Avatar:   "",
				},
			}
		}

		mockMessageUsecase.EXPECT().
			FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 5).
			Return(fullPageMessages, nil)

		mockMessageUsecase.EXPECT().
			GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var response struct {
			Body inbox.InboxResponse `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response.Body.Pagination.HasNext {
			t.Error("Expected HasNext=true when returned full page")
		}
	})

	t.Run("GET - HasNext false when partial page", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/inbox?limit=10", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		partialPageMessages := make([]domain.Message, 3)
		for i := 0; i < 3; i++ {
			partialPageMessages[i] = domain.Message{
				ID:       string(rune(i + 1)),
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime.Add(-time.Duration(i) * time.Hour),
				IsRead:   false,
				Sender: domain.Sender{
					Email:    "sender@example.com",
					Username: "sender",
					Avatar:   "",
				},
			}
		}

		mockMessageUsecase.EXPECT().
			FindByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 10).
			Return(partialPageMessages, nil)

		mockMessageUsecase.EXPECT().
			GetMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var response struct {
			Body inbox.InboxResponse `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Body.Pagination.HasNext {
			t.Error("Expected HasNext=false when returned partial page")
		}
	})
}
