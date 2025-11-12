package app

import (
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/handlers/messages/inbox"
	messagepage "2025_2_a4code/internal/http-server/handlers/messages/message-page"
	"2025_2_a4code/internal/http-server/handlers/messages/reply"
	"2025_2_a4code/internal/http-server/handlers/messages/send"

	// "2025_2_a4code/internal/http-server/handlers/messages/threads"
	// uploadfile "2025_2_a4code/internal/http-server/handlers/user/upload/upload-file"
	"2025_2_a4code/internal/http-server/middleware/cors"
	"2025_2_a4code/internal/http-server/middleware/logger"
	init_database "2025_2_a4code/internal/pkg/init-database"
	init_logger "2025_2_a4code/internal/pkg/init-logger"
	avatarrepository "2025_2_a4code/internal/storage/minio/avatar-repository"
	messagerepository "2025_2_a4code/internal/storage/postgres/message-repository"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	avatarUcase "2025_2_a4code/internal/usecase/avatar"
	messageUcase "2025_2_a4code/internal/usecase/message"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	envLocal = "local" // TO DO: или убрать в init_logger "2025_2_a4code/internal/pkg/init-logger" или здесь или вынести в отдельный файл
	envDev   = "dev"
	envProd  = "prod"
)

func Init() {
	// Читаем конфиг
	cfg, err := config.GetConfig()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	var SECRET = []byte(cfg.AppConfig.Secret)

	// Создание логгера
	log := init_logger.SetupLogger(envLocal)
	slog.SetDefault(log)
	log.Debug("messages: debug messages are enabled")
	loggerMiddleware := logger.New(log)

	// Установка соединения с бд
	connection, err := init_database.NewDbConnection(cfg.DBConfig)
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

	// Создание хэндлеров
	inboxHandler := inbox.New(messageUCase, avatarUCase, SECRET)
	messagePageHandler := messagepage.New(messageUCase, avatarUCase, SECRET)
	sendMessageHandler := send.New(messageUCase, SECRET)
	// threadsHandler := threads.New(profileUCase, messageUCase, SECRET)
	// uploadFileHandler, err := uploadfile.New(FileUploadPath)
	replyHandler := reply.New(messageUCase, SECRET)

	// настройка corsMiddlewares
	corsMiddleware := cors.New()

	slog.Info("Starting server...", slog.String("address", cfg.AppConfig.Host+":"+cfg.AppConfig.MessagesPort))

	// роутинг + настройка middleware
	http.Handle("/messages/inbox", loggerMiddleware(corsMiddleware(http.HandlerFunc(inboxHandler.ServeHTTP))))
	http.Handle("/messages/{message_id}", loggerMiddleware(corsMiddleware(http.HandlerFunc(messagePageHandler.ServeHTTP))))
	http.Handle("/messages/send", loggerMiddleware(corsMiddleware(http.HandlerFunc(sendMessageHandler.ServeHTTP))))
	// http.Handle("/messages/threads", loggerMiddleware(corsMiddleware(http.HandlerFunc(threadsHandler.ServeHTTP))))
	// http.Handle("/upload/file", loggerMiddleware(corsMiddleware(http.HandlerFunc(uploadFileHandler.ServeHTTP))))
	http.Handle("/messages/reply", loggerMiddleware(corsMiddleware(http.HandlerFunc(replyHandler.ServeHTTP))))

	err = http.ListenAndServe(cfg.AppConfig.Host+":"+cfg.AppConfig.MessagesPort, nil)

	// Для локального тестирования
	//err = http.ListenAndServe(":8082", nil)
	slog.Info("Messages microservice: server has started working...")
	if err != nil {
		panic(err)
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
