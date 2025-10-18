package app

import (
	"2025_2_a4code/internal/config"
	health_check "2025_2_a4code/internal/http-server/handlers/health-check"
	"2025_2_a4code/internal/http-server/handlers/inbox"
	"2025_2_a4code/internal/http-server/handlers/login"
	"2025_2_a4code/internal/http-server/handlers/logout"
	"2025_2_a4code/internal/http-server/handlers/me"
	message_page "2025_2_a4code/internal/http-server/handlers/message-page"
	"2025_2_a4code/internal/http-server/handlers/signup"
	messagerepository "2025_2_a4code/internal/storage/postgres/message-repository"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

// TODO: подключение хэндлеров, бд и тд

const (
	StoragePath = "./storage"
)

var SECRET = []byte("secret")

type Storage struct {
	db *sql.DB
}

func Init() {
	cfg := config.GetConfig() // Путь к конфигу config/prod.yml

	connection, err := newDbConnection(StoragePath)
	if err != nil {
		log.Fatal(err)
	}
	messageRepository := messagerepository.New(connection)
	profileRepository := profilerepository.New(connection)

	messageUCase := messageUcase.New(messageRepository)
	profileUCase := profileUcase.New(profileRepository)

	healthCheckHandler := health_check.New(SECRET)
	loginHandler := login.New(profileUCase, SECRET)
	signupHandler := signup.New(profileUCase, SECRET)
	logoutHandler := logout.New()
	inboxHandler := inbox.New(profileUCase, messageUCase)
	meHandler := me.New(profileUCase)
	messagePageHandler := message_page.New(profileUCase, messageUCase)

	http.Handle("/", corsMiddleware(http.HandlerFunc(healthCheckHandler.ServeHTTP)))
	http.Handle("/login", corsMiddleware(http.HandlerFunc(loginHandler.ServeHTTP)))
	http.Handle("/signup", corsMiddleware(http.HandlerFunc(signupHandler.ServeHTTP)))
	http.Handle("/logout", corsMiddleware(http.HandlerFunc(logoutHandler.ServeHTTP)))
	http.Handle("/inbox", corsMiddleware(http.HandlerFunc(inboxHandler.ServeHTTP)))
	http.Handle("/me", corsMiddleware(http.HandlerFunc(meHandler.ServeHTTP)))
	http.Handle("/message-page", corsMiddleware(http.HandlerFunc(messagePageHandler.ServeHTTP))) // TODO: создать индивидуальный параметр в url

	err = http.ListenAndServe(":"+cfg.AppConfig.ConfigPath, nil)

	// Для локального тестирования
	//err := http.ListenAndServe(":8080", nil)

	if err != nil {
		panic(err)
	}

}

func newDbConnection(storagePath string) (*sql.DB, error) {
	const op = "app.newDbConnection"

	db, err := sql.Open("postgres", storagePath)
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
