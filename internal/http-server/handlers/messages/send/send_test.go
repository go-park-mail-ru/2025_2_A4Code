package send

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"2025_2_a4code/internal/http-server/handlers/messages/send/mocks"

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

func TestHandlerSend_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	secret := []byte("test-secret")

	handler := New(mockMessageUsecase, secret)

	tests := []struct {
		name           string
		method         string
		requestBody    interface{}
		setupMocks     func()
		setupRequest   func() *http.Request
		expectedStatus int
	}{
		{
			name:   "POST - Success send message",
			method: http.MethodPost,
			requestBody: Request{
				Topic: "Test Topic",
				Text:  "Test message text",
				Receivers: []Receiver{
					{Email: "receiver1@example.com"},
				},
				Files: []File{},
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver1@example.com", int64(1), "Test Topic", "Test message text").
					Return(int64(100), nil)
				mockMessageUsecase.EXPECT().
					SaveThread(gomock.Any(), int64(100)).
					Return(int64(200), nil)
				mockMessageUsecase.EXPECT().
					SaveThreadIdToMessage(gomock.Any(), int64(100), int64(200)).
					Return(nil)
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					Topic: "Test Topic",
					Text:  "Test message text",
					Receivers: []Receiver{
						{Email: "receiver1@example.com"},
					},
					Files: []File{},
				})
				req := httptest.NewRequest("POST", "/messages/send", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST - Success with multiple receivers and files",
			method: http.MethodPost,
			requestBody: Request{
				Topic: "Test Topic with Files",
				Text:  "Test message with files",
				Receivers: []Receiver{
					{Email: "receiver1@example.com"},
					{Email: "receiver2@example.com"},
				},
				Files: []File{
					{
						Name:        "test.pdf",
						FileType:    "application/pdf",
						Size:        1024,
						StoragePath: "/uploads/test.pdf",
					},
				},
			},
			setupMocks: func() {
				// First receiver
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver1@example.com", int64(1), "Test Topic with Files", "Test message with files").
					Return(int64(101), nil)
				mockMessageUsecase.EXPECT().
					SaveThread(gomock.Any(), int64(101)).
					Return(int64(201), nil)
				mockMessageUsecase.EXPECT().
					SaveThreadIdToMessage(gomock.Any(), int64(101), int64(201)).
					Return(nil)
				mockMessageUsecase.EXPECT().
					SaveFile(gomock.Any(), int64(101), "test.pdf", "application/pdf", "/uploads/test.pdf", int64(1024)).
					Return(int64(301), nil)

				// Second receiver
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver2@example.com", int64(1), "Test Topic with Files", "Test message with files").
					Return(int64(102), nil)
				mockMessageUsecase.EXPECT().
					SaveThread(gomock.Any(), int64(102)).
					Return(int64(202), nil)
				mockMessageUsecase.EXPECT().
					SaveThreadIdToMessage(gomock.Any(), int64(102), int64(202)).
					Return(nil)
				mockMessageUsecase.EXPECT().
					SaveFile(gomock.Any(), int64(102), "test.pdf", "application/pdf", "/uploads/test.pdf", int64(1024)).
					Return(int64(302), nil)
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					Topic: "Test Topic with Files",
					Text:  "Test message with files",
					Receivers: []Receiver{
						{Email: "receiver1@example.com"},
						{Email: "receiver2@example.com"},
					},
					Files: []File{
						{
							Name:        "test.pdf",
							FileType:    "application/pdf",
							Size:        1024,
							StoragePath: "/uploads/test.pdf",
						},
					},
				})
				req := httptest.NewRequest("POST", "/messages/send", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST - Unauthorized",
			method: http.MethodPost,
			requestBody: Request{
				Topic: "Test Topic",
				Text:  "Test message text",
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			setupMocks: func() {},
			setupRequest: func() *http.Request {
				body, _ := json.Marshal(Request{
					Topic: "Test Topic",
					Text:  "Test message text",
					Receivers: []Receiver{
						{Email: "receiver@example.com"},
					},
				})
				return httptest.NewRequest("POST", "/messages/send", bytes.NewReader(body))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "POST - Empty text",
			method: http.MethodPost,
			requestBody: Request{
				Topic: "Test Topic",
				Text:  "",
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			setupMocks: func() {},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					Topic: "Test Topic",
					Text:  "",
					Receivers: []Receiver{
						{Email: "receiver@example.com"},
					},
				})
				req := httptest.NewRequest("POST", "/messages/send", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST - Empty receivers",
			method: http.MethodPost,
			requestBody: Request{
				Topic:     "Test Topic",
				Text:      "Test message text",
				Receivers: []Receiver{},
			},
			setupMocks: func() {},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					Topic:     "Test Topic",
					Text:      "Test message text",
					Receivers: []Receiver{},
				})
				req := httptest.NewRequest("POST", "/messages/send", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "POST - SaveMessage error",
			method: http.MethodPost,
			requestBody: Request{
				Topic: "Test Topic",
				Text:  "Test message text",
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver@example.com", int64(1), "Test Topic", "Test message text").
					Return(int64(0), errors.New("database error"))
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					Topic: "Test Topic",
					Text:  "Test message text",
					Receivers: []Receiver{
						{Email: "receiver@example.com"},
					},
				})
				req := httptest.NewRequest("POST", "/messages/send", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "POST - SaveThread error",
			method: http.MethodPost,
			requestBody: Request{
				Topic: "Test Topic",
				Text:  "Test message text",
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver@example.com", int64(1), "Test Topic", "Test message text").
					Return(int64(100), nil)
				mockMessageUsecase.EXPECT().
					SaveThread(gomock.Any(), int64(100)).
					Return(int64(0), errors.New("thread creation error"))
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					Topic: "Test Topic",
					Text:  "Test message text",
					Receivers: []Receiver{
						{Email: "receiver@example.com"},
					},
				})
				req := httptest.NewRequest("POST", "/messages/send", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "GET - Method not allowed",
			method:         http.MethodGet,
			requestBody:    nil,
			setupMocks:     func() {},
			setupRequest:   func() *http.Request { return httptest.NewRequest("GET", "/messages/send", nil) },
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
		})
	}
}

func TestHandlerSend_Validation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	secret := []byte("test-secret")

	handler := New(mockMessageUsecase, secret)

	validationTests := []struct {
		name           string
		requestBody    Request
		expectedStatus int
	}{
		{
			name: "Invalid email format",
			requestBody: Request{
				Topic: "Test Topic",
				Text:  "Test message text",
				Receivers: []Receiver{
					{Email: "invalid-email"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate receivers",
			requestBody: Request{
				Topic: "Test Topic",
				Text:  "Test message text",
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
					{Email: "receiver@example.com"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Topic too long",
			requestBody: Request{
				Topic: string(make([]byte, 256)),
				Text:  "Test message text",
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Text too long",
			requestBody: Request{
				Topic: "Test Topic",
				Text:  string(make([]byte, 10001)),
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Too many files",
			requestBody: Request{
				Topic: "Test Topic",
				Text:  "Test message text",
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
				Files: make([]File, 21),
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			token := createTestToken(secret, 1)
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/messages/send", bytes.NewReader(body))
			req.AddCookie(&http.Cookie{
				Name:  "access_token",
				Value: token,
			})

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Test '%s': Expected status %d, got %d. Body: %s",
					tt.name, tt.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
