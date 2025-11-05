package login_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"2025_2_a4code/internal/http-server/handlers/auth/login"
	"2025_2_a4code/internal/http-server/handlers/auth/login/mocks"
	"2025_2_a4code/internal/usecase/profile"

	"github.com/golang/mock/gomock"
)

func TestHandlerLogin_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileUsecase := mocks.NewMockProfileUsecase(ctrl)
	secret := []byte("test-secret")

	handler := login.New(mockProfileUsecase, secret)

	tests := []struct {
		name            string
		requestBody     interface{}
		setupMocks      func()
		expectedStatus  int
		expectedMessage string
		checkCookies    bool
	}{
		{
			name: "Success login with username",
			requestBody: login.Request{
				Login:    "testuser",
				Password: "password123",
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					Login(gomock.Any(), profile.LoginRequest{
						Username: "testuser",
						Password: "password123",
					}).
					Return(int64(1), nil)
			},
			expectedStatus:  http.StatusOK,
			expectedMessage: "success",
			checkCookies:    true,
		},
		{
			name: "Success login with email",
			requestBody: login.Request{
				Login:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					Login(gomock.Any(), profile.LoginRequest{
						Username: "test",
						Password: "password123",
					}).
					Return(int64(1), nil)
			},
			expectedStatus:  http.StatusOK,
			expectedMessage: "success",
			checkCookies:    true,
		},
		{
			name: "Login failure - email not found",
			requestBody: login.Request{
				Login:    "test@example.com",
				Password: "password123",
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					Login(gomock.Any(), profile.LoginRequest{
						Username: "test",
						Password: "password123",
					}).
					Return(int64(0), errors.New("user not found"))
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid login or password",
			checkCookies:    false,
		},
		{
			name:            "Invalid HTTP method",
			requestBody:     login.Request{},
			setupMocks:      func() {},
			expectedStatus:  http.StatusMethodNotAllowed,
			expectedMessage: "method not allowed",
			checkCookies:    false,
		},
		{
			name:            "Invalid JSON",
			requestBody:     "invalid json {",
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid request format",
			checkCookies:    false,
		},
		{
			name: "Empty login",
			requestBody: login.Request{
				Login:    "",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "all fields are required",
			checkCookies:    false,
		},
		{
			name: "Empty password",
			requestBody: login.Request{
				Login:    "testuser",
				Password: "",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "all fields are required",
			checkCookies:    false,
		},
		{
			name: "Login failure - wrong credentials",
			requestBody: login.Request{
				Login:    "testuser",
				Password: "wrongpassword",
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					Login(gomock.Any(), profile.LoginRequest{
						Username: "testuser",
						Password: "wrongpassword",
					}).
					Return(int64(0), errors.New("invalid credentials"))
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "invalid login or password",
			checkCookies:    false,
		},
		{
			name: "Username too short",
			requestBody: login.Request{
				Login:    "ab",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "username must be between 3 and 50 characters",
			checkCookies:    false,
		},
		{
			name: "Password too short",
			requestBody: login.Request{
				Login:    "testuser",
				Password: "123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "password must be at least 6 characters",
			checkCookies:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			var bodyBytes []byte
			var err error

			switch body := tt.requestBody.(type) {
			case string:
				bodyBytes = []byte(body)
			default:
				bodyBytes, err = json.Marshal(body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			var req *http.Request
			if tt.name == "Invalid HTTP method" {
				req = httptest.NewRequest("GET", "/auth/login", bytes.NewReader(bodyBytes))
			} else {
				req = httptest.NewRequest("POST", "/auth/login", bytes.NewReader(bodyBytes))
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if rr.Body.Len() > 0 {
				var response struct {
					Status  int         `json:"status"`
					Message string      `json:"message"`
					Body    interface{} `json:"body"`
				}

				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					if tt.name != "Invalid JSON" {
						t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, rr.Body.String())
					}
				} else {
					if response.Status != tt.expectedStatus {
						t.Errorf("Response status = %d, want %d", response.Status, tt.expectedStatus)
					}

					if response.Message != tt.expectedMessage {
						t.Errorf("Response message = %s, want %s", response.Message, tt.expectedMessage)
					}

					if response.Body != nil {
						bodyBytes, _ := json.Marshal(response.Body)
						if string(bodyBytes) != "{}" {
							t.Errorf("Response body = %s, want {}", string(bodyBytes))
						}
					}
				}
			}

			cookies := rr.Result().Cookies()
			if tt.checkCookies {
				if len(cookies) != 2 {
					t.Errorf("Expected 2 cookies, got %d", len(cookies))
				}

				var accessTokenFound, refreshTokenFound bool
				for _, cookie := range cookies {
					if cookie.Name == "access_token" {
						accessTokenFound = true
						if cookie.Value == "" {
							t.Error("Access token cookie value is empty")
						}
						if !cookie.HttpOnly {
							t.Error("Access token cookie should be HttpOnly")
						}
						if !cookie.Secure {
							t.Error("Access token cookie should be Secure")
						}
					}
					if cookie.Name == "refresh_token" {
						refreshTokenFound = true
						if cookie.Value == "" {
							t.Error("Refresh token cookie value is empty")
						}
						if !cookie.HttpOnly {
							t.Error("Refresh token cookie should be HttpOnly")
						}
						if !cookie.Secure {
							t.Error("Refresh token cookie should be Secure")
						}
					}
				}

				if !accessTokenFound {
					t.Error("Access token cookie not found")
				}
				if !refreshTokenFound {
					t.Error("Refresh token cookie not found")
				}
			} else {
				if len(cookies) > 0 {
					t.Errorf("Expected no cookies for error response, but got %d cookies", len(cookies))
				}
			}
		})
	}
}
