package reply

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"2025_2_a4code/internal/http-server/handlers/messages/reply/mocks"

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

func TestHandlerReply_ServeHTTP(t *testing.T) {
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
			name:   "POST - Success reply to message",
			method: http.MethodPost,
			requestBody: Request{
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          "Test reply text",
				ThreadRoot:    200,
				Receivers: []Receiver{
					{Email: "receiver1@example.com"},
				},
				Files: []File{},
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver1@example.com", int64(1), "Re: Test Topic", "Test reply text").
					Return(int64(300), nil)
				mockMessageUsecase.EXPECT().
					SaveThreadIdToMessage(gomock.Any(), int64(300), int64(200)).
					Return(nil)
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					RootMessageID: 100,
					Topic:         "Re: Test Topic",
					Text:          "Test reply text",
					ThreadRoot:    200,
					Receivers: []Receiver{
						{Email: "receiver1@example.com"},
					},
					Files: []File{},
				})
				req := httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "POST - Success reply with multiple receivers and files",
			method: http.MethodPost,
			requestBody: Request{
				RootMessageID: 100,
				Topic:         "Re: Test Topic with Files",
				Text:          "Test reply with files",
				ThreadRoot:    200,
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
					SaveMessage(gomock.Any(), "receiver1@example.com", int64(1), "Re: Test Topic with Files", "Test reply with files").
					Return(int64(301), nil)
				mockMessageUsecase.EXPECT().
					SaveThreadIdToMessage(gomock.Any(), int64(301), int64(200)).
					Return(nil)
				mockMessageUsecase.EXPECT().
					SaveFile(gomock.Any(), int64(301), "test.pdf", "application/pdf", "/uploads/test.pdf", int64(1024)).
					Return(int64(401), nil)

				// Second receiver
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver2@example.com", int64(1), "Re: Test Topic with Files", "Test reply with files").
					Return(int64(302), nil)
				mockMessageUsecase.EXPECT().
					SaveThreadIdToMessage(gomock.Any(), int64(302), int64(200)).
					Return(nil)
				mockMessageUsecase.EXPECT().
					SaveFile(gomock.Any(), int64(302), "test.pdf", "application/pdf", "/uploads/test.pdf", int64(1024)).
					Return(int64(402), nil)
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					RootMessageID: 100,
					Topic:         "Re: Test Topic with Files",
					Text:          "Test reply with files",
					ThreadRoot:    200,
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
				req := httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
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
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          "Test reply text",
				ThreadRoot:    200,
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			setupMocks: func() {},
			setupRequest: func() *http.Request {
				body, _ := json.Marshal(Request{
					RootMessageID: 100,
					Topic:         "Re: Test Topic",
					Text:          "Test reply text",
					ThreadRoot:    200,
					Receivers: []Receiver{
						{Email: "receiver@example.com"},
					},
				})
				return httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "POST - Empty text",
			method: http.MethodPost,
			requestBody: Request{
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          "",
				ThreadRoot:    200,
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			setupMocks: func() {},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					RootMessageID: 100,
					Topic:         "Re: Test Topic",
					Text:          "",
					ThreadRoot:    200,
					Receivers: []Receiver{
						{Email: "receiver@example.com"},
					},
				})
				req := httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
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
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          "Test reply text",
				ThreadRoot:    200,
				Receivers:     []Receiver{},
			},
			setupMocks: func() {},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					RootMessageID: 100,
					Topic:         "Re: Test Topic",
					Text:          "Test reply text",
					ThreadRoot:    200,
					Receivers:     []Receiver{},
				})
				req := httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
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
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          "Test reply text",
				ThreadRoot:    200,
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver@example.com", int64(1), "Re: Test Topic", "Test reply text").
					Return(int64(0), errors.New("database error"))
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					RootMessageID: 100,
					Topic:         "Re: Test Topic",
					Text:          "Test reply text",
					ThreadRoot:    200,
					Receivers: []Receiver{
						{Email: "receiver@example.com"},
					},
				})
				req := httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "POST - SaveThreadIdToMessage error",
			method: http.MethodPost,
			requestBody: Request{
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          "Test reply text",
				ThreadRoot:    200,
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			setupMocks: func() {
				mockMessageUsecase.EXPECT().
					SaveMessage(gomock.Any(), "receiver@example.com", int64(1), "Re: Test Topic", "Test reply text").
					Return(int64(300), nil)
				mockMessageUsecase.EXPECT().
					SaveThreadIdToMessage(gomock.Any(), int64(300), int64(200)).
					Return(errors.New("thread save error"))
			},
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, _ := json.Marshal(Request{
					RootMessageID: 100,
					Topic:         "Re: Test Topic",
					Text:          "Test reply text",
					ThreadRoot:    200,
					Receivers: []Receiver{
						{Email: "receiver@example.com"},
					},
				})
				req := httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
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
			setupRequest:   func() *http.Request { return httptest.NewRequest("GET", "/messages/reply", nil) },
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

func TestHandlerReply_Validation(t *testing.T) {
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
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          "Test reply text",
				ThreadRoot:    200,
				Receivers: []Receiver{
					{Email: "invalid-email"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Duplicate receivers",
			requestBody: Request{
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          "Test reply text",
				ThreadRoot:    200,
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
				RootMessageID: 100,
				Topic:         string(make([]byte, 256)),
				Text:          "Test reply text",
				ThreadRoot:    200,
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "Text too long",
			requestBody: Request{
				RootMessageID: 100,
				Topic:         "Re: Test Topic",
				Text:          string(make([]byte, 10001)),
				ThreadRoot:    200,
				Receivers: []Receiver{
					{Email: "receiver@example.com"},
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			token := createTestToken(secret, 1)
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
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

func TestHandlerReply_ThreadRoot_Handling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockMessageUsecase := mocks.NewMockMessageUsecase(ctrl)
	secret := []byte("test-secret")

	handler := New(mockMessageUsecase, secret)

	t.Run("POST - Correct ThreadRoot passed to SaveThreadIdToMessage", func(t *testing.T) {
		threadRoot := int64(999)

		mockMessageUsecase.EXPECT().
			SaveMessage(gomock.Any(), "receiver@example.com", int64(1), "Re: Test", "Test reply").
			Return(int64(300), nil)
		mockMessageUsecase.EXPECT().
			SaveThreadIdToMessage(gomock.Any(), int64(300), threadRoot).
			Return(nil)

		token := createTestToken(secret, 1)
		body, _ := json.Marshal(Request{
			RootMessageID: 100,
			Topic:         "Re: Test",
			Text:          "Test reply",
			ThreadRoot:    threadRoot,
			Receivers: []Receiver{
				{Email: "receiver@example.com"},
			},
			Files: []File{},
		})
		req := httptest.NewRequest("POST", "/messages/reply", bytes.NewReader(body))
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})
}
