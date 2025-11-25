package auth_service

import (
	"2025_2_a4code/auth-service/pkg/authproto"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/profile"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MockProfileUsecase struct {
	mock.Mock
}

func (m *MockProfileUsecase) Login(ctx context.Context, req profile.LoginRequest) (int64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProfileUsecase) Signup(ctx context.Context, req profile.SignupRequest) (int64, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(int64), args.Error(1)
}

func setupTestServer() (*Server, *MockProfileUsecase) {
	mockProfileUsecase := &MockProfileUsecase{}
	jwtSecret := []byte("test-secret-key-very-long-for-testing")
	server := New(mockProfileUsecase, jwtSecret)
	return server, mockProfileUsecase
}

func createTestContext() context.Context {
	return context.Background()
}

func TestServer_Login(t *testing.T) {
	server, mockProfile := setupTestServer()

	tests := []struct {
		name           string
		request        *authproto.LoginRequest
		mockSetup      func()
		expectedError  bool
		expectedCode   codes.Code
		validateTokens bool
	}{
		{
			name: "Success",
			request: &authproto.LoginRequest{
				Login:    "testuser",
				Password: "password123",
			},
			mockSetup: func() {
				mockProfile.On("Login", mock.Anything, profile.LoginRequest{
					Username: "testuser",
					Password: "password123",
				}).Return(int64(1), nil)
			},
			expectedError:  false,
			validateTokens: true,
		},
		{
			name: "EmptyLogin",
			request: &authproto.LoginRequest{
				Login:    "",
				Password: "password123",
			},
			mockSetup:     func() {},
			expectedError: true,
			expectedCode:  codes.InvalidArgument,
		},
		{
			name: "EmptyPassword",
			request: &authproto.LoginRequest{
				Login:    "testuser",
				Password: "",
			},
			mockSetup:     func() {},
			expectedError: true,
			expectedCode:  codes.InvalidArgument,
		},
		{
			name: "InvalidCredentials",
			request: &authproto.LoginRequest{
				Login:    "testuser",
				Password: "wrongpassword",
			},
			mockSetup: func() {
				mockProfile.On("Login", mock.Anything, profile.LoginRequest{
					Username: "testuser",
					Password: "wrongpassword",
				}).Return(int64(0), errors.New("invalid credentials"))
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := server.Login(createTestContext(), tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedCode != codes.OK {
					grpcStatus, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedCode, grpcStatus.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)

				if tt.validateTokens {
					userID, err := session.GetProfileIDFromTokenString(resp.AccessToken, server.JWTSecret, "access")
					assert.NoError(t, err)
					assert.Equal(t, int64(1), userID)

					userID, err = session.GetProfileIDFromTokenString(resp.RefreshToken, server.JWTSecret, "refresh")
					assert.NoError(t, err)
					assert.Equal(t, int64(1), userID)
				}
			}

			mockProfile.AssertExpectations(t)
		})
	}
}

func TestServer_Signup(t *testing.T) {
	server, mockProfile := setupTestServer()

	tests := []struct {
		name           string
		request        *authproto.SignupRequest
		mockSetup      func()
		expectedError  bool
		expectedCode   codes.Code
		validateTokens bool
	}{
		{
			name: "Success",
			request: &authproto.SignupRequest{
				Name:     "Test User",
				Username: "testuser",
				Birthday: "1990-01-01",
				Gender:   "male",
				Password: "password123",
			},
			mockSetup: func() {
				mockProfile.On("Signup", mock.Anything, profile.SignupRequest{
					Name:     "Test User",
					Username: "testuser",
					Birthday: "1990-01-01",
					Gender:   "male",
					Password: "password123",
				}).Return(int64(1), nil)
			},
			expectedError:  false,
			validateTokens: true,
		},
		{
			name: "EmptyUsername",
			request: &authproto.SignupRequest{
				Name:     "Test User",
				Username: "",
				Birthday: "1990-01-01",
				Gender:   "male",
				Password: "password123",
			},
			mockSetup:     func() {},
			expectedError: true,
			expectedCode:  codes.InvalidArgument,
		},
		{
			name: "EmptyPassword",
			request: &authproto.SignupRequest{
				Name:     "Test User",
				Username: "testuser",
				Birthday: "1990-01-01",
				Gender:   "male",
				Password: "",
			},
			mockSetup:     func() {},
			expectedError: true,
			expectedCode:  codes.InvalidArgument,
		},
		{
			name: "UserAlreadyExists",
			request: &authproto.SignupRequest{
				Name:     "Test User",
				Username: "existinguser",
				Birthday: "1990-01-01",
				Gender:   "male",
				Password: "password123",
			},
			mockSetup: func() {
				mockProfile.On("Signup", mock.Anything, profile.SignupRequest{
					Name:     "Test User",
					Username: "existinguser",
					Birthday: "1990-01-01",
					Gender:   "male",
					Password: "password123",
				}).Return(int64(0), profile.ErrUserAlreadyExists)
			},
			expectedError: true,
			expectedCode:  codes.AlreadyExists,
		},
		{
			name: "InternalError",
			request: &authproto.SignupRequest{
				Name:     "Test User",
				Username: "testuser",
				Birthday: "1990-01-01",
				Gender:   "male",
				Password: "password123",
			},
			mockSetup: func() {
				mockProfile.On("Signup", mock.Anything, profile.SignupRequest{
					Name:     "Test User",
					Username: "testuser",
					Birthday: "1990-01-01",
					Gender:   "male",
					Password: "password123",
				}).Return(int64(0), errors.New("internal error"))
			},
			expectedError: true,
			expectedCode:  codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := server.Signup(createTestContext(), tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedCode != codes.OK {
					grpcStatus, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedCode, grpcStatus.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)
				assert.NotEmpty(t, resp.RefreshToken)

				if tt.validateTokens {
					// Проверяем, что токены валидны
					userID, err := session.GetProfileIDFromTokenString(resp.AccessToken, server.JWTSecret, "access")
					assert.NoError(t, err)
					assert.Equal(t, int64(1), userID)

					userID, err = session.GetProfileIDFromTokenString(resp.RefreshToken, server.JWTSecret, "refresh")
					assert.NoError(t, err)
					assert.Equal(t, int64(1), userID)
				}
			}

			mockProfile.AssertExpectations(t)
		})
	}
}

