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
	profile_page "2025_2_a4code/internal/http-server/handlers/user/profile-page"
	"2025_2_a4code/internal/http-server/handlers/user/settings"
	uploadfile "2025_2_a4code/internal/http-server/handlers/user/upload-file"
	"2025_2_a4code/internal/http-server/middleware/logger"
	messagerepository "2025_2_a4code/internal/storage/postgres/message-repository"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

const (
	FileUploadPath = "./files" // TODO: в дальнейшем будет минио
)

var SECRET = []byte("secret")

type Storage struct {
	db *sql.DB
}

func Init() {
	// Читаем конфиг
	cfg := config.GetConfig()

	// Установка соединения с базой
	connection, err := newDbConnection(cfg.DBConfig)
	if err != nil {
		log.Fatal(err)
	}

	// Создание репозиториев
	messageRepository := messagerepository.New(connection)
	profileRepository := profilerepository.New(connection)

	// Создание юзкейсов
	messageUCase := messageUcase.New(messageRepository)
	profileUCase := profileUcase.New(profileRepository)

	// Создание хэндлеров
	loginHandler := login.New(profileUCase, SECRET)
	signupHandler := signup.New(profileUCase, SECRET)
	refreshHandler := refresh.New(SECRET)
	logoutHandler := logout.New()
	inboxHandler := inbox.New(profileUCase, messageUCase)
	meHandler := profile_page.New(profileUCase)
	messagePageHandler := messagepage.New(profileUCase, messageUCase)
	sendMessageHandler := send.New(messageUCase)
	uploadFileHandler, err := uploadfile.New(FileUploadPath)
	settingsHandler := settings.New(profileUCase, SECRET)
	replyHandler := reply.New(messageUCase, SECRET)

	// Создание логгера
	lg, err := zap.NewProduction()
	if err != nil {
		slog.Error(err.Error())
	}
	defer lg.Sync()
	lg.Info("Starting server...", zap.String("address", cfg.AppConfig.Host+":"+cfg.AppConfig.Port))
	sugar := lg.Sugar().With(zap.String("mode", "[access_log]"))
	zlog := logger.Logger{Zlog: sugar}

	// роутинг + настройка middleware
	http.Handle("/auth/login", zlog.Initialize(corsMiddleware(http.HandlerFunc(loginHandler.ServeHTTP))))
	http.Handle("/auth/signup", zlog.Initialize(corsMiddleware(http.HandlerFunc(signupHandler.ServeHTTP))))
	http.Handle("/auth/refresh", zlog.Initialize(corsMiddleware(http.HandlerFunc(refreshHandler.ServeHTTP))))
	http.Handle("/auth/logout", zlog.Initialize(corsMiddleware(http.HandlerFunc(logoutHandler.ServeHTTP))))
	http.Handle("/messages/inbox", zlog.Initialize(corsMiddleware(http.HandlerFunc(inboxHandler.ServeHTTP))))
	http.Handle("/user/profile", zlog.Initialize(corsMiddleware(http.HandlerFunc(meHandler.ServeHTTP))))
	http.Handle("/messages/{message_id}", zlog.Initialize(corsMiddleware(http.HandlerFunc(messagePageHandler.ServeHTTP))))
	http.Handle("/messages/compose", zlog.Initialize(corsMiddleware(http.HandlerFunc(sendMessageHandler.ServeHTTP))))
	http.Handle("/upload", zlog.Initialize(corsMiddleware(http.HandlerFunc(uploadFileHandler.ServeHTTP))))
	http.Handle("/user/settings", zlog.Initialize(corsMiddleware(http.HandlerFunc(settingsHandler.ServeHTTP))))
	http.Handle("/messages/reply", zlog.Initialize(corsMiddleware(http.HandlerFunc(replyHandler.ServeHTTP))))

	err = http.ListenAndServe(cfg.AppConfig.Host+":"+cfg.AppConfig.Port, nil)

	// Для локального тестирования
	// err = http.ListenAndServe(":8080", nil)

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
		return nil, fmt.Errorf(op+": %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf(op+": %w", err)
	}

	log.Println("Connected to postgresql successfully")

	return db, nil
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000") // TODO: change it
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
