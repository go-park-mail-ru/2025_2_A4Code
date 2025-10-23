package app

import (
	"2025_2_a4code/internal/config"
	healthcheck "2025_2_a4code/internal/http-server/handlers/health-check"
	"2025_2_a4code/internal/http-server/handlers/inbox"
	"2025_2_a4code/internal/http-server/handlers/login"
	"2025_2_a4code/internal/http-server/handlers/logout"
	"2025_2_a4code/internal/http-server/handlers/me"
	messagepage "2025_2_a4code/internal/http-server/handlers/message-page"
	sendmessage "2025_2_a4code/internal/http-server/handlers/send-message"
	"2025_2_a4code/internal/http-server/handlers/signup"
	uploadfile "2025_2_a4code/internal/http-server/handlers/upload-file"
	messagerepository "2025_2_a4code/internal/storage/postgres/message-repository"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	messageUcase "2025_2_a4code/internal/usecase/message"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	_ "github.com/lib/pq"
)

const (
	FileUploadPath = "./files" // TODO: в дальнейшем будет минио
)

var SECRET = []byte("secret")

type Storage struct {
	db *sql.DB
}

func Init() {
	cfg := config.GetConfig() // Путь к конфигу config/prod.yml

	connection, err := newDbConnection(cfg.DBConfig)
	if err != nil {
		log.Fatal(err)
	}
	messageRepository := messagerepository.New(connection)
	profileRepository := profilerepository.New(connection)

	messageUCase := messageUcase.New(messageRepository)
	profileUCase := profileUcase.New(profileRepository)

	healthCheckHandler := healthcheck.New(SECRET)
	loginHandler := login.New(profileUCase, SECRET)
	signupHandler := signup.New(profileUCase, SECRET)
	logoutHandler := logout.New()
	inboxHandler := inbox.New(profileUCase, messageUCase)
	meHandler := me.New(profileUCase)
	messagePageHandler := messagepage.New(profileUCase, messageUCase)
	sendMessageHandler := sendmessage.New(messageUCase)
	uploadFileHandler, err := uploadfile.New(FileUploadPath)

	http.Handle("/", corsMiddleware(http.HandlerFunc(healthCheckHandler.ServeHTTP)))
	http.Handle("/login", corsMiddleware(http.HandlerFunc(loginHandler.ServeHTTP)))
	http.Handle("/signup", corsMiddleware(http.HandlerFunc(signupHandler.ServeHTTP)))
	http.Handle("/logout", corsMiddleware(http.HandlerFunc(logoutHandler.ServeHTTP)))
	http.Handle("/inbox", corsMiddleware(http.HandlerFunc(inboxHandler.ServeHTTP)))
	http.Handle("/me", corsMiddleware(http.HandlerFunc(meHandler.ServeHTTP)))
	http.Handle("/{message_id}", corsMiddleware(http.HandlerFunc(messagePageHandler.ServeHTTP)))
	http.Handle("/compose", corsMiddleware(http.HandlerFunc(sendMessageHandler.ServeHTTP)))
	http.Handle("/upload", corsMiddleware(http.HandlerFunc(uploadFileHandler.ServeHTTP)))

	//err = http.ListenAndServe(":"+cfg.AppConfig.Port, nil)

	// Для локального тестирования
	err = http.ListenAndServe(":8080", nil)

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
