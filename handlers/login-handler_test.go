package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLoginHandler(t *testing.T) {
	h := New()
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
			body:           map[string]string{"login": "test", "password": "test"},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid JSON",
			method:         "POST",
			body:           nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing login field",
			method:         "POST",
			body:           map[string]string{"password": "test"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing password field",
			method:         "POST",
			body:           map[string]string{"login": "test"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid credentials",
			method:         "POST",
			body:           map[string]string{"login": "nonexistent", "password": "wrong"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			if tt.body != nil {
				bodyBytes, _ = json.Marshal(tt.body)
			}

			req := httptest.NewRequest(tt.method, "/login", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			h.LoginHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestLoginHandler_Success(t *testing.T) {
	h := New()
	defer h.Reset()

	users := h.GetUsers()
	testUser := map[string]string{
		"login":    "testuser",
		"password": "testpass",
	}
	h.SetUsers(append(users, testUser))

	body := map[string]string{
		"login":    "testuser",
		"password": "testpass",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest("POST", "/login", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()

	h.LoginHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	cookies := w.Result().Cookies()
	foundSessionCookie := false
	for _, cookie := range cookies {
		if cookie.Name == "session_id" && cookie.Value != "" {
			foundSessionCookie = true
			break
		}
	}
	if !foundSessionCookie {
		t.Error("Session cookie was not set during successful login")
	}
}
