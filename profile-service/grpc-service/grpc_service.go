package profile_service

import (
	"2025_2_a4code/internal/lib/metrics"
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	commonE "2025_2_a4code/internal/lib/errors"
	"2025_2_a4code/internal/lib/session"
	"2025_2_a4code/internal/usecase/profile"
	pb "2025_2_a4code/profile-service/pkg/profileproto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedProfileServiceServer
	profileUCase ProfileUsecase
	avatarUCase  AvatarUsecase
	JWTSecret    []byte
}

type ProfileUsecase interface {
	FindInfoByID(ctx context.Context, id int64) (domain.ProfileInfo, error)
	UpdateProfileInfo(ctx context.Context, profileID int64, req profile.UpdateProfileRequest) error
	FindSettingsByProfileId(ctx context.Context, profileID int64) (domain.Settings, error)
	InsertProfileAvatar(ctx context.Context, profileID int64, avatarURL string) error
}

type AvatarUsecase interface {
	GetAvatarPresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error)
	UploadAvatar(ctx context.Context, userID string, file io.Reader, size int64, originalFilename string) (string, string, error)
}

// Максимальный размер загружаемого аватара - 5 Мб
const maxAvatarSize = 5 << 20

func New(profileUCase ProfileUsecase, avatarUCase AvatarUsecase, secret []byte) *Server {
	return &Server{
		profileUCase: profileUCase,
		avatarUCase:  avatarUCase,
		JWTSecret:    secret,
	}
}

func (s *Server) getProfileID(ctx context.Context) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	tokens := md.Get("authorization")
	if len(tokens) == 0 {
		return 0, status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	tokenString := strings.TrimPrefix(tokens[0], "Bearer ")
	return session.GetProfileIDFromTokenString(tokenString, s.JWTSecret, "access")
}

func (s *Server) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	const op = "profileservice.GetProfile"
	log := logger.GetLogger(ctx)
	log.Debug("handle user/profile (GET)")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	profileInfo, err := s.profileUCase.FindInfoByID(ctx, profileID)
	if err != nil {
		log.Error(op + ": failed to get profile: " + err.Error())
		if errors.Is(err, commonE.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "profile not found")
		}
		return nil, status.Error(codes.Internal, "could not get profile")
	}

	if err := s.enrichAvatarURL(ctx, &profileInfo); err != nil {
		log.Warn("failed to enrich avatar url: " + err.Error())
	}

	return &pb.GetProfileResponse{
		Profile: s.domainProfileToProto(profileInfo),
	}, nil
}

func (s *Server) UpdateProfile(ctx context.Context, req *pb.UpdateProfileRequest) (*pb.UpdateProfileResponse, error) {
	const op = "profileservice.UpdateProfile"
	log := logger.GetLogger(ctx)
	log.Info("handle user/profile (PUT)")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	updateReq := profile.UpdateProfileRequest{
		FirstName:  req.Name,
		LastName:   req.Surname,
		MiddleName: req.Patronymic,
		Gender:     req.Gender,
		Birthday:   req.Birthday,
	}

	if err := s.profileUCase.UpdateProfileInfo(ctx, profileID, updateReq); err != nil {
		log.Error(op + ": failed to update profile: " + err.Error())
		return nil, status.Error(codes.InvalidArgument, "could not update profile")
	}

	profileInfo, err := s.profileUCase.FindInfoByID(ctx, profileID)
	if err != nil {
		log.Error(op + ": failed to get updated profile: " + err.Error())
		if errors.Is(err, commonE.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "profile not found")
		}
		return nil, status.Error(codes.Internal, "could not get updated profile")
	}

	if err := s.enrichAvatarURL(ctx, &profileInfo); err != nil {
		log.Warn("failed to enrich avatar url: " + err.Error())
	}

	return &pb.UpdateProfileResponse{
		Profile: s.domainProfileToProto(profileInfo),
	}, nil
}

