package profile_service

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/usecase/profile"
	"context"
	"errors"
	"io"
	"net/url"
	"testing"
	"time"

	pb "2025_2_a4code/profile-service/pkg/profileproto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type MockProfileUsecase struct {
	mock.Mock
}

type MockAvatarUsecase struct {
	mock.Mock
}

func (m *MockProfileUsecase) FindInfoByID(ctx context.Context, id int64) (domain.ProfileInfo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.ProfileInfo), args.Error(1)
}

func (m *MockProfileUsecase) UpdateProfileInfo(ctx context.Context, profileID int64, req profile.UpdateProfileRequest) error {
	args := m.Called(ctx, profileID, req)
	return args.Error(0)
}

func (m *MockProfileUsecase) FindSettingsByProfileId(ctx context.Context, profileID int64) (domain.Settings, error) {
	args := m.Called(ctx, profileID)
	return args.Get(0).(domain.Settings), args.Error(1)
}

func (m *MockProfileUsecase) InsertProfileAvatar(ctx context.Context, profileID int64, avatarURL string) error {
	args := m.Called(ctx, profileID, avatarURL)
	return args.Error(0)
}

func (m *MockAvatarUsecase) GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
	args := m.Called(ctx, objectName, duration)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*url.URL), args.Error(1)
}

func (m *MockAvatarUsecase) UploadAvatar(ctx context.Context, userID string, file io.Reader, size int64, originalFilename string) (string, string, error) {
	args := m.Called(ctx, userID, file, size, originalFilename)
	return args.String(0), args.String(1), args.Error(2)
}

func setupTestServer() (*Server, *MockProfileUsecase, *MockAvatarUsecase) {
	mockProfileUsecase := &MockProfileUsecase{}
	mockAvatarUsecase := &MockAvatarUsecase{}
	jwtSecret := []byte("test-secret-key-very-long-for-testing")
	server := New(mockProfileUsecase, mockAvatarUsecase, jwtSecret)
	return server, mockProfileUsecase, mockAvatarUsecase
}

func generateTestToken(userID int64, secret []byte, tokenType string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    tokenType,
	})
	return token.SignedString(secret)
}

func createTestContextWithToken(userID int64, secret []byte) context.Context {
	token, err := generateTestToken(userID, secret, "access")
	if err != nil {
		panic(err)
	}
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	return metadata.NewIncomingContext(context.Background(), md)
}

func createTestContextWithoutAuth() context.Context {
	return context.Background()
}

func createTestContextWithInvalidToken() context.Context {
	md := metadata.New(map[string]string{"authorization": "Bearer invalid-token"})
	return metadata.NewIncomingContext(context.Background(), md)
}

func TestServer_GetProfile(t *testing.T) {
	server, mockProfile, mockAvatar := setupTestServer()

	tests := []struct {
		name          string
		ctx           context.Context
		mockSetup     func()
		expectedError bool
		expectedCode  codes.Code
	}{
		{
			name: "Success",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			mockSetup: func() {
				mockProfile.On("FindInfoByID", mock.Anything, int64(1)).Return(domain.ProfileInfo{
					ID:         1,
					Username:   "testuser",
					CreatedAt:  time.Now(),
					Name:       "John",
					Surname:    "Doe",
					Patronymic: "Middle",
					Gender:     "male",
					Birthday:   "1990-01-01",
					AvatarPath: "avatar.jpg",
				}, nil)
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
					Scheme: "https",
					Host:   "example.com",
					Path:   "/avatar.jpg",
				}, nil)
			},
			expectedError: false,
		},
		{
			name: "Unauthorized_NoMetadata",
			ctx:  createTestContextWithoutAuth(),
			mockSetup: func() {
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "Unauthorized_InvalidToken",
			ctx:  createTestContextWithInvalidToken(),
			mockSetup: func() {
			},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "ProfileNotFound",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			mockSetup: func() {
				mockProfile.On("FindInfoByID", mock.Anything, int64(1)).Return(domain.ProfileInfo{}, errors.New("not found"))
			},
			expectedError: true,
			expectedCode:  codes.NotFound,
		},
		{
			name: "InternalError",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			mockSetup: func() {
				mockProfile.On("FindInfoByID", mock.Anything, int64(1)).Return(domain.ProfileInfo{}, errors.New("database error"))
			},
			expectedError: true,
			expectedCode:  codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := server.GetProfile(tt.ctx, &pb.GetProfileRequest{})

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
				assert.NotNil(t, resp.Profile)
				assert.Equal(t, "1", resp.Profile.Id)
				assert.Equal(t, "John", resp.Profile.Name)
				assert.Equal(t, "Doe", resp.Profile.Surname)
				assert.Equal(t, "https://example.com/avatar.jpg", resp.Profile.AvatarPath)
			}

			mockProfile.AssertExpectations(t)
			mockAvatar.AssertExpectations(t)
		})
	}
}

