package profile

import (
	"2025_2_a4code/internal/domain"
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type SignupRequest struct {
	Name     string
	Username string
	Birthday time.Time
	Gender   string
	Password string
}

type LoginRequest struct {
	Email    string
	Password string
}

type ProfileRepository interface {
	FindByID(id int64) (*domain.Profile, error)
	FindSenderByID(id int64) (*domain.Sender, error)
	UserExists(ctx context.Context, username string) (bool, error)
	CreateUser(ctx context.Context, profile domain.Profile) (int64, error)
	FindByUsernameAndDomain(ctx context.Context, username string, domain string) (*domain.Profile, error)
}

type ProfileUcase struct {
	repo ProfileRepository
}

func New(repo ProfileRepository) *ProfileUcase {
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

	// Хэширование пароля
	PasswordHash, err := bcrypt.GenerateFromPassword([]byte(SignupReq.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	// Создаем domain модель с хэшированным паролем
	profile := domain.Profile{
		Name:         SignupReq.Name,
		Username:     SignupReq.Username, // Исправлено: было Userame
		Birthday:     SignupReq.Birthday,
		Gender:       SignupReq.Gender,
		PasswordHash: string(PasswordHash), // Сохраняем хэш
	}

	// Сохранение в БД через репозиторий
	userID, err := uc.repo.CreateUser(ctx, profile)
	if err != nil {
		return 0, err
	}

	return userID, nil
}

func (uc *ProfileUcase) Login(ctx context.Context, req LoginRequest) (int64, error) {
	// Парсим email для извлечения username
	username, err := uc.ExtractUsernameFromEmail(req.Email)
	if err != nil {
		return 0, err
	}

	// Ищем профиль по username и фиксированному домену
	profile, err := uc.repo.FindByUsernameAndDomain(ctx, username, "a4mail.ru")
	if err != nil {
		return 0, errors.New("Пользователь с таким адресом почты отсутствует")
	}

	// Проверяем пароль
	if !uc.checkPassword(req.Password, profile.PasswordHash) {
		return 0, errors.New("Неверный пароль")
	}

	return profile.ID, nil
}

func (uc *ProfileUcase) ExtractUsernameFromEmail(email string) (string, error) {
	// Убеждаемся, что email заканчивается на наш домен
	if !strings.HasSuffix(email, "@a4mail.ru") {
		return "", errors.New("Адрес должен оканчиваться на @a4mail.ru")
	}

	// Извлекаем username (всё до @)
	username := strings.TrimSuffix(email, "@a4mail.ru")

	// Дополнительная валидация username
	if username == "" {
		return "", errors.New("Неправильный формат адреса почты")
	}

	return username, nil
}

func (uc *ProfileUcase) checkPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
