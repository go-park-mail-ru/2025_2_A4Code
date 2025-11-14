package auth_service

import (
	"2025_2_a4code/internal/http-server/middleware/logger"
	"2025_2_a4code/internal/lib/session"
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"2025_2_a4code/internal/usecase/profile"
	pb "2025_2_a4code/pkg/authproto"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
		return nil, status.Error(codes.Unauthenticated, "invalid login or password")
	}

	accToken, refToken, err := s.generateTokenPair(userID)
	if err != nil {
		log.Error(op + ": failed to generate token pair: " + err.Error())
		return nil, status.Error(codes.Internal, "could not process login")
	}

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
		if errors.Is(err, profile.ErrUserAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "user with this username already exists")
		}
		return nil, status.Error(codes.Internal, "could not process signup")
	}

	accToken, refToken, err := s.generateTokenPair(userID)
	if err != nil {
		log.Error(op+": failed to generate token pair: ", err.Error())
		return nil, status.Error(codes.Internal, "could not process signup")
	}

	return &pb.SignupResponse{
		AccessToken:  accToken,
		RefreshToken: refToken,
	}, nil
}

func (s *Server) Refresh(ctx context.Context, req *pb.RefreshRequest) (*pb.RefreshResponse, error) {
	const op = "authservice.Refresh"
	log := logger.GetLogger(ctx)
	log.Debug("handle /auth/refresh")

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		log.Error(op + ": could not get metadata")
		return nil, status.Error(codes.Unauthenticated, "could not process refresh")
	}

	tokens := md.Get("refresh_token")
	if len(tokens) == 0 {
		return nil, status.Error(codes.Unauthenticated, "could not process refresh")
	}
	refreshToken := tokens[0]

	userID, err := session.GetProfileIDFromTokenString(refreshToken, s.JWTSecret, "refresh")
	if err != nil {
		log.Debug(op + ": invalid refresh token: " + err.Error())
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
		return nil, status.Error(codes.Internal, "failed to generate access token")
	}

	return &pb.RefreshResponse{
		AccessToken: newAccessTokenString,
	}, nil
}

func (s *Server) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	return &pb.LogoutResponse{}, nil
}

func (s *Server) generateAccessToken(userID int64) (string, error) {
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(15 * time.Minute).Unix(),
		"type":    "access",
	})
	return accessToken.SignedString(s.JWTSecret)
}

func (s *Server) generateTokenPair(userID int64) (string, string, error) {
	accessToken, err := s.generateAccessToken(userID)
	if err != nil {
		return "", "", err
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(7 * 24 * time.Hour).Unix(),
		"type":    "refresh",
	})

	refreshTokenString, err := refreshToken.SignedString(s.JWTSecret)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshTokenString, nil
}