func TestServer_UpdateProfile(t *testing.T) {
	server, mockProfile, mockAvatar := setupTestServer()

	validRequest := &pb.UpdateProfileRequest{
		Name:       "UpdatedName",
		Surname:    "UpdatedSurname",
		Patronymic: "UpdatedPatronymic",
		Gender:     "female",
		Birthday:   "1995-05-15",
	}

	tests := []struct {
		name          string
		ctx           context.Context
		request       *pb.UpdateProfileRequest
		mockSetup     func()
		expectedError bool
		expectedCode  codes.Code
	}{
		{
			name:    "Success",
			ctx:     createTestContextWithToken(1, server.JWTSecret),
			request: validRequest,
			mockSetup: func() {
				mockProfile.On("UpdateProfileInfo", mock.Anything, int64(1), profile.UpdateProfileRequest{
					FirstName:  "UpdatedName",
					LastName:   "UpdatedSurname",
					MiddleName: "UpdatedPatronymic",
					Gender:     "female",
					Birthday:   "1995-05-15",
				}).Return(nil)
				mockProfile.On("FindInfoByID", mock.Anything, int64(1)).Return(domain.ProfileInfo{
					ID:         1,
					Username:   "testuser",
					CreatedAt:  time.Now(),
					Name:       "UpdatedName",
					Surname:    "UpdatedSurname",
					Patronymic: "UpdatedPatronymic",
					Gender:     "female",
					Birthday:   "1995-05-15",
					AvatarPath: "avatar.jpg",
				}, nil)
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
					Scheme: "https",
					Host:   "example.com",
					Path:   "/avatar.jpg",
				}, nil)
			},
			expectedError: false,
		},
		{
			name:          "Unauthorized",
			ctx:           createTestContextWithoutAuth(),
			request:       validRequest,
			mockSetup:     func() {},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "UpdateError",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			request: &pb.UpdateProfileRequest{
				Name:   "Invalid",
				Gender: "invalid",
			},
			mockSetup: func() {
				mockProfile.On("UpdateProfileInfo", mock.Anything, int64(1), mock.Anything).Return(errors.New("validation error"))
			},
			expectedError: true,
			expectedCode:  codes.InvalidArgument,
		},
		{
			name: "GetUpdatedProfileError",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			request: &pb.UpdateProfileRequest{
				Name: "John",
			},
			mockSetup: func() {
				mockProfile.On("UpdateProfileInfo", mock.Anything, int64(1), mock.Anything).Return(nil)
				mockProfile.On("FindInfoByID", mock.Anything, int64(1)).Return(domain.ProfileInfo{}, errors.New("not found"))
			},
			expectedError: true,
			expectedCode:  codes.NotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := server.UpdateProfile(tt.ctx, tt.request)

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
				assert.NotNil(t, resp.Profile)
				assert.Equal(t, "UpdatedName", resp.Profile.Name)
				assert.Equal(t, "UpdatedSurname", resp.Profile.Surname)
				assert.Equal(t, "female", resp.Profile.Gender)
			}

			mockProfile.AssertExpectations(t)
			mockAvatar.AssertExpectations(t)
		})
	}
}

