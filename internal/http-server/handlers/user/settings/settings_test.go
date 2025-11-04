package settings_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/handlers/user/settings"
	"2025_2_a4code/internal/http-server/handlers/user/settings/mocks"

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

func TestHandlerSettings_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileUsecase := mocks.NewMockProfileUsecase(ctrl)
	secret := []byte("test-secret")

	handler := settings.New(mockProfileUsecase, secret)

	tests := []struct {
		name             string
		method           string
		setupRequest     func() *http.Request
		setupMocks       func()
		expectedStatus   int
		validateResponse func(t *testing.T, body string)
	}{
		{
			name:   "GET - Success get settings",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/user/settings", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					FindSettingsByProfileId(gomock.Any(), int64(1)).
					Return(domain.Settings{
						NotificationTolerance: "30",
						Language:              "en",
						Theme:                 "dark",
						Signatures:            []string{"Best regards", "Thanks"},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Status  int               `json:"status"`
					Message string            `json:"message"`
					Body    settings.Settings `json:"body"`
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

				if response.Body.NotificationTolerance != "30" {
					t.Errorf("Expected notification tolerance '30', got '%s'", response.Body.NotificationTolerance)
				}

				if response.Body.Language != "en" {
					t.Errorf("Expected language 'en', got '%s'", response.Body.Language)
				}

				if response.Body.Theme != "dark" {
					t.Errorf("Expected theme 'dark', got '%s'", response.Body.Theme)
				}

				if len(response.Body.Signatures) != 2 {
					t.Errorf("Expected 2 signatures, got %d", len(response.Body.Signatures))
				}
			},
		},
		{
			name:   "GET - Success with empty signatures",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/user/settings", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					FindSettingsByProfileId(gomock.Any(), int64(1)).
					Return(domain.Settings{
						NotificationTolerance: "15",
						Language:              "ru",
						Theme:                 "light",
						Signatures:            []string{},
					}, nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Body settings.Settings `json:"body"`
				}

				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(response.Body.Signatures) != 0 {
					t.Errorf("Expected empty signatures, got %v", response.Body.Signatures)
				}
			},
		},
		{
			name:   "GET - Unauthorized - no cookie",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/user/settings", nil)
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Status  int    `json:"status"`
					Message string `json:"message"`
				}

				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Status != http.StatusUnauthorized {
					t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, response.Status)
				}
			},
		},
		{
			name:   "GET - Unauthorized - invalid token",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/user/settings", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: "invalid_token",
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "GET - Internal server error - usecase error",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/user/settings", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					FindSettingsByProfileId(gomock.Any(), int64(1)).
					Return(domain.Settings{}, errors.New("settings not found"))
			},
			expectedStatus: http.StatusInternalServerError,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Status  int    `json:"status"`
					Message string `json:"message"`
				}

				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Status != http.StatusInternalServerError {
					t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, response.Status)
				}

				if response.Message != "something went wrong" {
					t.Errorf("Expected message 'something went wrong', got '%s'", response.Message)
				}
			},
		},
		{
			name:   "POST - Method not allowed",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("POST", "/user/settings", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Status  int    `json:"status"`
					Message string `json:"message"`
				}

				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Status != http.StatusMethodNotAllowed {
					t.Errorf("Expected status %d, got %d", http.StatusMethodNotAllowed, response.Status)
				}
			},
		},
		{
			name:   "PUT - Method not allowed",
			method: http.MethodPut,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("PUT", "/user/settings", nil)
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
				req := httptest.NewRequest("DELETE", "/user/settings", nil)
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
			name:   "GET - Expired token",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id": 1,
					"exp":     time.Now().Add(-time.Hour).Unix(),
					"type":    "access",
				})
				tokenString, _ := token.SignedString(secret)

				req := httptest.NewRequest("GET", "/user/settings", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: tokenString,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
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
