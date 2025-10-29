package logout

import (
	handlers2 "2025_2_a4code/internal/http-server/handlers"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLogoutHandler(t *testing.T) {
	h := handlers2.New()

	tests := []struct {
		name           string
		method         string
		expectedStatus int
	}{
		{
			name:           "Wrong method",
			method:         "GET",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Successful logout",
			method:         "POST",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/logout", nil)
			w := httptest.NewRecorder()

			h.LogoutHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.method == "POST" && w.Code == http.StatusOK {
				cookies := w.Result().Cookies()
				foundSessionCookie := false
				for _, cookie := range cookies {
					if cookie.Name == "session_id" && cookie.Value == "" {
						foundSessionCookie = true
						break
					}
				}
				if !foundSessionCookie {
					t.Error("Session cookie was not cleared during logout")
				}
			}
		})
	}
}