func TestServer_Settings(t *testing.T) {
	server, mockProfile, _ := setupTestServer()

	tests := []struct {
		name          string
		ctx           context.Context
		mockSetup     func()
		expectedError bool
		expectedCode  codes.Code
	}{
		{
			name: "Success",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			mockSetup: func() {
				mockProfile.On("FindSettingsByProfileId", mock.Anything, int64(1)).Return(domain.Settings{
					NotificationTolerance: "30",
					Language:              "en",
					Theme:                 "dark",
					Signatures:            []string{"Best regards", "Thanks"},
				}, nil)
			},
			expectedError: false,
		},
		{
			name:          "Unauthorized",
			ctx:           createTestContextWithoutAuth(),
			mockSetup:     func() {},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "InternalError",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			mockSetup: func() {
				mockProfile.On("FindSettingsByProfileId", mock.Anything, int64(1)).Return(domain.Settings{}, errors.New("database error"))
			},
			expectedError: true,
			expectedCode:  codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := server.Settings(tt.ctx, &pb.SettingsRequest{})

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
				assert.NotNil(t, resp.Settings)
				assert.Equal(t, "30", resp.Settings.NotificationTolerance)
				assert.Equal(t, "en", resp.Settings.Language)
				assert.Equal(t, "dark", resp.Settings.Theme)
				assert.Len(t, resp.Settings.Signatures, 2)
				assert.Equal(t, "Best regards", resp.Settings.Signatures[0])
			}

			mockProfile.AssertExpectations(t)
		})
	}
}

func TestServer_UploadAvatar(t *testing.T) {
	server, mockProfile, mockAvatar := setupTestServer()

	smallAvatarData := make([]byte, 1024) // 1KB
	largeAvatarData := make([]byte, maxAvatarSize+1)

	tests := []struct {
		name          string
		ctx           context.Context
		request       *pb.UploadAvatarRequest
		mockSetup     func()
		expectedError bool
		expectedCode  codes.Code
	}{
		{
			name: "Success",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			request: &pb.UploadAvatarRequest{
				AvatarData:  smallAvatarData,
				FileName:    "avatar.jpg",
				ContentType: "image/jpeg",
			},
			mockSetup: func() {
				mockAvatar.On("UploadAvatar", mock.Anything, "1", mock.Anything, int64(1024), "avatar.jpg").Return("object-key", "https://example.com/avatar.jpg", nil)
				mockProfile.On("InsertProfileAvatar", mock.Anything, int64(1), "object-key").Return(nil)
			},
			expectedError: false,
		},
		{
			name:          "Unauthorized",
			ctx:           createTestContextWithoutAuth(),
			request:       &pb.UploadAvatarRequest{},
			mockSetup:     func() {},
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "FileTooLarge",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			request: &pb.UploadAvatarRequest{
				AvatarData: largeAvatarData,
				FileName:   "large.jpg",
			},
			mockSetup:     func() {},
			expectedError: true,
			expectedCode:  codes.InvalidArgument,
		},
		{
			name: "UploadError",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			request: &pb.UploadAvatarRequest{
				AvatarData: smallAvatarData,
				FileName:   "avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("UploadAvatar", mock.Anything, "1", mock.Anything, int64(1024), "avatar.jpg").Return("", "", errors.New("upload failed"))
			},
			expectedError: true,
			expectedCode:  codes.Internal,
		},
		{
			name: "InsertAvatarError",
			ctx:  createTestContextWithToken(1, server.JWTSecret),
			request: &pb.UploadAvatarRequest{
				AvatarData: smallAvatarData,
				FileName:   "avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("UploadAvatar", mock.Anything, "1", mock.Anything, int64(1024), "avatar.jpg").Return("object-key", "https://example.com/avatar.jpg", nil)
				mockProfile.On("InsertProfileAvatar", mock.Anything, int64(1), "object-key").Return(errors.New("database error"))
			},
			expectedError: true,
			expectedCode:  codes.Internal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			resp, err := server.UploadAvatar(tt.ctx, tt.request)

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
				assert.Equal(t, "https://example.com/avatar.jpg", resp.AvatarPath)
			}

			mockProfile.AssertExpectations(t)
			mockAvatar.AssertExpectations(t)
		})
	}
}

