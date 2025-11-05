package sent_test

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
	"2025_2_a4code/internal/http-server/handlers/messages/sent"
	"2025_2_a4code/internal/http-server/handlers/messages/sent/mocks"

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

func TestHandlerSent_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := sent.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()
	testMessages := []domain.Message{
		{
			ID: "1",
			Sender: domain.Sender{
				Email:    "sender1@example.com",
				Username: "sender1",
				Avatar:   "avatar1.jpg",
			},
			Snippet:  "This is a test message snippet 1",
			Datetime: testTime.Add(-2 * time.Hour),
			IsRead:   true,
		},
		{
			ID: "2",
			Sender: domain.Sender{
				Email:    "sender2@example.com",
				Username: "sender2",
				Avatar:   "avatar2.jpg",
			},
			Topic:    "Test Topic 2",
			Snippet:  "This is a test message snippet 2",
			Datetime: testTime.Add(-1 * time.Hour),
			IsRead:   false,
		},
	}

	testMessagesInfo := domain.Messages{
		MessageTotal:  100,
		MessageUnread: 25,
	}

	tests := []struct {
		name             string
		method           string
		queryParams      map[string]string
		setupRequest     func() *http.Request
		setupMocks       func()
		expectedStatus   int
		validateResponse func(t *testing.T, body string)
	}{
		{
			name:   "GET - Success get sent messages with default pagination",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/sent", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
					Return(testMessages, nil)

				mockMessageUsecase.EXPECT().
					GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
					Return(testMessagesInfo, nil)

				presignedURL1, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
					Return(presignedURL1, nil)

				presignedURL2, _ := url.Parse("https://storage.example.com/avatar2.jpg?signature=def")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar2.jpg", 15*time.Minute).
					Return(presignedURL2, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Status  int                `json:"status"`
					Message string             `json:"message"`
					Body    sent.InboxResponse `json:"body"`
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

				if response.Body.MessageTotal != testMessagesInfo.MessageTotal {
					t.Errorf("Expected message total %d, got %d", testMessagesInfo.MessageTotal, response.Body.MessageTotal)
				}

				if response.Body.MessageUnread != testMessagesInfo.MessageUnread {
					t.Errorf("Expected message unread %d, got %d", testMessagesInfo.MessageUnread, response.Body.MessageUnread)
				}

				if len(response.Body.Messages) != len(testMessages) {
					t.Errorf("Expected %d messages, got %d", len(testMessages), len(response.Body.Messages))
				}

				for i, msg := range response.Body.Messages {
					if msg.ID != testMessages[i].ID {
						t.Errorf("Message %d: expected ID %s, got %s", i, testMessages[i].ID, msg.ID)
					}
					if msg.Topic != testMessages[i].Topic {
						t.Errorf("Message %d: expected topic %s, got %s", i, testMessages[i].Topic, msg.Topic)
					}
					if msg.Snippet != testMessages[i].Snippet {
						t.Errorf("Message %d: expected snippet %s, got %s", i, testMessages[i].Snippet, msg.Snippet)
					}
					if !msg.Datetime.Equal(testMessages[i].Datetime) {
						t.Errorf("Message %d: expected datetime %v, got %v", i, testMessages[i].Datetime, msg.Datetime)
					}
					if msg.IsRead != testMessages[i].IsRead {
						t.Errorf("Message %d: expected isRead %v, got %v", i, testMessages[i].IsRead, msg.IsRead)
					}
				}

				if !response.Body.Pagination.HasNext {
					t.Errorf("Expected pagination hasNext to be true")
				}
			},
		},
		{
			name:   "GET - Success get sent messages with custom pagination",
			method: http.MethodGet,
			queryParams: map[string]string{
				"last_message_id": "50",
				"last_datetime":   testTime.Add(-3 * time.Hour).Format(time.RFC3339),
				"limit":           "10",
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/sent", nil)
				q := req.URL.Query()
				q.Add("last_message_id", "50")
				q.Add("last_datetime", testTime.Add(-3*time.Hour).Format(time.RFC3339))
				q.Add("limit", "10")
				req.URL.RawQuery = q.Encode()
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(50), testTime.Add(-3*time.Hour), 10).
					Return(testMessages, nil)

				mockMessageUsecase.EXPECT().
					GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
					Return(testMessagesInfo, nil)

				presignedURL1, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
					Return(presignedURL1, nil)

				presignedURL2, _ := url.Parse("https://storage.example.com/avatar2.jpg?signature=def")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar2.jpg", 15*time.Minute).
					Return(presignedURL2, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "GET - Unauthorized",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/messages/sent", nil)
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "POST - Method not allowed",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("POST", "/messages/sent", nil)
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
				req := httptest.NewRequest("PUT", "/messages/sent", nil)
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
				req := httptest.NewRequest("DELETE", "/messages/sent", nil)
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
			name:   "GET - Find messages error",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/sent", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "GET - Get messages info error",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/messages/sent", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
					Return(testMessages, nil)

				mockMessageUsecase.EXPECT().
					GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
					Return(domain.Messages{}, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
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

func TestHandlerSent_PaginationEdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := sent.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()

	t.Run("GET - Invalid last_message_id parameter", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent?last_message_id=invalid", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "avatar1.jpg",
				},
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		trr := httptest.NewRecorder()
		handler.ServeHTTP(trr, req)

		if trr.Code != http.StatusOK {
			t.Errorf("Expected status %d for invalid last_message_id, got %d", http.StatusOK, trr.Code)
		}
	})

	t.Run("GET - Invalid last_datetime parameter", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent?last_datetime=invalid", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "avatar1.jpg",
				},
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for invalid last_datetime, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GET - Invalid limit parameter", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent?limit=invalid", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "avatar1.jpg",
				},
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for invalid limit, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GET - Limit parameter exceeds maximum", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent?limit=150", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "avatar1.jpg",
				},
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for limit exceeding maximum, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GET - Valid limit within range", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent?limit=50", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "avatar1.jpg",
				},
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 50).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for valid limit, got %d", http.StatusOK, rr.Code)
		}
	})
}

