package main

import (
	"2025_2_a4code/config"
	"2025_2_a4code/handlers"
	"net/http"
)

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

func main() {
	cfg := config.GetConfig()

	h := handlers.New()

	http.Handle("/", corsMiddleware(http.HandlerFunc(h.HealthCheckHandler)))
	http.Handle("/login", corsMiddleware(http.HandlerFunc(h.LoginHandler)))
	http.Handle("/inbox", corsMiddleware(http.HandlerFunc(h.MainPageHandler)))
	http.Handle("/signup", corsMiddleware(http.HandlerFunc(h.SignupHandler)))

	err := http.ListenAndServe(":"+cfg.AppConfig.Port, nil)
	if err != nil {
		panic(err)
	}
}