func TestServer_getProfileID(t *testing.T) {
	server, _, _ := setupTestServer()

	tests := []struct {
		name          string
		ctx           context.Context
		expectedError bool
		expectedCode  codes.Code
	}{
		{
			name: "Success",
			ctx: func() context.Context {
				token, _ := generateTestToken(1, server.JWTSecret, "access")
				md := metadata.New(map[string]string{"authorization": "Bearer " + token})
				return metadata.NewIncomingContext(context.Background(), md)
			}(),
			expectedError: false,
		},
		{
			name:          "NoMetadata",
			ctx:           context.Background(),
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "NoAuthorizationHeader",
			ctx: func() context.Context {
				md := metadata.New(map[string]string{})
				return metadata.NewIncomingContext(context.Background(), md)
			}(),
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "InvalidToken",
			ctx: func() context.Context {
				md := metadata.New(map[string]string{"authorization": "Bearer invalid-token"})
				return metadata.NewIncomingContext(context.Background(), md)
			}(),
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
		{
			name: "WrongTokenType",
			ctx: func() context.Context {
				token, _ := generateTestToken(1, server.JWTSecret, "refresh")
				md := metadata.New(map[string]string{"authorization": "Bearer " + token})
				return metadata.NewIncomingContext(context.Background(), md)
			}(),
			expectedError: true,
			expectedCode:  codes.Unauthenticated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profileID, err := server.getProfileID(tt.ctx)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.expectedCode != codes.OK {
					grpcStatus, ok := status.FromError(err)
					assert.True(t, ok)
					assert.Equal(t, tt.expectedCode, grpcStatus.Code())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, int64(1), profileID)
			}
		})
	}
}

func TestServer_domainProfileToProto(t *testing.T) {
	server, _, _ := setupTestServer()

	profileInfo := domain.ProfileInfo{
		ID:         1,
		Username:   "testuser",
		CreatedAt:  time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		Name:       "John",
		Surname:    "Doe",
		Patronymic: "Middle",
		Gender:     "male",
		Birthday:   "1990-01-01",
		AvatarPath: "avatar.jpg",
	}

	pbProfile := server.domainProfileToProto(profileInfo)

	assert.Equal(t, "1", pbProfile.Id)
	assert.Equal(t, "testuser", pbProfile.Username)
	assert.Equal(t, "2023-01-01T00:00:00Z", pbProfile.CreatedAt)
	assert.Equal(t, "John", pbProfile.Name)
	assert.Equal(t, "Doe", pbProfile.Surname)
	assert.Equal(t, "Middle", pbProfile.Patronymic)
	assert.Equal(t, "male", pbProfile.Gender)
	assert.Equal(t, "1990-01-01", pbProfile.Birthday)
	assert.Equal(t, "avatar.jpg", pbProfile.AvatarPath)
}

func TestServer_domainSettingsToProto(t *testing.T) {
	server, _, _ := setupTestServer()

	settings := domain.Settings{
		NotificationTolerance: "30",
		Language:              "en",
		Theme:                 "dark",
		Signatures:            []string{"Sig1", "Sig2"},
	}

	pbSettings := server.domainSettingsToProto(settings)

	assert.Equal(t, "30", pbSettings.NotificationTolerance)
	assert.Equal(t, "en", pbSettings.Language)
	assert.Equal(t, "dark", pbSettings.Theme)
	assert.Len(t, pbSettings.Signatures, 2)
	assert.Equal(t, "Sig1", pbSettings.Signatures[0])
	assert.Equal(t, "Sig2", pbSettings.Signatures[1])
}

