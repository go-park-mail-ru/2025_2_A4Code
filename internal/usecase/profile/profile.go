package profile

import (
	"2025_2_a4code/internal/domain"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	FindByID(id int64) (*domain.Profile, error)
	FindSenderByID(id int64) (*domain.Sender, error)
	UserExists(ctx context.Context, username string) (bool, error)
	CreateUser(ctx context.Context, profile domain.Profile) (int64, error)
	FindByUsernameAndDomain(ctx context.Context, username string, domain string) (*domain.Profile, error)
	FindInfoByID(int64) (domain.ProfileInfo, error)
	FindSettingsById(profileID int64) (domain.Settings, error)
}

type ProfileUcase struct {
	repo ProfileRepository
}

func New(repo *profilerepository.ProfileRepository) *ProfileUcase {
	return &ProfileUcase{repo: repo}
}

func (uc *ProfileUcase) FindByID(id int64) (*domain.Profile, error) {
	return uc.repo.FindByID(int64(id))
}

func (uc *ProfileUcase) FindSenderByID(id int64) (*domain.Sender, error) {
	return uc.repo.FindSenderByID(int64(id))
}

func (uc *ProfileUcase) Signup(ctx context.Context, SignupReq SignupRequest) (int64, error) {
	// Проверка уникальности логина
	exists, err := uc.repo.UserExists(ctx, SignupReq.Username)
	if err != nil {
		return 0, err
	}
	if exists {
		return 0, errors.New("Пользователь с таким логином уже существует")
	}

	birthday, err := time.Parse("02.01.2006", SignupReq.Birthday)
	if err != nil {
		return 0, errors.New("Неверный формат даты. Используйте ДД.ММ.ГГГГ")
	}

	// Хэширование пароля
	PasswordHash, err := bcrypt.GenerateFromPassword([]byte(SignupReq.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	// Создаем domain модель с хэшированным паролем
	profile := domain.Profile{
		Name:         SignupReq.Name,
		Username:     SignupReq.Username,
		Birthday:     birthday,
		Gender:       SignupReq.Gender,
		PasswordHash: string(PasswordHash),
	}

	// Сохранение в БД через репозиторий
	userID, err := uc.repo.CreateUser(ctx, profile)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (uc *ProfileUcase) Login(ctx context.Context, req LoginRequest) (int64, error) {
	// Ищем профиль по username и фиксированному домену
	profile, err := uc.repo.FindByUsernameAndDomain(ctx, req.Username, "a4mail.ru")
	if err != nil {
		return 0, errors.New("Пользователь с таким адресом почты отсутствует")
	}

	// Проверяем пароль
	if !uc.checkPassword(req.Password, profile.PasswordHash) {
		return 0, errors.New("Неверный пароль")
	}

	return profile.ID, nil
}

func (uc *ProfileUcase) checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (uc *ProfileUcase) FindInfoByID(profileID int64) (domain.ProfileInfo, error) {
	return uc.repo.FindInfoByID(profileID)
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

func (uc *ProfileUcase) FindSettingsById(profileID int64) (domain.Settings, error) {
	return uc.repo.FindSettingsById(profileID)
}
