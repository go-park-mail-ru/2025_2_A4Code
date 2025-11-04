package signup_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"2025_2_a4code/internal/http-server/handlers/auth/signup"
	"2025_2_a4code/internal/http-server/handlers/auth/signup/mocks"
	"2025_2_a4code/internal/usecase/profile"

	"github.com/golang/mock/gomock"
)

func TestHandlerSignup_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileUsecase := mocks.NewMockProfileUsecase(ctrl)
	secret := []byte("test-secret")

	handler := signup.New(mockProfileUsecase, secret)

	tests := []struct {
		name            string
		requestBody     interface{}
		setupMocks      func()
		expectedStatus  int
		expectedMessage string
		checkCookies    bool
	}{
		{
			name: "Success signup",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					Signup(gomock.Any(), profile.SignupRequest{
						Name:     "John Doe",
						Username: "johndoe",
						Birthday: "15.05.1990",
						Gender:   "male",
						Password: "password123",
					}).
					Return(int64(1), nil)
			},
			expectedStatus:  http.StatusOK,
			expectedMessage: "success",
			checkCookies:    true,
		},
		{
			name: "Success signup with female gender",
			requestBody: signup.Request{
				Name:     "Jane Smith",
				Username: "janesmith",
				Birthday: "20.08.1995",
				Gender:   "female",
				Password: "password456",
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					Signup(gomock.Any(), profile.SignupRequest{
						Name:     "Jane Smith",
						Username: "janesmith",
						Birthday: "20.08.1995",
						Gender:   "female",
						Password: "password456",
					}).
					Return(int64(2), nil)
			},
			expectedStatus:  http.StatusOK,
			expectedMessage: "success",
			checkCookies:    true,
		},
		{
			name: "Signup failure - user already exists",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					Signup(gomock.Any(), gomock.Any()).
					Return(int64(0), profile.ErrUserAlreadyExists)
			},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "user with this username already exists",
			checkCookies:    false,
		},
		{
			name: "Signup failure - unexpected error",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					Signup(gomock.Any(), gomock.Any()).
					Return(int64(0), errors.New("database error"))
			},
			expectedStatus:  http.StatusInternalServerError,
			expectedMessage: "something went wrong",
			checkCookies:    false,
		},
		{
			name:            "Invalid HTTP method",
			requestBody:     signup.Request{},
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
			name: "Empty name",
			requestBody: signup.Request{
				Name:     "",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "all fields are required",
			checkCookies:    false,
		},
		{
			name: "Empty username",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "all fields are required",
			checkCookies:    false,
		},
		{
			name: "Empty birthday",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "all fields are required",
			checkCookies:    false,
		},
		{
			name: "Empty gender",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "all fields are required",
			checkCookies:    false,
		},
		{
			name: "Empty password",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "all fields are required",
			checkCookies:    false,
		},
		{
			name: "Name too short",
			requestBody: signup.Request{
				Name:     "J",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "name must be between 2 and 100 characters",
			checkCookies:    false,
		},
		{
			name: "Name with invalid characters",
			requestBody: signup.Request{
				Name:     "John123",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "name can only contain letters, spaces and hyphens",
			checkCookies:    false,
		},
		{
			name: "Username too short",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "jo",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "username must be between 3 and 50 characters",
			checkCookies:    false,
		},
		{
			name: "Username with invalid characters",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "john@doe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "username can only contain letters, numbers and underscores",
			checkCookies:    false,
		},
		{
			name: "Invalid birthday format - wrong length",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.19900",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "birthday must be in DD.MM.YYYY format",
			checkCookies:    false,
		},
		{
			name: "Invalid birthday format - missing dots",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15051990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "birthday must be in DD.MM.YYYY format",
			checkCookies:    false,
		},
		{
			name: "Invalid birthday - invalid month",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.13.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "birthday month must be between 01 and 12",
			checkCookies:    false,
		},
		{
			name: "Invalid birthday - invalid day for month",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "31.04.1990",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "birthday day is out of range for the month",
			checkCookies:    false,
		},
		{
			name: "Invalid birthday - future date",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.2050",
				Gender:   "male",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "birthday must not be in the future",
			checkCookies:    false,
		},
		{
			name: "Invalid gender",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "other",
				Password: "password123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "gender must be male or female",
			checkCookies:    false,
		},
		{
			name: "Password too short",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "password must be at least 6 characters",
			checkCookies:    false,
		},
		{
			name: "Password without letters",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "123456789",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "password must contain both letters and numbers",
			checkCookies:    false,
		},
		{
			name: "Password without numbers",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "abcdefgh",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "password must contain both letters and numbers",
			checkCookies:    false,
		},
		{
			name: "Password with spaces",
			requestBody: signup.Request{
				Name:     "John Doe",
				Username: "johndoe",
				Birthday: "15.05.1990",
				Gender:   "male",
				Password: "pass word123",
			},
			setupMocks:      func() {},
			expectedStatus:  http.StatusBadRequest,
			expectedMessage: "password must not contain spaces",
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
				req = httptest.NewRequest("GET", "/auth/signup", bytes.NewReader(bodyBytes))
			} else {
				req = httptest.NewRequest("POST", "/auth/signup", bytes.NewReader(bodyBytes))
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if rr.Body.Len() > 0 && tt.expectedStatus != http.StatusInternalServerError {
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
					}
					if cookie.Name == "refresh_token" {
						refreshTokenFound = true
						if cookie.Value == "" {
							t.Error("Refresh token cookie value is empty")
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
