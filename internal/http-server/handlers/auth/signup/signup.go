package signup

//go:generate mockgen -source=$GOFILE -destination=./mocks/mock_profile_usecase.go -package=mocks

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	resp "2025_2_a4code/internal/lib/api/response"
	"context"

	valid "2025_2_a4code/internal/lib/validation"
	"2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/golang-jwt/jwt/v5"
)

type Request struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Birthday string `json:"birthday"`
	Gender   string `json:"gender"`
	Password string `json:"password"`
}

type ProfileUsecase interface {
	Signup(ctx context.Context, SignupReq profile.SignupRequest) (int64, error)
}
type Response struct {
	resp.Response
}

type HandlerSignup struct {
	profileUCase ProfileUsecase
	JWTSecret    []byte
}

func New(profileUCase ProfileUsecase, secret []byte) *HandlerSignup {
	return &HandlerSignup{
		profileUCase: profileUCase,
		JWTSecret:    secret,
	}
}

func (h *HandlerSignup) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log := logger.GetLogger(r.Context())
	log.Debug("handle /auth/signup")

	if r.Method != http.MethodPost {
		resp.SendErrorResponse(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp.SendErrorResponse(w, "invalid request format", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Username = strings.TrimSpace(req.Username)
	req.Birthday = strings.TrimSpace(req.Birthday)
	req.Gender = strings.TrimSpace(req.Gender)
	req.Password = strings.TrimSpace(req.Password)

	if err := h.validateRequest(&req); err != nil {
		resp.SendErrorResponse(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Преобразуем в UseCase запрос
	SignupReq := profile.SignupRequest{
		Name:     req.Name,
		Username: req.Username,
		Birthday: req.Birthday,
		Gender:   req.Gender,
		Password: req.Password,
	}

	userID, err := h.profileUCase.Signup(r.Context(), SignupReq)
	if err != nil {
		log.Warn("signup failed",
			slog.String("username", req.Username),
			slog.String("error", err.Error()))

		switch {
		case errors.Is(err, profile.ErrUserAlreadyExists):
			resp.SendErrorResponse(w, "user with this username already exists", http.StatusBadRequest)
		default:
			log.Error("unexpected signup error",
				slog.String("error", err.Error()),
				slog.String("username", req.Username))
			resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		}
		return
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(), // 15 минут
		"type":    "access",
	})

	accessTokenString, err := accessToken.SignedString(h.JWTSecret)
	if err != nil {
		log.Error("failed to sign access token")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(), // 7 дней
		"type":    "refresh",
	})

	refreshTokenString, err := refreshToken.SignedString(h.JWTSecret)
	if err != nil {
		log.Error("failed to sign resfresh token")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    accessTokenString,
		MaxAge:   15 * 60, // 15 минут
		HttpOnly: true,
		Path:     "/",
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, accessCookie)

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshTokenString,
		MaxAge:   7 * 24 * 3600, // 7  дней
		HttpOnly: true,
		Path:     "/",
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, refreshCookie)

	w.Header().Set("Content-Type", "application/json")
	response := Response{
		Response: resp.Response{
			Status:  http.StatusOK,
			Message: "success",
			Body:    struct{}{},
		},
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error("failed to encode response")
		resp.SendErrorResponse(w, "something went wrong", http.StatusInternalServerError)
		return
	}
}

func (h *HandlerSignup) validateRequest(req *Request) error {
	if req.Username == "" || req.Password == "" || req.Name == "" || req.Gender == "" || req.Birthday == "" {
		return fmt.Errorf("all fields are required")
	}

	// Валидация имени
	if len(req.Name) < 2 || len(req.Name) > 100 {
		return fmt.Errorf("name must be between 2 and 100 characters")
	}
	for _, char := range req.Name {
		if !unicode.IsLetter(char) && char != ' ' && char != '-' {
			return fmt.Errorf("name can only contain letters, spaces and hyphens")
		}
	}

	if valid.HasDangerousCharacters(req.Name) {
		return fmt.Errorf("name contains invalid characters")
	}

	// Валидация username
	if len(req.Username) < 3 || len(req.Username) > 50 {
		return fmt.Errorf("username must be between 3 and 50 characters")
	}
	for _, char := range req.Username {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return fmt.Errorf("username can only contain letters, numbers and underscores")
		}
	}

	if valid.HasDangerousCharacters(req.Username) {
		return fmt.Errorf("username contains invalid characters")
	}

	// Валдация даты
	if len(req.Birthday) != 10 {
		return fmt.Errorf("birthday must be in DD.MM.YYYY format")
	}
	if req.Birthday[2] != '.' || req.Birthday[5] != '.' {
		return fmt.Errorf("birthday must be in DD.MM.YYYY format")
	}
	for i, char := range req.Birthday {
		if i != 2 && i != 5 {
			if char < '0' || char > '9' {
				return fmt.Errorf("birthday must contain only numbers and dots")
			}
		}
	}

	dayStr := req.Birthday[0:2]
	monthStr := req.Birthday[3:5]
	yearStr := req.Birthday[6:10]

	day, err1 := strconv.Atoi(dayStr)
	month, err2 := strconv.Atoi(monthStr)
	year, err3 := strconv.Atoi(yearStr)

	if err1 != nil || err2 != nil || err3 != nil {
		return fmt.Errorf("invalid birthday format")
	}

	if month < 1 || month > 12 {
		return fmt.Errorf("birthday month must be between 01 and 12")
	}

	daysInMonth := 31
	switch month {
	case 4, 6, 9, 11:
		daysInMonth = 30
	case 2:
		if (year%4 == 0 && year%100 != 0) || (year%400 == 0) {
			daysInMonth = 29
		} else {
			daysInMonth = 28
		}
	}

	if day < 1 || day > daysInMonth {
		return fmt.Errorf("birthday day is out of range for the month")
	}

	now := time.Now()
	inputDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if inputDate.After(now) {
		return fmt.Errorf("birthday must not be in the future")
	}

	// Валидация пола
	gender := strings.ToLower(req.Gender)
	if gender != "male" && gender != "female" {
		return fmt.Errorf("gender must be male or female")
	}

	// Валидация пароля
	if len(req.Password) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}
	hasLetter, hasDigit := false, false
	for _, char := range req.Password {
		if unicode.IsLetter(char) {
			hasLetter = true
		}
		if unicode.IsDigit(char) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("password must contain both letters and numbers")
	}

	if strings.ContainsAny(req.Password, " \t\n\r") {
		return fmt.Errorf("password must not contain spaces")
	}

	if valid.HasDangerousCharacters(req.Password) {
		return fmt.Errorf("password contains invalid characters")
	}

	return nil
}
