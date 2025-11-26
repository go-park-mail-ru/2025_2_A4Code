package auth_service

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	"2025_2_a4code/internal/lib/metrics"
	"2025_2_a4code/internal/lib/session"
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	pb "2025_2_a4code/auth-service/pkg/authproto"
	"2025_2_a4code/internal/usecase/profile"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedAuthServiceServer
	profileUCase ProfileUsecase
	JWTSecret    []byte
}

type ProfileUsecase interface {
	Login(ctx context.Context, req profile.LoginRequest) (int64, error)
	Signup(ctx context.Context, SignupReq profile.SignupRequest) (int64, error)
}

func New(profileUCase ProfileUsecase, secret []byte) *Server {
	return &Server{
		profileUCase: profileUCase,
		JWTSecret:    secret,
	}
}

func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	const op = "authservice.Login"
	log := logger.GetLogger(ctx)
	log.Debug("handle /auth/login")

	// TODO: validation
	if req.Login == "" || req.Password == "" {
		metrics.AuthLoginAttempts.WithLabelValues("error").Inc()
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "login", "validation_error").Inc()
		return nil, status.Error(codes.InvalidArgument, "login and password are required")
	}

	req.Login = strings.TrimSpace(req.Login)
	req.Password = strings.TrimSpace(req.Password)
	loginReq := profile.LoginRequest{
		Username: req.Login,
		Password: req.Password,
	}
	userID, err := s.profileUCase.Login(ctx, loginReq)
	if err != nil {
		log.Debug(op + ": login failed: " + err.Error())
		metrics.AuthLoginAttempts.WithLabelValues("error").Inc()
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "login", "auth_failed").Inc()
		return nil, status.Error(codes.Unauthenticated, "invalid login or password")
	}

	accToken, refToken, err := s.generateTokenPair(userID)
	if err != nil {
		log.Error(op + ": failed to generate token pair: " + err.Error())
		metrics.AuthLoginAttempts.WithLabelValues("error").Inc()
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "login", "token_generation_error").Inc()
		return nil, status.Error(codes.Internal, "could not process login")
	}

	metrics.AuthLoginAttempts.WithLabelValues("success").Inc()
	return &pb.LoginResponse{
		AccessToken:  accToken,
		RefreshToken: refToken,
	}, nil
}

func (s *Server) Signup(ctx context.Context, req *pb.SignupRequest) (*pb.SignupResponse, error) {
	const op = "authservice.Signup"
	log := logger.GetLogger(ctx)
	log.Debug("handle /auth/signup")

	// TODO: validation
	if req.Username == "" || req.Password == "" {
		metrics.AuthSignupAttempts.WithLabelValues("error").Inc()
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "signup", "validation_error").Inc()
		return nil, status.Error(codes.InvalidArgument, "all fields are required")
	}

	signupReq := profile.SignupRequest{
		Name:     req.Name,
		Username: req.Username,
		Birthday: req.Birthday,
		Gender:   req.Gender,
		Password: req.Password,
	}

	userID, err := s.profileUCase.Signup(ctx, signupReq)
	if err != nil {
		log.Warn(op + ": signup failed: " + err.Error())
		metrics.AuthSignupAttempts.WithLabelValues("error").Inc()
		if errors.Is(err, profile.ErrUserAlreadyExists) {
			metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "signup", "user_exists").Inc()
			return nil, status.Error(codes.AlreadyExists, "user with this username already exists")
		}
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "signup", "internal_error").Inc()
		return nil, status.Error(codes.Internal, "could not process signup")
	}

	accToken, refToken, err := s.generateTokenPair(userID)
	if err != nil {
		log.Error(op + ": failed to generate token pair: " + err.Error())
		metrics.AuthSignupAttempts.WithLabelValues("error").Inc()
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "signup", "token_generation_error").Inc()
		return nil, status.Error(codes.Internal, "could not process signup")
	}

	metrics.AuthSignupAttempts.WithLabelValues("success").Inc()
	return &pb.SignupResponse{
		AccessToken:  accToken,
		RefreshToken: refToken,
	}, nil
}

func (s *Server) Refresh(ctx context.Context, req *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	const op = "authservice.Refresh"
	log := logger.GetLogger(ctx)
	log.Debug("handle /auth/refresh")

	refreshToken := req.RefreshToken
	if refreshToken == "" {
		metrics.AuthTokenRefreshes.WithLabelValues("error").Inc()
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "refresh", "empty_token").Inc()
		return nil, status.Error(codes.Unauthenticated, "refresh token is required")
	}

	validationStart := time.Now()
	userID, err := session.GetProfileIDFromTokenString(refreshToken, s.JWTSecret, "refresh")
	validationDuration := time.Since(validationStart).Seconds()

	metrics.TokenValidationDuration.WithLabelValues("refresh").Observe(validationDuration)

	if err != nil {
		log.Debug(op + ": invalid refresh token: " + err.Error())
		metrics.AuthTokenRefreshes.WithLabelValues("error").Inc()

		errorType := "invalid"
		if err.Error() == "token is expired" {
			errorType = "expired"
		} else if strings.Contains(err.Error(), "signature") {
			errorType = "signature"
		}
		metrics.JWTValidationErrors.WithLabelValues("refresh", errorType).Inc()

		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "refresh", "invalid_token").Inc()
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	newAccessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    "access",
	})

	newAccessTokenString, err := newAccessToken.SignedString(s.JWTSecret)
	if err != nil {
		log.Error("failed to sign new access token", slog.String("error", err.Error()))
		metrics.AuthTokenRefreshes.WithLabelValues("error").Inc()
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "refresh", "token_signing_error").Inc()
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	metrics.AuthTokenRefreshes.WithLabelValues("success").Inc()
	return &pb.RefreshResponse{
		AccessToken: newAccessTokenString,
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	const op = "authservice.Logout"
	log := logger.GetLogger(ctx)
	log.Debug("handle /auth/logout")

	metrics.AuthLogoutsTotal.WithLabelValues("success").Inc()

	return &pb.LogoutResponse{}, nil
}

func (s *Server) generateAccessToken(userID int64) (string, error) {
	start := time.Now()

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    "access",
	})

	tokenString, err := accessToken.SignedString(s.JWTSecret)
	duration := time.Since(start).Seconds()

	if err != nil {
		metrics.TokenGenerationErrors.WithLabelValues("access").Inc()
	} else {
		metrics.TokenGenerations.WithLabelValues("access", "success").Inc()
	}
	metrics.TokenGenerationDuration.WithLabelValues("access").Observe(duration)

	return tokenString, err
}

func (s *Server) generateTokenPair(userID int64) (string, string, error) {
	start := time.Now()

	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		metrics.TokenGenerations.WithLabelValues("pair", "error").Inc()
		return "", "", err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type":    "refresh",
	})

	refreshTokenString, err := refreshToken.SignedString(s.JWTSecret)
	if err != nil {
		metrics.TokenGenerations.WithLabelValues("pair", "error").Inc()
		return "", "", err
	}

	duration := time.Since(start).Seconds()

	metrics.TokenGenerations.WithLabelValues("pair", "success").Inc()
	metrics.TokenGenerationDuration.WithLabelValues("pair").Observe(duration)

	return accessToken, refreshTokenString, nil
}
