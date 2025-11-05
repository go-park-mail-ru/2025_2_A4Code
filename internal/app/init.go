package app

import (
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/handlers/auth/login"
	"2025_2_a4code/internal/http-server/handlers/auth/logout"
	"2025_2_a4code/internal/http-server/handlers/auth/refresh"
	"2025_2_a4code/internal/http-server/handlers/auth/signup"
	"2025_2_a4code/internal/http-server/handlers/messages/inbox"
	messagepage "2025_2_a4code/internal/http-server/handlers/messages/message-page"
	"2025_2_a4code/internal/http-server/handlers/messages/reply"
	"2025_2_a4code/internal/http-server/handlers/messages/send"
	"2025_2_a4code/internal/http-server/handlers/messages/sent"
	profilepage "2025_2_a4code/internal/http-server/handlers/user/profile-page"
	"2025_2_a4code/internal/http-server/handlers/user/settings"
	uploadavatar "2025_2_a4code/internal/http-server/handlers/user/upload/upload-avatar"
	uploadfile "2025_2_a4code/internal/http-server/handlers/user/upload/upload-file"
	"2025_2_a4code/internal/http-server/middleware/cors"
	"2025_2_a4code/internal/http-server/middleware/logger"
	e "2025_2_a4code/internal/lib/wrapper"
	avatarrepository "2025_2_a4code/internal/storage/minio/avatar-repository"
	messagerepository "2025_2_a4code/internal/storage/postgres/message-repository"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	avatarUcase "2025_2_a4code/internal/usecase/avatar"
	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	FileUploadPath = "./files" // TODO: в дальнейшем будет минио
	envLocal       = "local"
	envDev         = "dev"
	envProd        = "prod"
)

type Storage struct {
	db *sql.DB
}

func Init() {
	// Читаем конфиг
	cfg, err := config.GetConfig()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	var SECRET = []byte(cfg.AppConfig.Secret)

	// Создание логгера
	log := setupLogger(envLocal)
	slog.SetDefault(log)
	log.Debug("debug messages are enabled")
	loggerMiddleware := logger.New(log)

	// Установка соединения с бд
	connection, err := newDbConnection(cfg.DBConfig)
	if err != nil {
		log.Error("error connecting to database")
		os.Exit(1)
	}

	// Миграции
	err = runMigrations(connection, "file://./db/migrations")
	if err != nil {
		log.Error(err.Error())
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
	profileRepository := profilerepository.New(connection)
	avatarRepository := avatarrepository.New(client, cfg.MinioConfig.BucketName)

	// Создание юзкейсов
	messageUCase := messageUcase.New(messageRepository)
	profileUCase := profileUcase.New(profileRepository)
	avatarUCase := avatarUcase.New(avatarRepository, profileRepository)

	// Создание хэндлеров
	loginHandler := login.New(profileUCase, SECRET)
	signupHandler := signup.New(profileUCase, SECRET)
	refreshHandler := refresh.New(SECRET)
	logoutHandler := logout.New()
	inboxHandler := inbox.New(messageUCase, avatarUCase, SECRET)
	profileHandler := profilepage.New(profileUCase, avatarUCase, SECRET)
	messagePageHandler := messagepage.New(messageUCase, avatarUCase, SECRET)
	sendMessageHandler := send.New(messageUCase, SECRET)
	uploadFileHandler, err := uploadfile.New(FileUploadPath)
	settingsHandler := settings.New(profileUCase, SECRET)
	replyHandler := reply.New(messageUCase, SECRET)
	uploadAvatarHandler := uploadavatar.New(avatarUCase, profileUCase, SECRET)
	sentHandler := sent.New(messageUCase, avatarUCase, SECRET)

	// настройка corsMiddleware
	corsMiddleware := cors.New()

	slog.Info("Starting server...", slog.String("address", cfg.AppConfig.Host+":"+cfg.AppConfig.Port))

	// роутинг + настройка middleware
	http.Handle("/auth/login", loggerMiddleware(corsMiddleware(http.HandlerFunc(loginHandler.ServeHTTP))))
	http.Handle("/auth/signup", loggerMiddleware(corsMiddleware(http.HandlerFunc(signupHandler.ServeHTTP))))
	http.Handle("/auth/refresh", loggerMiddleware(corsMiddleware(http.HandlerFunc(refreshHandler.ServeHTTP))))
	http.Handle("/auth/logout", loggerMiddleware(corsMiddleware(http.HandlerFunc(logoutHandler.ServeHTTP))))
	http.Handle("/messages/inbox", loggerMiddleware(corsMiddleware(http.HandlerFunc(inboxHandler.ServeHTTP))))
	http.Handle("/user/profile", loggerMiddleware(corsMiddleware(http.HandlerFunc(profileHandler.ServeHTTP))))
	http.Handle("/messages/{message_id}", loggerMiddleware(corsMiddleware(http.HandlerFunc(messagePageHandler.ServeHTTP))))
	http.Handle("/messages/send", loggerMiddleware(corsMiddleware(http.HandlerFunc(sendMessageHandler.ServeHTTP))))
	http.Handle("/user/upload/file", loggerMiddleware(corsMiddleware(http.HandlerFunc(uploadFileHandler.ServeHTTP))))
	http.Handle("/user/settings", loggerMiddleware(corsMiddleware(http.HandlerFunc(settingsHandler.ServeHTTP))))
	http.Handle("/messages/reply", loggerMiddleware(corsMiddleware(http.HandlerFunc(replyHandler.ServeHTTP))))
	http.Handle("/user/upload/avatar", loggerMiddleware(corsMiddleware(http.HandlerFunc(uploadAvatarHandler.ServeHTTP))))
	http.Handle("/messages/sent", loggerMiddleware(corsMiddleware(http.HandlerFunc(sentHandler.ServeHTTP))))

	//err = http.ListenAndServe(cfg.AppConfig.Host+":"+cfg.AppConfig.Port, nil)

	// Для локального тестирования
	err = http.ListenAndServe(":8080", nil)

	slog.Info("Server has started working...")

	if err != nil {
		panic(err)
	}

}

func newDbConnection(dbConfig *config.DBConfig) (*sql.DB, error) {
	const op = "app.newDbConnection"

	connectionString := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.Name, dbConfig.SSLMode,
	)

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, e.Wrap(op, err)
	}

	if err = db.Ping(); err != nil {
		return nil, e.Wrap(op, err)
	}

	slog.Info("Connected to postgresql successfully")

	return db, nil
}