func TestServer_enrichAvatarURL(t *testing.T) {
	server, _, mockAvatar := setupTestServer()

	tests := []struct {
		name          string
		profileInfo   *domain.ProfileInfo
		mockSetup     func()
		expectedError bool
		expectedURL   string
	}{
		{
			name: "SuccessWithSimplePath",
			profileInfo: &domain.ProfileInfo{
				AvatarPath: "avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
					Scheme: "https",
					Host:   "example.com",
					Path:   "/avatar.jpg",
				}, nil)
			},
			expectedError: false,
			expectedURL:   "https://example.com/avatar.jpg",
		},
		{
			name: "SuccessWithHTTPPrefix",
			profileInfo: &domain.ProfileInfo{
				AvatarPath: "http://old.example.com/avatars/avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
					Scheme: "https",
					Host:   "new.example.com",
					Path:   "/avatar.jpg",
				}, nil)
			},
			expectedError: false,
			expectedURL:   "https://new.example.com/avatar.jpg",
		},
		{
			name: "SuccessWithHTTPSPrefix",
			profileInfo: &domain.ProfileInfo{
				AvatarPath: "https://old.example.com/avatars/avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
					Scheme: "https",
					Host:   "new.example.com",
					Path:   "/avatar.jpg",
				}, nil)
			},
			expectedError: false,
			expectedURL:   "https://new.example.com/avatar.jpg",
		},
		{
			name: "EmptyAvatarPath",
			profileInfo: &domain.ProfileInfo{
				AvatarPath: "",
			},
			mockSetup:     func() {},
			expectedError: false,
			expectedURL:   "",
		},
		{
			name: "GetPresignedURLError",
			profileInfo: &domain.ProfileInfo{
				AvatarPath: "avatar.jpg",
			},
			mockSetup: func() {
				mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(nil, errors.New("s3 error"))
			},
			expectedError: true,
			expectedURL:   "avatar.jpg",
		},
		{
			name: "EmptyAfterProcessing",
			profileInfo: &domain.ProfileInfo{
				AvatarPath: "http://example.com/",
			},
			mockSetup:     func() {},
			expectedError: false,
			expectedURL:   "http://example.com/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := server.enrichAvatarURL(context.Background(), tt.profileInfo)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedURL != "" {
				assert.Equal(t, tt.expectedURL, tt.profileInfo.AvatarPath)
			}

			mockAvatar.AssertExpectations(t)
		})
	}
}

func BenchmarkServer_GetProfile(b *testing.B) {
	server, mockProfile, mockAvatar := setupTestServer()

	token, _ := generateTestToken(1, server.JWTSecret, "access")
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	mockProfile.On("FindInfoByID", mock.Anything, int64(1)).Return(domain.ProfileInfo{
		ID:         1,
		Username:   "testuser",
		CreatedAt:  time.Now(),
		Name:       "John",
		Surname:    "Doe",
		Patronymic: "Middle",
		Gender:     "male",
		Birthday:   "1990-01-01",
		AvatarPath: "avatar.jpg",
	}, nil)
	mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
		Scheme: "https",
		Host:   "example.com",
		Path:   "/avatar.jpg",
	}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.GetProfile(ctx, &pb.GetProfileRequest{})
	}
}

