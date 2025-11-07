package upload_avatar_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	upload_avatar "2025_2_a4code/internal/http-server/handlers/user/upload/upload-avatar"
	"2025_2_a4code/internal/http-server/handlers/user/upload/upload-avatar/mocks"

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

func createMultipartForm(t *testing.T, fieldName, fileName string, fileContent []byte) (*bytes.Buffer, string) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile(fieldName, fileName)
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}

	_, err = part.Write(fileContent)
	if err != nil {
		t.Fatalf("Failed to write file content: %v", err)
	}

	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	return body, writer.FormDataContentType()
}

func TestHandlerUploadAvatar_ServeHTTP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	mockProfileUsecase := mocks.NewMockProfileUsecase(ctrl)
	secret := []byte("test-secret")

	handler := upload_avatar.New(mockAvatarUsecase, mockProfileUsecase, secret)

	testAvatarContent := []byte("fake image content")
	testAvatarName := "avatar.jpg"

	tests := []struct {
		name             string
		method           string
		setupRequest     func() *http.Request
		setupMocks       func()
		expectedStatus   int
		validateResponse func(t *testing.T, body string)
	}{
		{
			name:   "POST - Success upload avatar",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, contentType := createMultipartForm(t, "avatar", testAvatarName, testAvatarContent)

				req := httptest.NewRequest("POST", "/user/upload/avatar", body)
				req.Header.Set("Content-Type", contentType)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockAvatarUsecase.EXPECT().
					UploadAvatar(gomock.Any(), "1", gomock.Any(), int64(len(testAvatarContent)), testAvatarName).
					Return("avatar123.jpg", "https://storage.example.com/avatar123.jpg", nil)

				mockProfileUsecase.EXPECT().
					InsertProfileAvatar(gomock.Any(), int64(1), "avatar123.jpg").
					Return(nil)
			},
			expectedStatus: http.StatusOK,
			validateResponse: func(t *testing.T, body string) {
				var response struct {
					Status  int    `json:"status"`
					Message string `json:"message"`
					Body    struct {
						AvatarPath string `json:"avatar_path"`
					} `json:"body"`
				}

				if err := json.Unmarshal([]byte(body), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v. Body: %s", err, body)
				}

				if response.Status != http.StatusOK {
					t.Errorf("Response status = %d, want %d", response.Status, http.StatusOK)
				}

				if response.Message != "success" {
					t.Errorf("Response message = %s, want 'success'", response.Message)
				}

				if response.Body.AvatarPath != "https://storage.example.com/avatar123.jpg" {
					t.Errorf("Expected avatar path 'https://storage.example.com/avatar123.jpg', got '%s'", response.Body.AvatarPath)
				}
			},
		},
		{
			name:   "POST - Unauthorized - no cookie",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				body, contentType := createMultipartForm(t, "avatar", testAvatarName, testAvatarContent)
				req := httptest.NewRequest("POST", "/user/upload/avatar", body)
				req.Header.Set("Content-Type", contentType)
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "POST - Unauthorized - invalid token",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				body, contentType := createMultipartForm(t, "avatar", testAvatarName, testAvatarContent)
				req := httptest.NewRequest("POST", "/user/upload/avatar", body)
				req.Header.Set("Content-Type", contentType)
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
			name:   "POST - No avatar file in form",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.WriteField("other_field", "value")
				writer.Close()

				req := httptest.NewRequest("POST", "/user/upload/avatar", body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "POST - File too large",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				largeContent := make([]byte, 6<<20) // 6MB
				body, contentType := createMultipartForm(t, "avatar", testAvatarName, largeContent)

				req := httptest.NewRequest("POST", "/user/upload/avatar", body)
				req.Header.Set("Content-Type", contentType)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "POST - Avatar upload error",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, contentType := createMultipartForm(t, "avatar", testAvatarName, testAvatarContent)

				req := httptest.NewRequest("POST", "/user/upload/avatar", body)
				req.Header.Set("Content-Type", contentType)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockAvatarUsecase.EXPECT().
					UploadAvatar(gomock.Any(), "1", gomock.Any(), int64(len(testAvatarContent)), testAvatarName).
					Return("", "", fmt.Errorf("upload failed"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "POST - Insert profile avatar error",
			method: http.MethodPost,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				body, contentType := createMultipartForm(t, "avatar", testAvatarName, testAvatarContent)

				req := httptest.NewRequest("POST", "/user/upload/avatar", body)
				req.Header.Set("Content-Type", contentType)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks: func() {
				mockAvatarUsecase.EXPECT().
					UploadAvatar(gomock.Any(), "1", gomock.Any(), int64(len(testAvatarContent)), testAvatarName).
					Return("avatar123.jpg", "https://storage.example.com/avatar123.jpg", nil)

				mockProfileUsecase.EXPECT().
					InsertProfileAvatar(gomock.Any(), int64(1), "avatar123.jpg").
					Return(fmt.Errorf("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:   "GET - Method not allowed",
			method: http.MethodGet,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("GET", "/user/upload/avatar", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "PUT - Method not allowed",
			method: http.MethodPut,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("PUT", "/user/upload/avatar", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
			},
			setupMocks:     func() {},
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:   "DELETE - Method not allowed",
			method: http.MethodDelete,
			setupRequest: func() *http.Request {
				token := createTestToken(secret, 1)
				req := httptest.NewRequest("DELETE", "/user/upload/avatar", nil)
				req.AddCookie(&http.Cookie{
					Name:  "access_token",
					Value: token,
				})
				return req
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

			if tt.validateResponse != nil && rr.Body.Len() > 0 {
				tt.validateResponse(t, rr.Body.String())
			}
		})
	}
}

func TestHandlerUploadAvatar_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	mockProfileUsecase := mocks.NewMockProfileUsecase(ctrl)
	secret := []byte("test-secret")

	handler := upload_avatar.New(mockAvatarUsecase, mockProfileUsecase, secret)

	t.Run("POST - Empty file", func(t *testing.T) {
		token := createTestToken(secret, 1)
		body, contentType := createMultipartForm(t, "avatar", "empty.jpg", []byte{})

		req := httptest.NewRequest("POST", "/user/upload/avatar", body)
		req.Header.Set("Content-Type", contentType)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		mockAvatarUsecase.EXPECT().
			UploadAvatar(gomock.Any(), "1", gomock.Any(), int64(0), "empty.jpg").
			Return("empty.jpg", "https://storage.example.com/empty.jpg", nil)

		mockProfileUsecase.EXPECT().
			InsertProfileAvatar(gomock.Any(), int64(1), "empty.jpg").
			Return(nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status %d for empty file, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("POST - Wrong field name", func(t *testing.T) {
		token := createTestToken(secret, 1)
		body, contentType := createMultipartForm(t, "wrong_field", "avatar.jpg", []byte("content"))

		req := httptest.NewRequest("POST", "/user/upload/avatar", body)
		req.Header.Set("Content-Type", contentType)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d for wrong field name, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("POST - Invalid multipart form", func(t *testing.T) {
		token := createTestToken(secret, 1)
		body := bytes.NewBufferString("invalid multipart data")

		req := httptest.NewRequest("POST", "/user/upload/avatar", body)
		req.Header.Set("Content-Type", "multipart/form-data")
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status %d for invalid multipart form, got %d", http.StatusInternalServerError, rr.Code)
		}
	})
}

func TestHandlerUploadAvatar_JSONEncoding(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockAvatarUsecase := mocks.NewMockAvatarUsecase(ctrl)
	mockProfileUsecase := mocks.NewMockProfileUsecase(ctrl)
	secret := []byte("test-secret")

	handler := upload_avatar.New(mockAvatarUsecase, mockProfileUsecase, secret)

	t.Run("POST - Valid JSON response", func(t *testing.T) {
		token := createTestToken(secret, 1)
		body, contentType := createMultipartForm(t, "avatar", "test.jpg", []byte("content"))

		req := httptest.NewRequest("POST", "/user/upload/avatar", body)
		req.Header.Set("Content-Type", contentType)
		req.AddCookie(&http.Cookie{
			Name:  "access_token",
			Value: token,
		})

		mockAvatarUsecase.EXPECT().
			UploadAvatar(gomock.Any(), "1", gomock.Any(), gomock.Any(), "test.jpg").
			Return("test.jpg", "https://example.com/test.jpg", nil)

		mockProfileUsecase.EXPECT().
			InsertProfileAvatar(gomock.Any(), int64(1), "test.jpg").
			Return(nil)

		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Errorf("Response is not valid JSON: %v", err)
		}

		if status, ok := response["status"].(float64); !ok || int(status) != http.StatusOK {
			t.Errorf("Invalid status in response: %v", response["status"])
		}

		if message, ok := response["message"].(string); !ok || message != "success" {
			t.Errorf("Invalid message in response: %v", response["message"])
		}

		bodyMap, ok := response["body"].(map[string]interface{})
		if !ok {
			t.Errorf("Invalid body in response: %v", response["body"])
		}

		if avatarPath, ok := bodyMap["avatar_path"].(string); !ok || avatarPath != "https://example.com/test.jpg" {
			t.Errorf("Invalid avatar_path in response: %v", bodyMap["avatar_path"])
		}
	})
}
