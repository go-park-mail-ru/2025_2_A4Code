package main

import (
	"2025_2_a4code/config"
	"2025_2_a4code/handlers"
	"net/http"
)

func main() {
	cfg := config.GetConfig()

	h := handlers.New()

	http.HandleFunc("/", h.HealthCheckHandler)
	http.HandleFunc("/login", h.LoginHandler)

	err := http.ListenAndServe(":"+cfg.AppConfig.Port, nil)

	if err != nil {
		panic(err)
	}
}
