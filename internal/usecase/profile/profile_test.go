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
	FindByIDFn                func(ctx context.Context, id int64) (*domain.Profile, error)
	FindSenderByIDFn          func(ctx context.Context, id int64) (*domain.Sender, error)
	FindInfoByIDFn            func(ctx context.Context, id int64) (domain.ProfileInfo, error)
	FindSettingsByProfileIdFn func(ctx context.Context, profileID int64) (domain.Settings, error)
	InsertProfileAvatarFn     func(ctx context.Context, profileID int64, avatarURL string) error
	UpdateProfileInfoFn       func(ctx context.Context, profileID int64, info domain.ProfileUpdate) error
}

func (m *MockProfileRepository) FindByID(ctx context.Context, id int64) (*domain.Profile, error) {
	if m.FindByIDFn != nil {
		return m.FindByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *MockProfileRepository) FindSenderByID(ctx context.Context, id int64) (*domain.Sender, error) {
	if m.FindSenderByIDFn != nil {
		return m.FindSenderByIDFn(ctx, id)
	}
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
	if m.FindInfoByIDFn != nil {
		return m.FindInfoByIDFn(ctx, id)
	}
	return domain.ProfileInfo{}, nil
}

func (m *MockProfileRepository) FindSettingsByProfileId(ctx context.Context, profileID int64) (domain.Settings, error) {
	if m.FindSettingsByProfileIdFn != nil {
		return m.FindSettingsByProfileIdFn(ctx, profileID)
	}
	return domain.Settings{}, nil
}

func (m *MockProfileRepository) InsertProfileAvatar(ctx context.Context, profileID int64, avatarURL string) error {
	if m.InsertProfileAvatarFn != nil {
		return m.InsertProfileAvatarFn(ctx, profileID, avatarURL)
	}
	return nil
}

func (m *MockProfileRepository) UpdateProfileInfo(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
	if m.UpdateProfileInfoFn != nil {
		return m.UpdateProfileInfoFn(ctx, profileID, info)
	}
	return nil
}

func generateHash(password string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash)
}

func TestProfileUcase_FindByID(t *testing.T) {
	mockProfile := &domain.Profile{ID: 1, Name: "Test User"}
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		id        int64
		want      *domain.Profile
		wantErr   error
	}{
		{
			name: "Success",
			repoSetup: &MockProfileRepository{
				FindByIDFn: func(ctx context.Context, id int64) (*domain.Profile, error) {
					return mockProfile, nil
				},
			},
			id:      1,
			want:    mockProfile,
			wantErr: nil,
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				FindByIDFn: func(ctx context.Context, id int64) (*domain.Profile, error) {
					return nil, mockError
				},
			},
			id:      1,
			want:    nil,
			wantErr: mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			got, err := uc.FindByID(context.Background(), tt.id)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("FindByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FindByID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileUcase_FindSenderByID(t *testing.T) {
	mockSender := &domain.Sender{Id: 1, Email: "test@example.com", Username: "testuser"}
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		id        int64
		want      *domain.Sender
		wantErr   error
	}{
		{
			name: "Success",
			repoSetup: &MockProfileRepository{
				FindSenderByIDFn: func(ctx context.Context, id int64) (*domain.Sender, error) {
					return mockSender, nil
				},
			},
			id:      1,
			want:    mockSender,
			wantErr: nil,
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				FindSenderByIDFn: func(ctx context.Context, id int64) (*domain.Sender, error) {
					return nil, mockError
				},
			},
			id:      1,
			want:    nil,
			wantErr: mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			got, err := uc.FindSenderByID(context.Background(), tt.id)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("FindSenderByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.Id != tt.want.Id || got.Email != tt.want.Email || got.Username != tt.want.Username {
				t.Errorf("FindSenderByID() got = %+v, want %+v", got, tt.want)
			}
		})
	}
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

