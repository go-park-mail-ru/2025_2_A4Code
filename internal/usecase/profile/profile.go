package profile

import (
	"2025_2_a4code/internal/domain"
	common_e "2025_2_a4code/internal/lib/errors"
	e "2025_2_a4code/internal/lib/wrapper"
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidDateFormat  = errors.New("invalid date format")
	ErrUserNotFound       = errors.New("user not found")
	ErrWrongPassword      = errors.New("wrong password")
	ErrPasswordHashFailed = errors.New("password hash failed")
	ErrUserCreationFailed = errors.New("user creation failed")
)

type SignupRequest struct {
	Name     string
	Username string
	Birthday string
	Gender   string
	Password string
}

type LoginRequest struct {
	Username string
	Password string
}

type ProfileRepository interface {
	FindByID(ctx context.Context, id int64) (*domain.Profile, error)
	FindSenderByID(ctx context.Context, id int64) (*domain.Sender, error)
	UserExists(ctx context.Context, username string) (bool, error)
	CreateUser(ctx context.Context, profile domain.Profile) (int64, error)
	FindByUsernameAndDomain(ctx context.Context, username string, domain string) (*domain.Profile, error)
	FindInfoByID(ctx context.Context, id int64) (domain.ProfileInfo, error)
	FindSettingsByProfileId(ctx context.Context, profileID int64) (domain.Settings, error)
}

type ProfileUcase struct {
	repo ProfileRepository
}

func New(repo ProfileRepository) *ProfileUcase {
	return &ProfileUcase{repo: repo}
}

func (uc *ProfileUcase) FindByID(ctx context.Context, id int64) (*domain.Profile, error) {
	return uc.repo.FindByID(ctx, id)
}

func (uc *ProfileUcase) FindSenderByID(ctx context.Context, id int64) (*domain.Sender, error) {
	return uc.repo.FindSenderByID(ctx, id)
}

func (uc *ProfileUcase) Signup(ctx context.Context, SignupReq SignupRequest) (int64, error) {
	const op = "usecase.profile.Signup"

	exists, err := uc.repo.UserExists(ctx, SignupReq.Username)
	if err != nil {
		return 0, e.Wrap(op, err)
	}
	if exists {
		return 0, e.Wrap(op, ErrUserAlreadyExists)
	}

	birthday, err := time.Parse("02.01.2006", SignupReq.Birthday)
	if err != nil {
		return 0, e.Wrap(op+"parsing data:", err)
	}

	PasswordHash, err := bcrypt.GenerateFromPassword([]byte(SignupReq.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, e.Wrap(op, ErrPasswordHashFailed)
	}

	profile := domain.Profile{
		Name:         SignupReq.Name,
		Username:     SignupReq.Username,
		Birthday:     birthday,
		Gender:       SignupReq.Gender,
		PasswordHash: string(PasswordHash),
	}

	userId, err := uc.repo.CreateUser(ctx, profile)
	if err != nil {
		return 0, e.Wrap(op, ErrUserCreationFailed)
	}

	return userId, nil
}

func (uc *ProfileUcase) Login(ctx context.Context, req LoginRequest) (int64, error) {
	const op = "usecase.profile.Login"
	profile, err := uc.repo.FindByUsernameAndDomain(ctx, req.Username, "flintmail.ru")
	if err != nil {
		if errors.Is(err, common_e.ErrNotFound) {
			return 0, e.Wrap(op, ErrUserNotFound)
		}
		return 0, e.Wrap(op, err)
	}

	if !uc.checkPassword(req.Password, profile.PasswordHash) {
		return 0, e.Wrap(op, ErrWrongPassword)
	}

	return profile.ID, nil
}

func (uc *ProfileUcase) checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (uc *ProfileUcase) FindInfoByID(ctx context.Context, profileID int64) (domain.ProfileInfo, error) {
	return uc.repo.FindInfoByID(ctx, profileID)
}

func (uc *ProfileUcase) UserExists(ctx context.Context, username string) (bool, error) {
	return uc.repo.UserExists(ctx, username)
}

func (uc *ProfileUcase) CreateUser(ctx context.Context, profile domain.Profile) (int64, error) {
	return uc.repo.CreateUser(ctx, profile)
}

func (uc *ProfileUcase) FindByUsernameAndDomain(ctx context.Context, username string, domain string) (*domain.Profile, error) {
	return uc.repo.FindByUsernameAndDomain(ctx, username, domain)
}

func (uc *ProfileUcase) FindSettingsByProfileId(ctx context.Context, profileID int64) (domain.Settings, error) {
	return uc.repo.FindSettingsByProfileId(ctx, profileID)
}