func BenchmarkServer_UpdateProfile(b *testing.B) {
	server, mockProfile, mockAvatar := setupTestServer()

	token, _ := generateTestToken(1, server.JWTSecret, "access")
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	req := &pb.UpdateProfileRequest{
		Name:    "John",
		Surname: "Doe",
		Gender:  "male",
	}

	mockProfile.On("UpdateProfileInfo", mock.Anything, int64(1), mock.Anything).Return(nil)
	mockProfile.On("FindInfoByID", mock.Anything, int64(1)).Return(domain.ProfileInfo{
		ID:         1,
		Username:   "testuser",
		CreatedAt:  time.Now(),
		Name:       "John",
		Surname:    "Doe",
		Patronymic: "Middle",
		Gender:     "male",
		Birthday:   "1990-01-01",
		AvatarPath: "avatar.jpg",
	}, nil)
	mockAvatar.On("GetAvatarPresignedURL", mock.Anything, "avatar.jpg", mock.Anything).Return(&url.URL{
		Scheme: "https",
		Host:   "example.com",
		Path:   "/avatar.jpg",
	}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.UpdateProfile(ctx, req)
	}
}

func BenchmarkServer_Settings(b *testing.B) {
	server, mockProfile, _ := setupTestServer()

	token, _ := generateTestToken(1, server.JWTSecret, "access")
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	mockProfile.On("FindSettingsByProfileId", mock.Anything, int64(1)).Return(domain.Settings{
		NotificationTolerance: "30",
		Language:              "en",
		Theme:                 "dark",
		Signatures:            []string{"Best regards"},
	}, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.Settings(ctx, &pb.SettingsRequest{})
	}
}

func BenchmarkServer_UploadAvatar(b *testing.B) {
	server, mockProfile, mockAvatar := setupTestServer()

	token, _ := generateTestToken(1, server.JWTSecret, "access")
	md := metadata.New(map[string]string{"authorization": "Bearer " + token})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	avatarData := make([]byte, 1024)
	req := &pb.UploadAvatarRequest{
		AvatarData: avatarData,
		FileName:   "avatar.jpg",
	}

	mockAvatar.On("UploadAvatar", mock.Anything, "1", mock.Anything, int64(1024), "avatar.jpg").Return("object-key", "https://example.com/avatar.jpg", nil)
	mockProfile.On("InsertProfileAvatar", mock.Anything, int64(1), "object-key").Return(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		server.UploadAvatar(ctx, req)
	}
}

func TestServer_UploadAvatar_FileReader(t *testing.T) {
	server, mockProfile, mockAvatar := setupTestServer()

	avatarData := []byte("test avatar data")
	req := &pb.UploadAvatarRequest{
		AvatarData: avatarData,
		FileName:   "test.jpg",
	}

	var capturedReader io.Reader
	mockAvatar.On("UploadAvatar", mock.Anything, "1", mock.Anything, int64(len(avatarData)), "test.jpg").Run(func(args mock.Arguments) {
		capturedReader = args.Get(2).(io.Reader)
	}).Return("object-key", "https://example.com/avatar.jpg", nil)
	mockProfile.On("InsertProfileAvatar", mock.Anything, int64(1), "object-key").Return(nil)

	ctx := createTestContextWithToken(1, server.JWTSecret)
	resp, err := server.UploadAvatar(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "https://example.com/avatar.jpg", resp.AvatarPath)

	if capturedReader != nil {
		readData, err := io.ReadAll(capturedReader)
		assert.NoError(t, err)
		assert.Equal(t, avatarData, readData)
	}

	mockProfile.AssertExpectations(t)
	mockAvatar.AssertExpectations(t)
}

func TestServer_getProfileID_EdgeCases(t *testing.T) {
	server, _, _ := setupTestServer()

	tests := []struct {
		name          string
		tokenString   string
		expectedError bool
	}{
		{
			name:          "EmptyBearer",
			tokenString:   "Bearer ",
			expectedError: true,
		},
		{
			name:          "MalformedBearer",
			tokenString:   "Bearer",
			expectedError: true,
		},
		{
			name:          "NoBearerPrefix",
			tokenString:   "token123",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := metadata.New(map[string]string{"authorization": tt.tokenString})
			ctx := metadata.NewIncomingContext(context.Background(), md)

			profileID, err := server.getProfileID(ctx)

			assert.Error(t, err)
			assert.Equal(t, int64(0), profileID)
		})
	}
}
