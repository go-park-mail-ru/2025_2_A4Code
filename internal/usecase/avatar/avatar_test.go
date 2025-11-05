package avatar

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/usecase/profile"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"
)

var (
	mockRepoError = errors.New("mock repository error")
)

type MockAvatarRepository struct {
	UploadFileFn            func(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error
	GetAvatarPresignedURLFn func(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error)
	DeleteAvatarFn          func(ctx context.Context, objectName string) error
}

func (m *MockAvatarRepository) UploadFile(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
	if m.UploadFileFn != nil {
		return m.UploadFileFn(ctx, objectName, data, size, contentType)
	}
	return nil
}

func (m *MockAvatarRepository) GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
	if m.GetAvatarPresignedURLFn != nil {
		return m.GetAvatarPresignedURLFn(ctx, objectName, duration)
	}
	u, _ := url.Parse(fmt.Sprintf("https://mock-storage/bucket/%s?token=mock", objectName))
	return u, nil
}

func (m *MockAvatarRepository) DeleteAvatar(ctx context.Context, objectName string) error {
	if m.DeleteAvatarFn != nil {
		return m.DeleteAvatarFn(ctx, objectName)
	}
	return nil
}

type MockProfileRepository struct{}

func (m *MockProfileRepository) FindByID(ctx context.Context, id int64) (*domain.Profile, error) {
	return nil, nil
}
func (m *MockProfileRepository) FindSenderByID(ctx context.Context, id int64) (*domain.Sender, error) {
	return nil, nil
}
func (m *MockProfileRepository) UserExists(ctx context.Context, username string) (bool, error) {
	return false, nil
}
func (m *MockProfileRepository) CreateUser(ctx context.Context, profile domain.Profile) (int64, error) {
	return 0, nil
}
func (m *MockProfileRepository) FindByUsernameAndDomain(ctx context.Context, username string, domain string) (*domain.Profile, error) {
	return nil, nil
}
func (m *MockProfileRepository) FindInfoByID(ctx context.Context, id int64) (domain.ProfileInfo, error) {
	return domain.ProfileInfo{}, nil
}
func (m *MockProfileRepository) FindSettingsByProfileId(ctx context.Context, profileID int64) (domain.Settings, error) {
	return domain.Settings{}, nil
}
func (m *MockProfileRepository) InsertProfileAvatar(ctx context.Context, profileID int64, avatarURL string) error {
	return nil
}
func (m *MockProfileRepository) UpdateProfileInfo(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
	return nil
}

type dummyReader struct{}

func (d *dummyReader) Read(p []byte) (n int, err error) { return 0, io.EOF }

func TestAvatarUcase_UploadAvatar(t *testing.T) {
	type fields struct {
		avatarRepo  AvatarRepository
		profileRepo profile.ProfileRepository
	}
	type args struct {
		ctx              context.Context
		userID           string
		file             io.Reader
		size             int64
		originalFilename string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantPrefix string
		wantExt    string
		wantErr    bool
	}{
		{
			name: "Success_WithExtension",
			fields: fields{
				avatarRepo: &MockAvatarRepository{
					UploadFileFn: func(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
						if !strings.HasSuffix(objectName, ".png") || !strings.Contains(objectName, "testuser_id") {
							t.Errorf("UploadFile called with unexpected objectName: %s", objectName)
						}
						return nil
					},
					GetAvatarPresignedURLFn: func(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
						u, _ := url.Parse("avatar/")
						return u, nil
					},
				},
				profileRepo: &MockProfileRepository{},
			},
			args: args{
				ctx:              context.Background(),
				userID:           "testuser_id",
				file:             &dummyReader{},
				size:             1024,
				originalFilename: "image.png",
			},
			wantPrefix: "avatar/testuser_id/",
			wantExt:    ".png",
			wantErr:    false,
		},
		{
			name: "Success_NoExtension_DefaultsToJpg",
			fields: fields{
				avatarRepo: &MockAvatarRepository{
					UploadFileFn: func(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
						if !strings.HasSuffix(objectName, ".jpg") {
							t.Errorf("UploadFile did not default to .jpg for objectName: %s", objectName)
						}
						return nil
					},
					GetAvatarPresignedURLFn: func(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
						u, _ := url.Parse("avatar/")
						return u, nil
					},
				},
				profileRepo: &MockProfileRepository{},
			},
			args: args{
				ctx:              context.Background(),
				userID:           "user_2",
				file:             &dummyReader{},
				size:             512,
				originalFilename: "imagefile",
			},
			wantPrefix: "avatar/user_2/",
			wantExt:    ".jpg",
			wantErr:    false,
		},
		{
			name: "Failure_UploadFileError",
			fields: fields{
				avatarRepo: &MockAvatarRepository{
					UploadFileFn: func(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
						return mockRepoError
					},
				},
				profileRepo: &MockProfileRepository{},
			},
			args: args{
				ctx:              context.Background(),
				userID:           "testuser_id",
				file:             &dummyReader{},
				size:             1024,
				originalFilename: "image.png",
			},
			wantPrefix: "",
			wantExt:    "",
			wantErr:    true,
		},
		{
			name: "Failure_PresignedURLError",
			fields: fields{
				avatarRepo: &MockAvatarRepository{
					UploadFileFn: func(ctx context.Context, objectName string, data io.Reader, size int64, contentType string) error {
						return nil
					},
					GetAvatarPresignedURLFn: func(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error) {
						return nil, mockRepoError
					},
				},
				profileRepo: &MockProfileRepository{},
			},
			args: args{
				ctx:              context.Background(),
				userID:           "testuser_id",
				file:             &dummyReader{},
				size:             1024,
				originalFilename: "image.png",
			},
			wantPrefix: "",
			wantExt:    "",
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := New(tt.fields.avatarRepo, tt.fields.profileRepo)
			got, got1, err := uc.UploadAvatar(tt.args.ctx, tt.args.userID, tt.args.file, tt.args.size, tt.args.originalFilename)

			if (err != nil) != tt.wantErr {
				t.Fatalf("UploadAvatar() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !strings.HasPrefix(got, tt.wantPrefix) {
					t.Errorf("UploadAvatar() objectName prefix got = %s, wantPrefix %s", got, tt.wantPrefix)
				}
				if !strings.HasSuffix(got, tt.wantExt) {
					t.Errorf("UploadAvatar() objectName extension got = %s, wantExt %s", got, tt.wantExt)
				}
				if !strings.Contains(got, got1) {
					t.Errorf("UploadAvatar() presignedURL does not contain objectName: url = %s, objectName = %s", got1, got)
				}
			} else {
				if got != "" {
					t.Errorf("UploadAvatar() got objectName = %v, want empty string on error", got)
				}
				if got1 != "" {
					t.Errorf("UploadAvatar() got presignedURL = %v, want empty string on error", got1)
				}
			}
		})
	}
}
