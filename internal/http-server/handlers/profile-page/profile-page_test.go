package profile_page

import (
	handlers2 "2025_2_a4code/internal/http-server/handlers"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMeHandler(t *testing.T) {
	h := handlers2.New()
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
			req := httptest.NewRequest(tt.method, "/profile-page", nil)

			if tt.setupAuth {
				users := h.GetUsers()
				testUser := map[string]string{
					"login":       "authtest",
					"password":    "authpass",
					"username":    "Auth User",
					"dateofbirth": "1990-01-01",
					"gender":      "other",
				}
				h.SetUsers(append(users, testUser))

				token, _ := h.CreateToken("authtest")
				req.AddCookie(&http.Cookie{
					Name:  "session_id",
					Value: token,
				})
			}

			w := httptest.NewRecorder()

			h.MeHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}
