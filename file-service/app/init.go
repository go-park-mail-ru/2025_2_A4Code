package app

import (
	"2025_2_a4code/file-service/grpc_service"
	fb "2025_2_a4code/file-service/pkg/fileproto"
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/middleware/logger"
	in "2025_2_a4code/internal/lib/init"
	"2025_2_a4code/internal/lib/metrics"
	files_repository "2025_2_a4code/internal/storage/minio/file-repository"
	filemessage_repository "2025_2_a4code/internal/storage/postgres/filemessage-repository"
	"2025_2_a4code/internal/usecase/file"
	file_message "2025_2_a4code/internal/usecase/file-message"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/grpc"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func FileInit() {
	// Читаем конфиг
	cfg, err := config.GetConfig()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	var SECRET = []byte(cfg.AppConfig.Secret)

	// Создание логгера
	log := in.SetupLogger(envLocal)
	slog.SetDefault(log)
	log.Debug("profile: debug messages are enabled")

	go metrics.StartMetricsServer(cfg.AppConfig.FileMetricsPort, log)

	// Установка соединения с бд
	connection, err := in.NewDbConnection(cfg.DBConfig)
	if err != nil {
		log.Error("error connecting to database")
		os.Exit(1)
	}

	connection.SetMaxOpenConns(20)
	connection.SetMaxIdleConns(8)

	go metrics.MonitorDBConnections(connection)

	// Подключение MinIO
	client, err := newMinioConnection(cfg.MinioConfig.Endpoint, cfg.MinioConfig.User, cfg.MinioConfig.Password, cfg.MinioConfig.UseSSL)
	if err != nil {
		log.Error("error connecting to minio")
	}

	err = bucketExists(client, cfg.MinioConfig.FilesBucketName)
	if err != nil {
		log.Error("error checking bucket: " + err.Error())
	}

	// Создание репозиториев
	filesRepository := files_repository.New(client, cfg.MinioConfig.FilesBucketName, cfg.MinioConfig.PublicEndpoint, cfg.MinioConfig.PublicUseSSL)
	fileMessageRepository := filemessage_repository.New(connection)

	// Создание юзкейсов
	fileUsecase := file.New(filesRepository)
	fileMessageUsecase := file_message.New(fileMessageRepository)

	slog.Info("Starting server...", slog.String("address", cfg.AppConfig.Host+":"+cfg.AppConfig.ProfilePort))

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logger.GrpcLoggerInterceptor(log),
			metrics.MetricsInterceptor("file-service"),
		),
	)

	fileService := grpc_service.New(fileUsecase, fileMessageUsecase, SECRET)
	fb.RegisterFileServiceServer(grpcServer, fileService)

	lis, err := net.Listen("tcp", cfg.AppConfig.Host+":"+cfg.AppConfig.FilePort)
	if err != nil {
		log.Error("Failed to start server: " + err.Error())
		os.Exit(1)
	}

	log.Info("Files microservice: server has started working...")

	if err := grpcServer.Serve(lis); err != nil {
		log.Error("gRPC server failed: " + err.Error())
		os.Exit(1)
	}

}

func newMinioConnection(endpoint, accessKey, secretKey string, useSSL bool) (*minio.Client, error) {
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = minioClient.ListBuckets(ctx)
	if err != nil {
		slog.Warn("Could not connect to MinIO: " + err.Error())
	}

	return minioClient, nil
}

func bucketExists(client *minio.Client, bucketName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return err
	}

	if !exists {
		slog.Info("Creating bucket...", "bucket", bucketName)
		err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", bucketName, err)
		}
		slog.Info("Bucket created successfully", "bucket", bucketName)
	} else {
		slog.Info("Bucket already exists", "bucket", bucketName)
	}

	return nil
}
