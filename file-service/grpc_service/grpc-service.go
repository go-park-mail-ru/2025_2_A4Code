package grpc_service

import (
	fb "2025_2_a4code/file-service/pkg/fileproto"
	"2025_2_a4code/internal/http-server/middleware/logger"
	"2025_2_a4code/internal/lib/metrics"
	"2025_2_a4code/internal/lib/session"
	"bytes"
	"context"
	"io"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Максимальный размер загружаемого файла - 5 Мб
const maxFileSize = 5 << 20

type Server struct {
	fb.UnimplementedFilesServiceServer
	fileUsecase        FileUsecase
	fileMessageUsecase FileMessageUsecase
	JWTSecret          []byte
}

type FileUsecase interface {
	UploadFileMain(ctx context.Context, messageID string, file io.Reader, size int64, originalFilename string) (string, string, error)
	GetFilePresignedURL(ctx context.Context, objectName string, duration time.Duration) (*url.URL, error)
	DeleteFile(ctx context.Context, objectName string) error
}

type FileMessageUsecase interface {
	DeleteFile(ctx context.Context, fileId int64) error
	InsertFile(ctx context.Context, messageId string, size int64, fileType, storagePath string) error
}

func New(fileUsecase FileUsecase, fileMessageUsecase FileMessageUsecase, secret []byte) *Server {
	return &Server{
		fileUsecase:        fileUsecase,
		fileMessageUsecase: fileMessageUsecase,
		JWTSecret:          secret,
	}
}

func (s *Server) UploadFile(ctx context.Context, req *fb.UploadFileRequest) (*fb.UploadFileResponse, error) {
	const op = "profileservice.UploadFile"
	log := logger.GetLogger(ctx)
	log.Debug("Handling /upload/file")

	opStatus := "error"
	defer func() {
		metrics.AvatarOperations.WithLabelValues("file-service", op, opStatus).Inc()
	}()

	_, err := s.getProfileID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	// Проверка размера файла
	if len(req.File.FileData) > maxFileSize {
		return nil, status.Error(codes.InvalidArgument, "file too large")
	}

	metrics.FileSize.WithLabelValues("file-service", "file").Observe(float64(len(req.File.FileData)))
	// Создаем reader из байтов
	fileReader := bytes.NewReader(req.File.FileData)

	_, presignedURL, err := s.fileUsecase.UploadFileMain(ctx, req.MessageId, fileReader, int64(len(req.File.FileData)), req.File.FileName)
	if err != nil {
		log.Error(op + ": failed to upload avatar: " + err.Error())
		return nil, status.Error(codes.Internal, "could not upload avatar")
	}

	err = s.fileMessageUsecase.InsertFile(ctx, req.MessageId, int64(len(req.File.FileData)), filepath.Ext(presignedURL), presignedURL)
	if err != nil {
		log.Error(op + ": failed to insert file: " + err.Error())
		return nil, status.Error(codes.Internal, "could not save avatar")
	}

	opStatus = "success"
	return &fb.UploadFileResponse{
		FileData: req.File.FileData,
		FilePath: presignedURL,
	}, nil
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