func TestServer_Refresh(t *testing.T) {
	server, _ := setupTestServer()

	validRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(1),
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type":    "refresh",
	})
	validRefreshTokenString, _ := validRefreshToken.SignedString(server.JWTSecret)

	expiredRefreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(1),
		"exp":     time.Now().Add(-1 * time.Hour).Unix(),
		"type":    "refresh",
	})
	expiredRefreshTokenString, _ := expiredRefreshToken.SignedString(server.JWTSecret)

	wrongSecret := []byte("wrong-secret")
	invalidSignatureToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": int64(1),
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type":    "refresh",
	})
	invalidSignatureTokenString, _ := invalidSignatureToken.SignedString(wrongSecret)

	tests := []struct {
		name          string
		request       *authproto.RefreshRequest
		expectedError bool
		expectedCode  codes.Code
	}{
		{
			name: "Success",
			request: &authproto.RefreshRequest{
				RefreshToken: validRefreshTokenString,
			},
			expectedError: false,
		},
		{
			name: "EmptyToken",
			request: &authproto.RefreshRequest{
				RefreshToken: "",
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "ExpiredToken",
			request: &authproto.RefreshRequest{
				RefreshToken: expiredRefreshTokenString,
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "InvalidSignature",
			request: &authproto.RefreshRequest{
				RefreshToken: invalidSignatureTokenString,
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "WrongTokenType",
			request: &authproto.RefreshRequest{
				RefreshToken: func() string {
					token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
						"user_id": int64(1),
						"exp":     time.Now().Add(15 * time.Minute).Unix(),
						"type":    "access", // неправильный тип
					})
					tokenString, _ := token.SignedString(server.JWTSecret)
					return tokenString
				}(),
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.Refresh(createTestContext(), tt.request)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedCode != codes.OK {
					grpcStatus, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedCode, grpcStatus.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NotEmpty(t, resp.AccessToken)

				userID, err := session.GetProfileIDFromTokenString(resp.AccessToken, server.JWTSecret, "access")
				assert.NoError(t, err)
				assert.Equal(t, int64(1), userID)
			}
		})
	}
}

func TestServer_Logout(t *testing.T) {
	server, _ := setupTestServer()

	t.Run("Success", func(t *testing.T) {
		resp, err := server.Logout(createTestContext(), &authproto.LogoutRequest{})

		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})
}

func TestServer_generateAccessToken(t *testing.T) {
	server, _ := setupTestServer()

	t.Run("Success", func(t *testing.T) {
		token, err := server.generateAccessToken(1)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		userID, err := session.GetProfileIDFromTokenString(token, server.JWTSecret, "access")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), userID)
	})

	t.Run("DifferentUserIDs", func(t *testing.T) {
		userIDs := []int64{1, 2, 100, 999}
		for _, userID := range userIDs {
			token, err := server.generateAccessToken(userID)
			assert.NoError(t, err)

			extractedUserID, err := session.GetProfileIDFromTokenString(token, server.JWTSecret, "access")
			assert.NoError(t, err)
			assert.Equal(t, userID, extractedUserID)
		}
	})
}

