package profile

import (
	"context"
	"errors"
	"testing"
	"time"

	"2025_2_a4code/internal/domain"
	commone "2025_2_a4code/internal/lib/errors"

	"golang.org/x/crypto/bcrypt"
)

var ErrWantParseError = errors.New("WANT_TIME_PARSE_ERROR")

type MockProfileRepository struct {
	UserExistsFn              func(ctx context.Context, username string) (bool, error)
	CreateUserFn              func(ctx context.Context, profile domain.Profile) (int64, error)
	FindByUsernameAndDomainFn func(ctx context.Context, username string, domain string) (*domain.Profile, error)
}

func (m *MockProfileRepository) FindByID(ctx context.Context, id int64) (*domain.Profile, error) {
	return nil, nil
}
func (m *MockProfileRepository) FindSenderByID(ctx context.Context, id int64) (*domain.Sender, error) {
	return nil, nil
}
func (m *MockProfileRepository) UserExists(ctx context.Context, username string) (bool, error) {
	if m.UserExistsFn != nil {
		return m.UserExistsFn(ctx, username)
	}
	return false, nil
}
func (m *MockProfileRepository) CreateUser(ctx context.Context, profile domain.Profile) (int64, error) {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(ctx, profile)
	}
	return 1, nil
}
func (m *MockProfileRepository) FindByUsernameAndDomain(ctx context.Context, username string, domain string) (*domain.Profile, error) {
	if m.FindByUsernameAndDomainFn != nil {
		return m.FindByUsernameAndDomainFn(ctx, username, domain)
	}
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

func generateHash(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func TestProfileUcase_Signup(t *testing.T) {
	const testPassword = "testpassword123"
	const testUserID = 42
	mockRepoError := errors.New("repository failed")

	type fields struct {
		repo ProfileRepository
	}
	type args struct {
		ctx       context.Context
		SignupReq SignupRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr error
	}{
		{
			name: "Success: User created",
			fields: fields{
				repo: &MockProfileRepository{
					UserExistsFn: func(ctx context.Context, username string) (bool, error) { return false, nil },
					CreateUserFn: func(ctx context.Context, profile domain.Profile) (int64, error) { return testUserID, nil },
				},
			},
			args: args{
				ctx: context.Background(),
				SignupReq: SignupRequest{
					Name:     "Test",
					Username: "newuser",
					Birthday: "01.01.2000",
					Gender:   "male",
					Password: testPassword,
				},
			},
			want:    testUserID,
			wantErr: nil,
		},
		{
			name: "Failure: User already exists",
			fields: fields{
				repo: &MockProfileRepository{
					UserExistsFn: func(ctx context.Context, username string) (bool, error) { return true, nil },
				},
			},
			args: args{
				ctx: context.Background(),
				SignupReq: SignupRequest{
					Username: "existinguser",
					Birthday: "01.01.2000",
					Password: testPassword,
				},
			},
			want:    0,
			wantErr: ErrUserAlreadyExists,
		},
		{
			name: "Failure: Repository error on UserExists",
			fields: fields{
				repo: &MockProfileRepository{
					UserExistsFn: func(ctx context.Context, username string) (bool, error) { return false, mockRepoError },
				},
			},
			args: args{
				ctx: context.Background(),
				SignupReq: SignupRequest{
					Username: "user",
					Birthday: "01.01.2000",
					Password: testPassword,
				},
			},
			want:    0,
			wantErr: mockRepoError,
		},
		{
			name: "Failure: Invalid Birthday format",
			fields: fields{
				repo: &MockProfileRepository{
					UserExistsFn: func(ctx context.Context, username string) (bool, error) { return false, nil },
				},
			},
			args: args{
				ctx: context.Background(),
				SignupReq: SignupRequest{
					Username: "user",
					Birthday: "2000-01-01",
					Password: testPassword,
				},
			},
			want:    0,
			wantErr: ErrWantParseError,
		},
		{
			name: "Failure: Repository error on CreateUser",
			fields: fields{
				repo: &MockProfileRepository{
					UserExistsFn: func(ctx context.Context, username string) (bool, error) { return false, nil },
					CreateUserFn: func(ctx context.Context, profile domain.Profile) (int64, error) { return 0, mockRepoError },
				},
			},
			args: args{
				ctx: context.Background(),
				SignupReq: SignupRequest{
					Username: "user",
					Birthday: "01.01.2000",
					Password: testPassword,
				},
			},
			want:    0,
			wantErr: ErrUserCreationFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.Signup(tt.args.ctx, tt.args.SignupReq)

			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("Signup() error presence mismatch: got error %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				if errors.Is(tt.wantErr, ErrWantParseError) {
					var parseErr *time.ParseError
					if !errors.As(err, &parseErr) {
						t.Errorf("Signup() returned error = %v, want a wrapped *time.ParseError", err)
					}
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("Signup() error mismatch: got %v, want %v", err, tt.wantErr)
				}
			}

			if got != tt.want {
				t.Errorf("Signup() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileUcase_Login(t *testing.T) {
	const testPassword = "testpassword123"
	const wrongPassword = "wrongpassword"
	const testUserID = 42
	hashedPassword := generateHash(testPassword)
	mockRepoError := errors.New("repository failed")

	validProfile := &domain.Profile{
		ID:           testUserID,
		PasswordHash: hashedPassword,
	}

	type fields struct {
		repo ProfileRepository
	}
	type args struct {
		ctx context.Context
		req LoginRequest
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr error
	}{
		{
			name: "Success: Valid login",
			fields: fields{
				repo: &MockProfileRepository{
					FindByUsernameAndDomainFn: func(ctx context.Context, username string, domain string) (*domain.Profile, error) {
						return validProfile, nil
					},
				},
			},
			args: args{
				ctx: context.Background(),
				req: LoginRequest{
					Username: "testuser",
					Password: testPassword,
				},
			},
			want:    testUserID,
			wantErr: nil,
		},
		{
			name: "Failure: User not found",
			fields: fields{
				repo: &MockProfileRepository{
					FindByUsernameAndDomainFn: func(ctx context.Context, username string, domain string) (*domain.Profile, error) {
						return nil, commone.ErrNotFound
					},
				},
			},
			args: args{
				ctx: context.Background(),
				req: LoginRequest{
					Username: "unknown",
					Password: testPassword,
				},
			},
			want:    0,
			wantErr: ErrUserNotFound,
		},
		{
			name: "Failure: Repository error (not notFound)",
			fields: fields{
				repo: &MockProfileRepository{
					FindByUsernameAndDomainFn: func(ctx context.Context, username string, domain string) (*domain.Profile, error) {
						return nil, mockRepoError
					},
				},
			},
			args: args{
				ctx: context.Background(),
				req: LoginRequest{
					Username: "testuser",
					Password: testPassword,
				},
			},
			want:    0,
			wantErr: mockRepoError,
		},
		{
			name: "Failure: Wrong password",
			fields: fields{
				repo: &MockProfileRepository{
					FindByUsernameAndDomainFn: func(ctx context.Context, username string, domain string) (*domain.Profile, error) {
						return validProfile, nil
					},
				},
			},
			args: args{
				ctx: context.Background(),
				req: LoginRequest{
					Username: "testuser",
					Password: wrongPassword,
				},
			},
			want:    0,
			wantErr: ErrWrongPassword,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{
				repo: tt.fields.repo,
			}
			got, err := uc.Login(tt.args.ctx, tt.args.req)

			if (err != nil) != (tt.wantErr != nil) {
				t.Fatalf("Login() error presence mismatch: got error %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Login() error mismatch: got %v, want %v", err, tt.wantErr)
			}

			if got != tt.want {
				t.Errorf("Login() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileUcase_checkPassword(t *testing.T) {
	const testPassword = "testpassword123"
	const wrongPassword = "wrongpassword"

	validHash := generateHash(testPassword)
	invalidHash := "notahash"

	type fields struct {
		repo ProfileRepository
	}
	type args struct {
		password string
		hash     string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Success: Correct password",
			args: args{
				password: testPassword,
				hash:     validHash,
			},
			want: true,
		},
		{
			name: "Failure: Wrong password",
			args: args{
				password: wrongPassword,
				hash:     validHash,
			},
			want: false,
		},
		{
			name: "Failure: Empty hash",
			args: args{
				password: testPassword,
				hash:     "",
			},
			want: false,
		},
		{
			name: "Failure: Invalid hash format",
			args: args{
				password: testPassword,
				hash:     invalidHash,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{
				repo: tt.fields.repo,
			}
			if got := uc.checkPassword(tt.args.password, tt.args.hash); got != tt.want {
				t.Errorf("checkPassword() got = %v, want %v", got, tt.want)
			}
		})
	}
}
