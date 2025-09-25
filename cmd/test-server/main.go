package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"test-server/config/local"
	default_handler "test-server/internal/http-server/handlers/default-handler"
)

func main() {

	mux := http.NewServeMux()

	defaultHandler := &default_handler.DefaultHandler{}
	mux.HandleFunc("/", defaultHandler.ServeHTTP)

	cfg := local.New()
	srv := &http.Server{
		Addr:    cfg.Address,
		Handler: mux,
	}

	slog.Info(cfg.Address)

	if err := srv.ListenAndServe(); err != nil {
		fmt.Println(err.Error())
		slog.Error("failed to start server")
	}

}