func TestServer_generateTokenPair(t *testing.T) {
	server, _ := setupTestServer()

	t.Run("Success", func(t *testing.T) {
		accessToken, refreshToken, err := server.generateTokenPair(1)

		assert.NoError(t, err)
		assert.NotEmpty(t, accessToken)
		assert.NotEmpty(t, refreshToken)

		userID, err := session.GetProfileIDFromTokenString(accessToken, server.JWTSecret, "access")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), userID)

		userID, err = session.GetProfileIDFromTokenString(refreshToken, server.JWTSecret, "refresh")
		assert.NoError(t, err)
		assert.Equal(t, int64(1), userID)
	})

	t.Run("TokenExpiration", func(t *testing.T) {
		accessToken, refreshToken, err := server.generateTokenPair(1)
		assert.NoError(t, err)

		token, _, err := new(jwt.Parser).ParseUnverified(accessToken, jwt.MapClaims{})
		assert.NoError(t, err)

		claims, ok := token.Claims.(jwt.MapClaims)
		assert.True(t, ok)

		exp, ok := claims["exp"].(float64)
		assert.True(t, ok)

		expectedExp := time.Now().Add(15 * time.Minute).Unix()
		assert.InDelta(t, expectedExp, int64(exp), 1)

		token, _, err = new(jwt.Parser).ParseUnverified(refreshToken, jwt.MapClaims{})
		assert.NoError(t, err)

		claims, ok = token.Claims.(jwt.MapClaims)
		assert.True(t, ok)

		exp, ok = claims["exp"].(float64)
		assert.True(t, ok)

		expectedExp = time.Now().Add(7 * 24 * time.Hour).Unix()
		assert.InDelta(t, expectedExp, int64(exp), 1)
	})
}

func TestServer_Login_TrimSpaces(t *testing.T) {
	server, mockProfile := setupTestServer()

	t.Run("TrimSpaces", func(t *testing.T) {
		mockProfile.On("Login", mock.Anything, profile.LoginRequest{
			Username: "testuser",
			Password: "password123",
		}).Return(int64(1), nil)

		req := &authproto.LoginRequest{
			Login:    "  testuser  ",
			Password: "  password123  ",
		}

		resp, err := server.Login(createTestContext(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockProfile.AssertExpectations(t)
	})
}

func TestServer_Signup_TrimSpaces(t *testing.T) {
	server, mockProfile := setupTestServer()

	t.Run("TrimSpaces", func(t *testing.T) {
		mockProfile.On("Signup", mock.Anything, profile.SignupRequest{
			Name:     "  Test User  ",
			Username: "  testuser  ",
			Birthday: "  1990-01-01  ",
			Gender:   "  male  ",
			Password: "  password123  ",
		}).Return(int64(1), nil)

		req := &authproto.SignupRequest{
			Name:     "  Test User  ",
			Username: "  testuser  ",
			Birthday: "  1990-01-01  ",
			Gender:   "  male  ",
			Password: "  password123  ",
		}

		resp, err := server.Signup(createTestContext(), req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		mockProfile.AssertExpectations(t)
	})
}

func TestServer_generateAccessToken_Error(t *testing.T) {
	invalidSecret := []byte("short")
	mockProfile := &MockProfileUsecase{}
	server := New(mockProfile, invalidSecret)

	token, err := server.generateAccessToken(1)

	if err != nil {
		assert.Error(t, err)
	} else {
		assert.NotEmpty(t, token)
	}
}

func TestServer_TokenStructure(t *testing.T) {
	server, mockProfile := setupTestServer()

	t.Run("TokenContainsRequiredClaims", func(t *testing.T) {
		mockProfile.On("Login", mock.Anything, mock.Anything).Return(int64(123), nil)

		resp, err := server.Login(createTestContext(), &authproto.LoginRequest{
			Login:    "testuser",
			Password: "password",
		})

		assert.NoError(t, err)

		token, _, err := new(jwt.Parser).ParseUnverified(resp.AccessToken, jwt.MapClaims{})
		assert.NoError(t, err)

		claims, ok := token.Claims.(jwt.MapClaims)
		assert.True(t, ok)
		assert.Equal(t, float64(123), claims["user_id"])
		assert.Equal(t, "access", claims["type"])
		assert.Contains(t, claims, "exp")

		token, _, err = new(jwt.Parser).ParseUnverified(resp.RefreshToken, jwt.MapClaims{})
		assert.NoError(t, err)

		claims, ok = token.Claims.(jwt.MapClaims)
		assert.True(t, ok)
		assert.Equal(t, float64(123), claims["user_id"])
		assert.Equal(t, "refresh", claims["type"])
		assert.Contains(t, claims, "exp")
	})
}

func BenchmarkServer_Login(b *testing.B) {
	server, mockProfile := setupTestServer()
	mockProfile.On("Login", mock.Anything, mock.Anything).Return(int64(1), nil)

	req := &authproto.LoginRequest{
		Login:    "testuser",
		Password: "password123",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.Login(createTestContext(), req)
	}
}

func BenchmarkServer_generateTokenPair(b *testing.B) {
	server, _ := setupTestServer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.generateTokenPair(1)
	}
}
