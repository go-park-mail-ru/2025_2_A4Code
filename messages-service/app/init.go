package app

import (
	"2025_2_a4code/internal/config"
	in "2025_2_a4code/internal/lib/init"
	messagesservice "2025_2_a4code/messages-service"
	"net"

	// "2025_2_a4code/internal/http-server/handlers/messages/threads"
	// uploadfile "2025_2_a4code/internal/http-server/handlers/user/upload/upload-file"

	avatarrepository "2025_2_a4code/internal/storage/minio/avatar-repository"
	messagerepository "2025_2_a4code/internal/storage/postgres/message-repository"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	avatarUcase "2025_2_a4code/internal/usecase/avatar"
	messageUcase "2025_2_a4code/internal/usecase/message"
	pb "2025_2_a4code/messages-service/pkg/messagesproto"
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/grpc"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	envLocal = "local" // TO DO: или убрать в init_logger "2025_2_a4code/internal/pkg/init-logger" или здесь или вынести в отдельный файл
	envDev   = "dev"
	envProd  = "prod"
)

func MessagesInit() {
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
	log.Debug("messages: debug messages are enabled")

	// Установка соединения с бд
	connection, err := in.NewDbConnection(cfg.DBConfig)
	if err != nil {
		log.Error("error connecting to database")
		os.Exit(1)
	}

	// Подключение MinIO
	client, err := newMinioConnection(cfg.MinioConfig.Endpoint, cfg.MinioConfig.User, cfg.MinioConfig.Password, cfg.MinioConfig.UseSSL)
	if err != nil {
		log.Error("error connecting to minio")
	}

	err = bucketExists(client, cfg.MinioConfig.BucketName)
	if err != nil {
		log.Error("error checking bucket: " + err.Error())
	}

	var publicMinioClient *minio.Client
	if cfg.MinioConfig.PublicEndpoint != "" {
		publicMinioClient, err = minio.New(cfg.MinioConfig.PublicEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.MinioConfig.User, cfg.MinioConfig.Password, ""),
			Secure: cfg.MinioConfig.PublicUseSSL,
		})
		if err != nil {
			log.Error("error configuring public minio endpoint: " + err.Error())
			os.Exit(1)
		}
	}

	// Создание репозиториев
	messageRepository := messagerepository.New(connection)
	profileRepository := profilerepository.New(connection)
	avatarRepository := avatarrepository.New(client, publicMinioClient, cfg.MinioConfig.BucketName)

	// Создание юзкейсов
	messageUCase := messageUcase.New(messageRepository)
	avatarUCase := avatarUcase.New(avatarRepository, profileRepository)

	slog.Info("Messages microservice: server has started working...")

	grpcServer := grpc.NewServer()
	messagesService := messagesservice.New(messageUCase, avatarUCase, SECRET)
	pb.RegisterMessagesServiceServer(grpcServer, messagesService)

	lis, err := net.Listen("tcp", cfg.AppConfig.Host+":"+cfg.AppConfig.MessagesPort)
	if err != nil {
		log.Error("Failed to start server: " + err.Error())
		os.Exit(1)
	}

	slog.Info("Messages microservice: server has started working...")

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