func TestHandlerSent_AvatarEnrichment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := sent.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()

	t.Run("GET - Message with full URL avatar", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "https://example.com/avatars/user123.jpg",
				},
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
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

	t.Run("GET - Message without avatar", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "",
				},
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var response struct {
			Body sent.InboxResponse `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Body.Messages[0].Sender.Avatar != "" {
			t.Errorf("Expected empty avatar, got %s", response.Body.Messages[0].Sender.Avatar)
		}
	})

	t.Run("GET - Avatar enrichment error", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "avatar.jpg",
				},
				Topic:    "Test Topic",
				Snippet:  "Test snippet",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar.jpg", 15*time.Minute).
			Return(nil, errors.New("avatar not found"))

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d when avatar enrichment fails, got %d", http.StatusOK, rr.Code)
		}
	})
}

func TestHandlerSent_PaginationLogic(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := sent.New(mockMessageUsecase, mockAvatarUsecase, secret)

	testTime := time.Now()

	t.Run("GET - Pagination hasNext true when full page received", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent?limit=2", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "avatar1.jpg",
				},
				Topic:    "Test Topic 1",
				Snippet:  "Test snippet 1",
				Datetime: testTime.Add(-2 * time.Hour),
				IsRead:   true,
			},
			{
				ID: "2",
				Sender: domain.Sender{
					Email:    "sender2@example.com",
					Username: "sender2",
					Avatar:   "avatar2.jpg",
				},
				Topic:    "Test Topic 2",
				Snippet:  "Test snippet 2",
				Datetime: testTime.Add(-1 * time.Hour),
				IsRead:   false,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 2).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL1, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL1, nil)

		presignedURL2, _ := url.Parse("https://storage.example.com/avatar2.jpg?signature=def")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar2.jpg", 15*time.Minute).
			Return(presignedURL2, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var response struct {
			Body sent.InboxResponse `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if !response.Body.Pagination.HasNext {
			t.Errorf("Expected hasNext to be true when full page received")
		}

		expectedLastMessageID, _ := strconv.ParseInt(testMessages[1].ID, 10, 64)
		if response.Body.Pagination.NextLastMessageID != expectedLastMessageID {
			t.Errorf("Expected NextLastMessageID %d, got %d", expectedLastMessageID, response.Body.Pagination.NextLastMessageID)
		}

		if response.Body.Pagination.NextLastDatetime != testMessages[1].Datetime.Format(time.RFC3339) {
			t.Errorf("Expected NextLastDatetime %s, got %s", testMessages[1].Datetime.Format(time.RFC3339), response.Body.Pagination.NextLastDatetime)
		}
	})

	t.Run("GET - Pagination hasNext false when partial page received", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent?limit=10", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{
			{
				ID: "1",
				Sender: domain.Sender{
					Email:    "sender1@example.com",
					Username: "sender1",
					Avatar:   "avatar1.jpg",
				},
				Topic:    "Test Topic 1",
				Snippet:  "Test snippet 1",
				Datetime: testTime,
				IsRead:   true,
			},
		}

		testMessagesInfo := domain.Messages{
			MessageTotal:  100,
			MessageUnread: 25,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 10).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		presignedURL, _ := url.Parse("https://storage.example.com/avatar1.jpg?signature=abc")
		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), "avatar1.jpg", 15*time.Minute).
			Return(presignedURL, nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var response struct {
			Body sent.InboxResponse `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response.Body.Pagination.HasNext {
			t.Errorf("Expected hasNext to be false when partial page received")
		}
	})

	t.Run("GET - Empty messages list", func(t *testing.T) {
		token := createTestToken(secret, 1)
		req := httptest.NewRequest("GET", "/messages/sent", nil)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		testMessages := []domain.Message{}
		testMessagesInfo := domain.Messages{
			MessageTotal:  0,
			MessageUnread: 0,
		}

		mockMessageUsecase.EXPECT().
			FindSentMessagesByProfileIDWithKeysetPagination(gomock.Any(), int64(1), int64(0), time.Time{}, 20).
			Return(testMessages, nil)

		mockMessageUsecase.EXPECT().
			GetSentMessagesInfoWithPagination(gomock.Any(), int64(1)).
			Return(testMessagesInfo, nil)

		mockAvatarUsecase.EXPECT().
			GetAvatarPresignedURL(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(0)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for empty messages, got %d", http.StatusOK, rr.Code)
		}

		var response struct {
			Body sent.InboxResponse `json:"body"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(response.Body.Messages) != 0 {
			t.Errorf("Expected 0 messages, got %d", len(response.Body.Messages))
		}

		if response.Body.MessageTotal != 0 {
			t.Errorf("Expected MessageTotal 0, got %d", response.Body.MessageTotal)
		}

		if response.Body.MessageUnread != 0 {
			t.Errorf("Expected MessageUnread 0, got %d", response.Body.MessageUnread)
		}

		if response.Body.Pagination.HasNext {
			t.Errorf("Expected hasNext to be false for empty messages")
		}
	})
}
