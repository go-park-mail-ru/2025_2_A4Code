package logout_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"2025_2_a4code/internal/http-server/handlers/auth/logout"
)

func TestHandlerLogout_ServeHTTP(t *testing.T) {
	handler := logout.New()

	tests := []struct {
		name            string
		method          string
		expectedStatus  int
		expectedMessage string
		checkCookies    bool
	}{
		{
			name:            "Success logout",
			method:          http.MethodPost,
			expectedStatus:  http.StatusOK,
			expectedMessage: "success",
			checkCookies:    true,
		},
		{
			name:            "Invalid HTTP method",
			method:          http.MethodGet,
			expectedStatus:  http.StatusMethodNotAllowed,
			expectedMessage: "method not allowed",
			checkCookies:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(struct{}{})

			req := httptest.NewRequest(tt.method, "/auth/logout", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

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
			if tt.checkCookies {
				if len(cookies) != 2 {
					t.Errorf("Expected 2 cookies, got %d", len(cookies))
				}

				var accessTokenFound, refreshTokenFound bool
				for _, cookie := range cookies {
					if cookie.Name == "access_token" {
						accessTokenFound = true
						if cookie.Value != "" {
							t.Error("Access token cookie value should be empty")
						}
						if cookie.MaxAge != -1 {
							t.Errorf("Access token cookie MaxAge should be -1, got %d", cookie.MaxAge)
						}
					}
					if cookie.Name == "refresh_token" {
						refreshTokenFound = true
						if cookie.Value != "" {
							t.Error("Refresh token cookie value should be empty")
						}
						if cookie.MaxAge != -1 {
							t.Errorf("Refresh token cookie MaxAge should be -1, got %d", cookie.MaxAge)
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