func (s *Server) Settings(ctx context.Context, req *pb.SettingsRequest) (*pb.SettingsResponse, error) {
	const op = "profileservice.Settings"
	log := logger.GetLogger(ctx)
	log.Debug("handle user/settings")

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	settings, err := s.profileUCase.FindSettingsByProfileId(ctx, profileID)
	if err != nil {
		log.Error(op + ": failed to get settings: " + err.Error())
		return nil, status.Error(codes.Internal, "could not get settings")
	}

	return &pb.SettingsResponse{
		Settings: s.domainSettingsToProto(settings),
	}, nil
}

func (s *Server) UploadAvatar(ctx context.Context, req *pb.UploadAvatarRequest) (*pb.UploadAvatarResponse, error) {
	const op = "profileservice.UploadAvatar"
	log := logger.GetLogger(ctx)
	log.Debug("Handling user/upload/avatar")

	// Будем отслеживать статус
	opStatus := "error"
	defer func() {
		metrics.AvatarOperations.WithLabelValues("profile-service", op, opStatus).Inc()
	}()

	profileID, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Проверка размера файла
	if len(req.AvatarData) > maxAvatarSize {
		return nil, status.Error(codes.InvalidArgument, "file too large")
	}

	metrics.FileSize.WithLabelValues("profile-service", "avatar").Observe(float64(len(req.AvatarData)))
	// Создаем reader из байтов
	fileReader := bytes.NewReader(req.AvatarData)

	stringID := strconv.FormatInt(profileID, 10)
	objectName, presignedURL, err := s.avatarUCase.UploadAvatar(ctx, stringID, fileReader, int64(len(req.AvatarData)), req.FileName)
	if err != nil {
		log.Error(op + ": failed to upload avatar: " + err.Error())
		return nil, status.Error(codes.Internal, "could not upload avatar")
	}

	err = s.profileUCase.InsertProfileAvatar(ctx, profileID, objectName)
	if err != nil {
		log.Error(op + ": failed to insert avatar: " + err.Error())
		return nil, status.Error(codes.Internal, "could not save avatar")
	}

	opStatus = "success"
	return &pb.UploadAvatarResponse{
		AvatarPath: presignedURL,
	}, nil
}

func (s *Server) domainProfileToProto(profileInfo domain.ProfileInfo) *pb.Profile {
	return &pb.Profile{
		Id:         strconv.FormatInt(profileInfo.ID, 10),
		Username:   profileInfo.Username,
		CreatedAt:  profileInfo.CreatedAt.Format(time.RFC3339),
		Name:       profileInfo.Name,
		Surname:    profileInfo.Surname,
		Patronymic: profileInfo.Patronymic,
		Gender:     profileInfo.Gender,
		Birthday:   profileInfo.Birthday,
		AvatarPath: profileInfo.AvatarPath,
	}
}

func (s *Server) domainSettingsToProto(settings domain.Settings) *pb.Settings {
	return &pb.Settings{
		NotificationTolerance: settings.NotificationTolerance,
		Language:              settings.Language,
		Theme:                 settings.Theme,
		Signatures:            settings.Signatures,
	}
}

func (s *Server) enrichAvatarURL(ctx context.Context, profileInfo *domain.ProfileInfo) error {
	if profileInfo.AvatarPath == "" {
		return nil
	}

	objectName := profileInfo.AvatarPath
	if strings.HasPrefix(profileInfo.AvatarPath, "http://") || strings.HasPrefix(profileInfo.AvatarPath, "https://") {
		parsed, err := url.Parse(profileInfo.AvatarPath)
		if err != nil {
			return err
		}
		objectName = strings.TrimPrefix(parsed.Path, "/")
	}

	objectName = strings.TrimLeft(objectName, "/")
	if objectName == "" {
		return nil
	}

	if idx := strings.Index(objectName, "/"); idx != -1 {
		prefix := objectName[:idx]
		if strings.EqualFold(prefix, "avatars") {
			objectName = objectName[idx+1:]
		}
	}

	if objectName == "" {
		return nil
	}

	presignedURL, err := s.avatarUCase.GetAvatarPresignedURL(ctx, objectName, 15*time.Minute)
	if err != nil {
		return err
	}

	profileInfo.AvatarPath = presignedURL.String()

	if !strings.HasPrefix(profileInfo.AvatarPath, "http") {
		profileInfo.AvatarPath = presignedURL.String()
	}

	return nil
}
