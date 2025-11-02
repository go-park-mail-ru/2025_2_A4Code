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
	profilepage "2025_2_a4code/internal/http-server/handlers/user/profile-page"
	"2025_2_a4code/internal/http-server/handlers/user/settings"
	uploadfile "2025_2_a4code/internal/http-server/handlers/user/upload-file"
	"2025_2_a4code/internal/http-server/middleware/cors"
	"2025_2_a4code/internal/http-server/middleware/logger"
	e "2025_2_a4code/internal/lib/wrapper"
	messagerepository "2025_2_a4code/internal/storage/postgres/message-repository"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
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
	cfg := config.GetConfig()

	var SECRET = []byte(cfg.AppConfig.Secret)

	// Создание логгера
	log := setupLogger(envLocal)
	log.Debug("debug messages are enabled")
	loggerMiddleware := logger.New(log)

	// Установка соединения с базой
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

	// Создание репозиториев
	messageRepository := messagerepository.New(connection)
	profileRepository := profilerepository.New(connection)

	// Создание юзкейсов
	messageUCase := messageUcase.New(messageRepository)
	profileUCase := profileUcase.New(profileRepository)

	//lg, err := zap.NewProduction()
	//if err != nil {
	//	slog.Error(err.Error())
	//}
	//defer lg.Sync()
	//
	//sugar := lg.Sugar()
	//appLogger := sugar.With(zap.String("service", "app"))
	//
	//accessLogger := sugar.With(zap.String("service", "access_log"))
	//zlog := logger.Zap{Log: accessLogger}

	// Создание хэндлеров
	loginHandler := login.New(profileUCase, log, SECRET)
	signupHandler := signup.New(profileUCase, log, SECRET)
	refreshHandler := refresh.New(log, SECRET)
	logoutHandler := logout.New(log)
	inboxHandler := inbox.New(profileUCase, messageUCase, log, SECRET)
	meHandler := profilepage.New(profileUCase, SECRET, log)
	messagePageHandler := messagepage.New(profileUCase, messageUCase, SECRET, log)
	sendMessageHandler := send.New(messageUCase, SECRET, log)
	uploadFileHandler, err := uploadfile.New(FileUploadPath, log)
	settingsHandler := settings.New(profileUCase, SECRET, log)
	replyHandler := reply.New(messageUCase, SECRET, log)

	// настройка corsMiddleware
	corsMiddleware := cors.New()

	slog.Info("Starting server...", zap.String("address", cfg.AppConfig.Host+":"+cfg.AppConfig.Port))

	// роутинг + настройка middleware
	http.Handle("/auth/login", loggerMiddleware(corsMiddleware(http.HandlerFunc(loginHandler.ServeHTTP))))
	http.Handle("/auth/signup", loggerMiddleware(corsMiddleware(http.HandlerFunc(signupHandler.ServeHTTP))))
	http.Handle("/auth/refresh", loggerMiddleware(corsMiddleware(http.HandlerFunc(refreshHandler.ServeHTTP))))
	http.Handle("/auth/logout", loggerMiddleware(corsMiddleware(http.HandlerFunc(logoutHandler.ServeHTTP))))
	http.Handle("/messages/inbox", loggerMiddleware(corsMiddleware(http.HandlerFunc(inboxHandler.ServeHTTP))))
	http.Handle("/user/profile", loggerMiddleware(corsMiddleware(http.HandlerFunc(meHandler.ServeHTTP))))
	http.Handle("/messages/{message_id}", loggerMiddleware(corsMiddleware(http.HandlerFunc(messagePageHandler.ServeHTTP))))
	http.Handle("/messages/compose", loggerMiddleware(corsMiddleware(http.HandlerFunc(sendMessageHandler.ServeHTTP))))
	http.Handle("/upload", loggerMiddleware(corsMiddleware(http.HandlerFunc(uploadFileHandler.ServeHTTP))))
	http.Handle("/user/settings", loggerMiddleware(corsMiddleware(http.HandlerFunc(settingsHandler.ServeHTTP))))
	http.Handle("/messages/reply", loggerMiddleware(corsMiddleware(http.HandlerFunc(replyHandler.ServeHTTP))))

	err = http.ListenAndServe(cfg.AppConfig.Host+":"+cfg.AppConfig.Port, nil)

	// Для локального тестирования
	// err = http.ListenAndServe(":8080", nil)
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
	if err != nil {
		return e.Wrap(op, err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		slog.Info("Migrations are already successfully applied")
	} else {
		slog.Info("Migrations are successfully applied")
	}

	return nil
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default: // If env config is invalid, set prod settings by default due to security
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}
