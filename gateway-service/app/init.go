package app

import (
	gatewayservice "2025_2_a4code/gateway-service"
	"2025_2_a4code/internal/config"
	in "2025_2_a4code/internal/lib/init"
	"context"
	"log/slog"
	"os"
)

const ( // TODO: или убрать в init_logger "2025_2_a4code/internal/pkg/init-logger" или здесь или вынести в отдельный файл
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func GatewayInit() {

	// Создание логгера
	log := in.SetupLogger(envLocal)
	slog.SetDefault(log)
	log.Debug("auth: debug messages are enabled")
	//loggerMiddleware := logger.New(log)

	log.Info("Starting API Gateway...")

	// Читаем конфиг
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error("Error loading config: " + err.Error())
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, err := gatewayservice.NewServer(cfg.AppConfig)
	if err != nil {
		log.Error("Failed to start gateway server: " + err.Error())
		os.Exit(1)
	}

	if err := srv.Start(ctx); err != nil {
		log.Error("Gateway server error: " + err.Error())
		os.Exit(1)
	}

	log.Info("Gateway server stopped.")
}
