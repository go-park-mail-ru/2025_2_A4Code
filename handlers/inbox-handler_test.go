package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInboxHandler(t *testing.T) {
	h := New()
	defer h.Reset()

	tests := []struct {
		name           string
		method         string
		setupAuth      bool
		expectedStatus int
	}{
		{
			name:           "Wrong method",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Without authentication",
			method:         "GET",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "With valid authentication",
			method:         "GET",
			setupAuth:      true,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/inbox", nil)

			if tt.setupAuth {
				users := h.GetUsers()
				testUser := map[string]string{
					"login": "inboxuser",
				}
				h.SetUsers(append(users, testUser))

				token, _ := h.CreateToken("inboxuser")
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: token,
				})
			}

			w := httptest.NewRecorder()

			h.InboxHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
