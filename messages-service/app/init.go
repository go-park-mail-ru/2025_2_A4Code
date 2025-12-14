package app

import (
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/middleware/logger"
	in "2025_2_a4code/internal/lib/init"
	"2025_2_a4code/internal/lib/metrics"
	"2025_2_a4code/internal/lib/notifications"
	messagesservice "2025_2_a4code/messages-service/grpc-service"
	profileclient "2025_2_a4code/messages-service/internal/client/profile"
	"net"

	avatarrepository "2025_2_a4code/internal/storage/minio/avatar-repository"
	messagerepository "2025_2_a4code/internal/storage/postgres/message-repository"
	avatarUcase "2025_2_a4code/internal/usecase/avatar"
	messageUcase "2025_2_a4code/internal/usecase/message"
	pb "2025_2_a4code/messages-service/pkg/messagesproto"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/redis/go-redis/v9"
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

	go metrics.StartMetricsServer(cfg.AppConfig.MessagesMetricsPort, log)

	// Установка соединения с бд
	connection, err := in.NewDbConnection(cfg.DBConfig)
	if err != nil {
		log.Error("error connecting to database")
		os.Exit(1)
	}

	connection.SetMaxOpenConns(20)
	connection.SetMaxIdleConns(8)
	go metrics.MonitorDBConnections(connection)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Подключение Redis для уведомлений
	var redisClient *redis.Client
	var notifier *notifications.Notifier

	if cfg.RedisConfig.Host != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisConfig.Host + ":" + cfg.RedisConfig.Port,
			Password: cfg.RedisConfig.Password,
			DB:       cfg.RedisConfig.DB,
		})

		// Проверка соединения с Redis
		redisCtx, redisCancel := context.WithTimeout(ctx, 5*time.Second)
		defer redisCancel()

		if err := redisClient.Ping(redisCtx).Err(); err != nil {
			log.Warn("Failed to connect to Redis, notifications disabled", "error", err)
		} else {
			log.Info("Redis connected successfully")
		}
	}

	// Создаем клиент для profile-service
	profileServiceAddr := cfg.AppConfig.Host + ":" + cfg.AppConfig.ProfilePort
	var profileClient *profileclient.ProfileClient

	profileClient, err = profileclient.New(profileServiceAddr)
	if err != nil {
		log.Warn("Failed to create profile client, notifications disabled", "error", err)
	} else {
		log.Info("Profile client created successfully", "addr", profileServiceAddr)
	}

	// Создаем notifier
	if redisClient != nil {
		notifier = notifications.NewNotifier(redisClient)
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

	// Создание репозиториев
	messageRepository := messagerepository.New(connection)
	// profileRepository := profilerepository.New(connection)
	avatarRepository := avatarrepository.New(client, cfg.MinioConfig.BucketName, cfg.MinioConfig.PublicEndpoint, cfg.MinioConfig.PublicUseSSL)

	// Создание юзкейсов
	messageUCase := messageUcase.New(messageRepository)
	avatarUCase := avatarUcase.New(avatarRepository, nil)

	slog.Info("Messages microservice: server has started working...")

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logger.GrpcLoggerInterceptor(log),
			metrics.MetricsInterceptor("messages-service"),
		),
	)

	messagesService := messagesservice.New(
		messageUCase,
		avatarUCase,
		profileClient,
		notifier,
		redisClient,
		SECRET)

	pb.RegisterMessagesServiceServer(grpcServer, messagesService)

	lis, err := net.Listen("tcp", cfg.AppConfig.Host+":"+cfg.AppConfig.MessagesPort)
	if err != nil {
		log.Error("Failed to start server: " + err.Error())
		os.Exit(1)
	}

	slog.Info("Messages microservice: server has started working...")

	// Graceful shutdown для закрытия соединений
	go func() {
		<-ctx.Done()
		log.Info("Shutting down server gracefully...")

		grpcServer.GracefulStop()

		if profileClient != nil {
			profileClient.Close()
		}
		if redisClient != nil {
			redisClient.Close()
		}

		log.Info("Server stopped")
	}()

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
