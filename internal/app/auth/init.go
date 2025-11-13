package app

import (
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/handlers/auth/login"
	"2025_2_a4code/internal/http-server/handlers/auth/logout"
	"2025_2_a4code/internal/http-server/handlers/auth/refresh"
	"2025_2_a4code/internal/http-server/handlers/auth/signup"
	"2025_2_a4code/internal/http-server/middleware/cors"
	"2025_2_a4code/internal/http-server/middleware/logger"
	init2 "2025_2_a4code/internal/lib/init"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
)

const ( // TO DO: или убрать в init_logger "2025_2_a4code/internal/pkg/init-logger" или здесь или вынести в отдельный файл
	envLocal = "local"
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
	log := init2.SetupLogger(envLocal)
	slog.SetDefault(log)
	log.Debug("auth: debug messages are enabled")
	loggerMiddleware := logger.New(log)

	// Установка соединения с бд
	connection, err := init2.NewDbConnection(cfg.DBConfig)
	if err != nil {
		log.Error("error connecting to database")
		os.Exit(1)
	}

	// Миграции
	err = init2.RunMigrations(connection, "file://./db/migrations")
	if err != nil {
		log.Error(err.Error())
	}

	profileRepository := profilerepository.New(connection)
	profileUCase := profileUcase.New(profileRepository)

	// Создание хэндлеров
	loginHandler := login.New(profileUCase, SECRET)
	signupHandler := signup.New(profileUCase, SECRET)
	refreshHandler := refresh.New(SECRET)
	logoutHandler := logout.New()

	// настройка corsMiddlewares
	corsMiddleware := cors.New()

	slog.Info("Starting server...", slog.String("address", cfg.AppConfig.Host+":"+cfg.AppConfig.AuthPort))

	// роутинг + настройка middleware
	http.Handle("/auth/login", loggerMiddleware(corsMiddleware(http.HandlerFunc(loginHandler.ServeHTTP))))
	http.Handle("/auth/signup", loggerMiddleware(corsMiddleware(http.HandlerFunc(signupHandler.ServeHTTP))))
	http.Handle("/auth/refresh", loggerMiddleware(corsMiddleware(http.HandlerFunc(refreshHandler.ServeHTTP))))
	http.Handle("/auth/logout", loggerMiddleware(corsMiddleware(http.HandlerFunc(logoutHandler.ServeHTTP))))

	err = http.ListenAndServe(cfg.AppConfig.Host+":"+cfg.AppConfig.AuthPort, nil)

	// Для локального тестирования
	//err = http.ListenAndServe(":8081", nil)
	slog.Info("Auth microservice: server has started working...")
	if err != nil {
		panic(err)
	}

}
