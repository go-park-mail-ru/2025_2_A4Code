package profile_page_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"2025_2_a4code/internal/domain"
	profile_page "2025_2_a4code/internal/http-server/handlers/user/profile-page"
	"2025_2_a4code/internal/http-server/handlers/user/profile-page/mocks"
	"2025_2_a4code/internal/usecase/profile"

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

func TestHandlerProfile_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockProfileUsecase := mocks.NewMockProfileUsecase(ctrl)
	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	secret := []byte("test-secret")

	handler := profile_page.New(mockProfileUsecase, mockAvatarUsecase, secret)

	tests := []struct {
		name           string
		method         string
		setupRequest   func() *http.Request
		setupMocks     func()
		expectedStatus int
	}{
		{
			name:   "GET - Success get profile",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/user/profile", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					FindInfoByID(gomock.Any(), int64(1)).
					Return(domain.ProfileInfo{
						ID:         1,
						Username:   "testuser",
						CreatedAt:  time.Now(),
						Name:       "John",
						Surname:    "Doe",
						Patronymic: "Smith",
						Gender:     "male",
						Birthday:   "1990-05-15",
						AvatarPath: "",
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "GET - Success with avatar",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/user/profile", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					FindInfoByID(gomock.Any(), int64(1)).
					Return(domain.ProfileInfo{
						ID:         1,
						Username:   "testuser",
						CreatedAt:  time.Now(),
						Name:       "John",
						Surname:    "Doe",
						Patronymic: "Smith",
						Gender:     "male",
						Birthday:   "1990-05-15",
						AvatarPath: "avatar123.jpg",
					}, nil)

				presignedURL, _ := url.Parse("https://storage.example.com/avatar123.jpg?signature=abc")
				mockAvatarUsecase.EXPECT().
					GetAvatarPresignedURL(gomock.Any(), "avatar123.jpg", 15*time.Minute).
					Return(presignedURL, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "GET - Unauthorized",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/user/profile", nil)
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "GET - Invalid token",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/user/profile", nil)
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
			name:   "PUT - Success update profile",
			method: http.MethodPut,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				updateReq := profile_page.UpdateProfileRequest{
					Name:       "John",
					Surname:    "Doe",
					Patronymic: "Smith",
					Gender:     "male",
					Birthday:   "15.05.1990",
				}
				body, _ := json.Marshal(updateReq)
				req := httptest.NewRequest("PUT", "/user/profile", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					UpdateProfileInfo(gomock.Any(), int64(1), profile.UpdateProfileRequest{
						FirstName:  "John",
						LastName:   "Doe",
						MiddleName: "Smith",
						Gender:     "male",
						Birthday:   "15.05.1990",
					}).
					Return(nil)

				mockProfileUsecase.EXPECT().
					FindInfoByID(gomock.Any(), int64(1)).
					Return(domain.ProfileInfo{
						ID:         1,
						Username:   "testuser",
						CreatedAt:  time.Now(),
						Name:       "John",
						Surname:    "Doe",
						Patronymic: "Smith",
						Gender:     "male",
						Birthday:   "1990-05-15",
						AvatarPath: "",
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "PUT - Unauthorized",
			method: http.MethodPut,
			setupRequest: func() *http.Request {
				updateReq := profile_page.UpdateProfileRequest{
					Name:       "John",
					Surname:    "Doe",
					Patronymic: "Smith",
					Gender:     "male",
					Birthday:   "15.05.1990",
				}
				body, _ := json.Marshal(updateReq)
				req := httptest.NewRequest("PUT", "/user/profile", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "PUT - Invalid JSON",
			method: http.MethodPut,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("PUT", "/user/profile", bytes.NewReader([]byte("invalid json")))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "PUT - Update error",
			method: http.MethodPut,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				updateReq := profile_page.UpdateProfileRequest{
					Name:       "John",
					Surname:    "Doe",
					Patronymic: "Smith",
					Gender:     "male",
					Birthday:   "15.05.1990",
				}
				body, _ := json.Marshal(updateReq)
				req := httptest.NewRequest("PUT", "/user/profile", bytes.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockProfileUsecase.EXPECT().
					UpdateProfileInfo(gomock.Any(), int64(1), gomock.Any()).
					Return(errors.New("user not found"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "Method not allowed - POST",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				return httptest.NewRequest("POST", "/user/profile", nil)
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "Method not allowed - DELETE",
			method: http.MethodDelete,
			setupRequest: func() *http.Request {
				return httptest.NewRequest("DELETE", "/user/profile", nil)
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
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

			if rr.Code == http.StatusOK && rr.Body.Len() > 0 {
				var response struct {
					Status  int                  `json:"status"`
					Message string               `json:"message"`
					Body    profile_page.Profile `json:"body"`
				}

				if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, rr.Body.String())
				}

				if response.Status != http.StatusOK {
					t.Errorf("Response status = %d, want %d", response.Status, http.StatusOK)
				}

				if response.Message != "success" {
					t.Errorf("Response message = %s, want 'success'", response.Message)
				}

				if response.Body.Username == "" {
					t.Error("Response body should contain username")
				}
			}
		})
	}
}