func TestProfileUcase_FindInfoByID(t *testing.T) {
	mockInfo := domain.ProfileInfo{ID: 1}
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		profileID int64
		want      domain.ProfileInfo
		wantErr   error
	}{
		{
			name: "Success",
			repoSetup: &MockProfileRepository{
				FindInfoByIDFn: func(ctx context.Context, id int64) (domain.ProfileInfo, error) {
					return mockInfo, nil
				},
			},
			profileID: 1,
			want:      mockInfo,
			wantErr:   nil,
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				FindInfoByIDFn: func(ctx context.Context, id int64) (domain.ProfileInfo, error) {
					return domain.ProfileInfo{}, mockError
				},
			},
			profileID: 1,
			want:      domain.ProfileInfo{},
			wantErr:   mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			got, err := uc.FindInfoByID(context.Background(), tt.profileID)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("FindInfoByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.ID != tt.want.ID {
				t.Errorf("FindInfoByID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileUcase_UserExists(t *testing.T) {
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		username  string
		want      bool
		wantErr   error
	}{
		{
			name: "Success - user exists",
			repoSetup: &MockProfileRepository{
				UserExistsFn: func(ctx context.Context, username string) (bool, error) {
					return true, nil
				},
			},
			username: "existinguser",
			want:     true,
			wantErr:  nil,
		},
		{
			name: "Success - user doesn't exist",
			repoSetup: &MockProfileRepository{
				UserExistsFn: func(ctx context.Context, username string) (bool, error) {
					return false, nil
				},
			},
			username: "nonexistinguser",
			want:     false,
			wantErr:  nil,
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				UserExistsFn: func(ctx context.Context, username string) (bool, error) {
					return false, mockError
				},
			},
			username: "user",
			want:     false,
			wantErr:  mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			got, err := uc.UserExists(context.Background(), tt.username)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("UserExists() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UserExists() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileUcase_CreateUser(t *testing.T) {
	mockProfile := domain.Profile{ID: 1}
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		profile   domain.Profile
		want      int64
		wantErr   error
	}{
		{
			name: "Success",
			repoSetup: &MockProfileRepository{
				CreateUserFn: func(ctx context.Context, profile domain.Profile) (int64, error) {
					return 1, nil
				},
			},
			profile: mockProfile,
			want:    1,
			wantErr: nil,
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				CreateUserFn: func(ctx context.Context, profile domain.Profile) (int64, error) {
					return 0, mockError
				},
			},
			profile: mockProfile,
			want:    0,
			wantErr: mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			got, err := uc.CreateUser(context.Background(), tt.profile)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CreateUser() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileUcase_FindByUsernameAndDomain(t *testing.T) {
	mockProfile := &domain.Profile{ID: 1}
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		username  string
		domain    string
		want      *domain.Profile
		wantErr   error
	}{
		{
			name: "Success",
			repoSetup: &MockProfileRepository{
				FindByUsernameAndDomainFn: func(ctx context.Context, username string, domain string) (*domain.Profile, error) {
					return mockProfile, nil
				},
			},
			username: "testuser",
			domain:   "flintmail.ru",
			want:     mockProfile,
			wantErr:  nil,
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				FindByUsernameAndDomainFn: func(ctx context.Context, username string, domain string) (*domain.Profile, error) {
					return nil, mockError
				},
			},
			username: "testuser",
			domain:   "flintmail.ru",
			want:     nil,
			wantErr:  mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			got, err := uc.FindByUsernameAndDomain(context.Background(), tt.username, tt.domain)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("FindByUsernameAndDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FindByUsernameAndDomain() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileUcase_FindSettingsByProfileId(t *testing.T) {
	mockSettings := domain.Settings{ProfileID: 1}
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		profileID int64
		want      domain.Settings
		wantErr   error
	}{
		{
			name: "Success",
			repoSetup: &MockProfileRepository{
				FindSettingsByProfileIdFn: func(ctx context.Context, profileID int64) (domain.Settings, error) {
					return mockSettings, nil
				},
			},
			profileID: 1,
			want:      mockSettings,
			wantErr:   nil,
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				FindSettingsByProfileIdFn: func(ctx context.Context, profileID int64) (domain.Settings, error) {
					return domain.Settings{}, mockError
				},
			},
			profileID: 1,
			want:      domain.Settings{},
			wantErr:   mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			got, err := uc.FindSettingsByProfileId(context.Background(), tt.profileID)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("FindSettingsByProfileId() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.ProfileID != tt.want.ProfileID {
				t.Errorf("FindSettingsByProfileId() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileUcase_InsertProfileAvatar(t *testing.T) {
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		profileID int64
		avatarURL string
		wantErr   error
	}{
		{
			name: "Success",
			repoSetup: &MockProfileRepository{
				InsertProfileAvatarFn: func(ctx context.Context, profileID int64, avatarURL string) error {
					return nil
				},
			},
			profileID: 1,
			avatarURL: "http://example.com/avatar.jpg",
			wantErr:   nil,
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				InsertProfileAvatarFn: func(ctx context.Context, profileID int64, avatarURL string) error {
					return mockError
				},
			},
			profileID: 1,
			avatarURL: "http://example.com/avatar.jpg",
			wantErr:   mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			err := uc.InsertProfileAvatar(context.Background(), tt.profileID, tt.avatarURL)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("InsertProfileAvatar() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestProfileUcase_UpdateProfileInfo(t *testing.T) {
	mockError := errors.New("repository error")

	tests := []struct {
		name      string
		repoSetup *MockProfileRepository
		profileID int64
		req       UpdateProfileRequest
		wantErr   error
	}{
		{
			name: "Success - all fields",
			repoSetup: &MockProfileRepository{
				UpdateProfileInfoFn: func(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
					return nil
				},
			},
			profileID: 1,
			req: UpdateProfileRequest{
				FirstName:  "John",
				LastName:   "Doe",
				MiddleName: "Smith",
				Gender:     "male",
				Birthday:   "01.01.1990",
			},
			wantErr: nil,
		},
		{
			name: "Success - trimmed fields",
			repoSetup: &MockProfileRepository{
				UpdateProfileInfoFn: func(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
					return nil
				},
			},
			profileID: 1,
			req: UpdateProfileRequest{
				FirstName:  "  John  ",
				LastName:   "  Doe  ",
				MiddleName: "  Smith  ",
				Gender:     "  MALE  ",
				Birthday:   "  01.01.1990  ",
			},
			wantErr: nil,
		},
		{
			name: "Success - empty birthday",
			repoSetup: &MockProfileRepository{
				UpdateProfileInfoFn: func(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
					return nil
				},
			},
			profileID: 1,
			req: UpdateProfileRequest{
				FirstName: "John",
				LastName:  "Doe",
				Gender:    "male",
				Birthday:  "",
			},
			wantErr: nil,
		},
		{
			name: "Success - invalid gender becomes empty",
			repoSetup: &MockProfileRepository{
				UpdateProfileInfoFn: func(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
					return nil
				},
			},
			profileID: 1,
			req: UpdateProfileRequest{
				FirstName: "John",
				LastName:  "Doe",
				Gender:    "invalid",
				Birthday:  "01.01.1990",
			},
			wantErr: nil,
		},
		{
			name: "Failure - invalid birthday format",
			repoSetup: &MockProfileRepository{
				UpdateProfileInfoFn: func(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
					return nil
				},
			},
			profileID: 1,
			req: UpdateProfileRequest{
				FirstName: "John",
				LastName:  "Doe",
				Gender:    "male",
				Birthday:  "1990-01-01",
			},
			wantErr: errors.New("invalid birthday format"),
		},
		{
			name: "Repository error",
			repoSetup: &MockProfileRepository{
				UpdateProfileInfoFn: func(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
					return mockError
				},
			},
			profileID: 1,
			req: UpdateProfileRequest{
				FirstName: "John",
				LastName:  "Doe",
				Gender:    "male",
				Birthday:  "01.01.1990",
			},
			wantErr: mockError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &ProfileUcase{repo: tt.repoSetup}
			err := uc.UpdateProfileInfo(context.Background(), tt.profileID, tt.req)

			if (err != nil) != (tt.wantErr != nil) {
				t.Errorf("UpdateProfileInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr != nil && err != nil && tt.wantErr.Error() != err.Error() {
				t.Errorf("UpdateProfileInfo() error message = %v, wantErr %v", err.Error(), tt.wantErr.Error())
			}
		})
	}
}
