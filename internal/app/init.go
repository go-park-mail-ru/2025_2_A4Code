package app

import (
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/handlers/inbox"
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

	messageUsecase := messageUcase.New(messageRepository)
	profileUsecase := profileUcase.New(profileRepository)
	inboxHandler := inbox.New(profileUsecase, messageUsecase)

	//http.Handle("/", corsMiddleware(http.HandlerFunc(h.HealthCheckHandler)))
	//http.Handle("/login", corsMiddleware(http.HandlerFunc(h.LoginHandler)))
	//http.Handle("/signup", corsMiddleware(http.HandlerFunc(h.SignupHandler)))
	//http.Handle("/logout", corsMiddleware(http.HandlerFunc(h.LogoutHandler)))
	//http.Handle("/me", corsMiddleware(http.HandlerFunc(h.MeHandler)))
	http.Handle("/inbox", corsMiddleware(http.HandlerFunc(inboxHandler.ServeHTTP)))

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