func runMigrations(db *sql.DB, migrationsDir string) error {
	const op = "app.runMigrations"

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return e.Wrap(op, err)
	}

	m, err := migrate.NewWithDatabaseInstance(migrationsDir, "postgres", driver)
	if err != nil {
		return e.Wrap(op, err)
	}

	slog.Info("Applying migrations...")
	err = m.Up()
	switch {
	case err == nil:
		slog.Info("Migrations are successfully applied")
		return nil
	case errors.Is(err, migrate.ErrNoChange):
		slog.Info("Migrations are already successfully applied")
		return nil
	case errors.As(err, new(*migrate.ErrDirty)):
		version, dirty, vErr := m.Version()
		if vErr != nil {
			return e.Wrap(op, vErr)
		}
		if dirty {
			forceVersion := int(version)
			if forceVersion > 0 {
				forceVersion--
			}

			slog.Warn("Detected dirty migration state, forcing database version", "current_version", version, "force_version", forceVersion)
			if forceErr := m.Force(forceVersion); forceErr != nil {
				return e.Wrap(op, forceErr)
			}

			if retryErr := m.Up(); retryErr != nil && !errors.Is(retryErr, migrate.ErrNoChange) {
				return e.Wrap(op, retryErr)
			}

			slog.Info("Migrations are successfully applied after resolving dirty state")
			return nil
		}
		return nil
	default:
		return e.Wrap(op, err)
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	replaceAttrFunc := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == slog.TimeKey && len(groups) == 0 {
			if t, ok := a.Value.Any().(time.Time); ok {
				a.Value = slog.StringValue(t.Format("2006/01/02 15:04:05"))
			}
		}
		return a
	}

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: replaceAttrFunc}),
		)
	case envDev:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug, ReplaceAttr: replaceAttrFunc}),
		)
	case envProd:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo, ReplaceAttr: replaceAttrFunc}),
		)
	default:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo, ReplaceAttr: replaceAttrFunc}),
		)
	}

	return log
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

	slog.Info("Connected to MinIO successfully")

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
