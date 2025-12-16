package profile_service

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/usecase/profile"
	"context"
	"io"
	"net/url"
	"testing"
	"time"

	pb "2025_2_a4code/profile-service/pkg/profileproto"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
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
