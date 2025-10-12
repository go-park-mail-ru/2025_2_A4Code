package signup

import (
	handlers2 "2025_2_a4code/internal/http-server/handlers"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignupHandler(t *testing.T) {
	h := handlers2.New()
	defer h.Reset()

	tests := []struct {
		name           string
		method         string
		body           map[string]string
		expectedStatus int
	}{
		{
			name:           "Wrong method",
			method:         "GET",
			body:           map[string]string{"login": "testuser", "password": "testpass", "username": "Test", "dateofbirth": "2000-01-01", "gender": "male"},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid JSON",
			method:         "POST",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing required fields",
			method:         "POST",
			body:           map[string]string{"login": "test"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Successful registration",
			method:         "POST",
			body:           map[string]string{"login": "newuser", "password": "newpass", "username": "New User", "dateofbirth": "2000-01-01", "gender": "female"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(tt.method, "/signup", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			h.SignupHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK && tt.method == "POST" {
				cookies := w.Result().Cookies()
				foundSessionCookie := false
				for _, cookie := range cookies {
					if cookie.Name == "session_id" && cookie.Value != "" {
						foundSessionCookie = true
						break
					}
				}
				if !foundSessionCookie {
					t.Error("Session cookie was not set during successful registration")
				}
			}
		})
	}
}

func TestSignupHandler_DuplicateUser(t *testing.T) {
	h := handlers2.New()
	defer h.Reset()

	users := h.GetUsers()
	existingUser := map[string]string{
		"login":    "duplicate",
		"password": "pass",
	}
	h.SetUsers(append(users, existingUser))

	body := map[string]string{
		"login":    "duplicate",
		"password": "newpass",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/signup", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.SignupHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}
