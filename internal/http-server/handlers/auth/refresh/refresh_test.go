package refresh_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"2025_2_a4code/internal/http-server/handlers/auth/refresh"

	"github.com/golang-jwt/jwt/v5"
)

func TestHandlerRefresh_ServeHTTP(t *testing.T) {
	secret := []byte("test-secret")
	handler := refresh.New(secret)

	tests := []struct {
		name              string
		method            string
		setupRequest      func() *http.Request
		expectedStatus    int
		expectedMessage   string
		checkAccessCookie bool
	}{
		{
			name:   "Success refresh",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id": int64(1),
					"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
					"type":    "refresh",
				})
				refreshTokenString, _ := refreshToken.SignedString(secret)

				bodyBytes, _ := json.Marshal(struct{}{})
				req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				req.AddCookie(&http.Cookie{
					Name:  "refresh_token",
					Value: refreshTokenString,
				})
				return req
			},
			expectedStatus:    http.StatusOK,
			expectedMessage:   "success",
			checkAccessCookie: true,
		},
		{
			name:   "Invalid HTTP method",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				bodyBytes, _ := json.Marshal(struct{}{})
				req := httptest.NewRequest("GET", "/auth/refresh", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectedStatus:    http.StatusMethodNotAllowed,
			expectedMessage:   "method not allowed",
			checkAccessCookie: false,
		},
		{
			name:   "Missing refresh token",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				bodyBytes, _ := json.Marshal(struct{}{})
				req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			expectedStatus:    http.StatusUnauthorized,
			expectedMessage:   "unauthorized",
			checkAccessCookie: false,
		},
		{
			name:   "Invalid refresh token",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				bodyBytes, _ := json.Marshal(struct{}{})
				req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				req.AddCookie(&http.Cookie{
					Name:  "refresh_token",
					Value: "invalid_token",
				})
				return req
			},
			expectedStatus:    http.StatusUnauthorized,
			expectedMessage:   "unauthorized",
			checkAccessCookie: false,
		},
		{
			name:   "Expired refresh token",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"user_id": int64(1),
					"exp":     time.Now().Add(-1 * time.Hour).Unix(),
					"type":    "refresh",
				})
				refreshTokenString, _ := refreshToken.SignedString(secret)

				bodyBytes, _ := json.Marshal(struct{}{})
				req := httptest.NewRequest("POST", "/auth/refresh", bytes.NewReader(bodyBytes))
				req.Header.Set("Content-Type", "application/json")

				req.AddCookie(&http.Cookie{
					Name:  "refresh_token",
					Value: refreshTokenString,
				})
				return req
			},
			expectedStatus:    http.StatusUnauthorized,
			expectedMessage:   "unauthorized",
			checkAccessCookie: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d. Body: %s", tt.expectedStatus, rr.Code, rr.Body.String())
			}

			if rr.Body.Len() > 0 {
				var response struct {
					Status  int    `json:"status"`
					Message string `json:"message"`
				}

				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, rr.Body.String())
				}

				if response.Status != tt.expectedStatus {
					t.Errorf("Response status = %d, want %d", response.Status, tt.expectedStatus)
				}

				if response.Message != tt.expectedMessage {
					t.Errorf("Response message = %s, want %s", response.Message, tt.expectedMessage)
				}
			}

			cookies := rr.Result().Cookies()

			if tt.checkAccessCookie {
				var accessTokenFound bool
				for _, cookie := range cookies {
					if cookie.Name == "access_token" {
						accessTokenFound = true
						if cookie.Value == "" {
							t.Error("New access token cookie value should not be empty")
						}
						if !cookie.HttpOnly {
							t.Error("Access token cookie should be HttpOnly")
						}
						if !cookie.Secure {
							t.Error("Access token cookie should be Secure")
						}
						if cookie.MaxAge != 15*60 {
							t.Errorf("Access token cookie MaxAge should be 900, got %d", cookie.MaxAge)
						}
						break
					}
				}

				if !accessTokenFound {
					t.Error("New access token cookie not found")
				}
			} else if tt.expectedStatus == http.StatusUnauthorized {
				var refreshTokenCleared bool
				for _, cookie := range cookies {
					if cookie.Name == "refresh_token" {
						if cookie.Value == "" && cookie.MaxAge == -1 {
							refreshTokenCleared = true
						}
						break
					}
				}

				if !refreshTokenCleared {
					t.Error("Refresh token should be cleared on unauthorized")
				}
			}
		})
	}
}
