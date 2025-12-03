package app

import (
	authservice "2025_2_a4code/auth-service/grpc-service"
	pb "2025_2_a4code/auth-service/pkg/authproto"
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/middleware/logger"
	in "2025_2_a4code/internal/lib/init"
	"2025_2_a4code/internal/lib/metrics"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"log/slog"
	"net"
	"os"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
	"google.golang.org/grpc"
)

const ( // TODO: или убрать в init_logger "2025_2_a4code/internal/pkg/init-logger" или здесь или вынести в отдельный файл
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func AuthInit() {
	// Читаем конфиг
	cfg, err := config.GetConfig()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}

	var SECRET = []byte(cfg.AppConfig.Secret)

	// Создание логгера
	log := in.SetupLogger(envLocal)
	slog.SetDefault(log)
	log.Debug("auth: debug messages are enabled")
	//loggerMiddleware := logger.New(log)

	go metrics.StartMetricsServer(cfg.AppConfig.AuthMetricsPort, log)

	// Установка соединения с бд
	log.Info(cfg.DBConfig.Host + ":" + cfg.DBConfig.Port)
	connection, err := in.NewDbConnection(cfg.DBConfig)
	if err != nil {
		log.Error("error connecting to database" + err.Error())
		metrics.DBQueryErrors.WithLabelValues("auth-service", "connection").Inc()
		os.Exit(1)
	}
	connection.SetMaxOpenConns(20)
	connection.SetMaxIdleConns(8)

	go metrics.MonitorDBConnections(connection)

	// Миграции
	err = in.RunMigrations(connection, "file://./db/migrations")
	if err != nil {
		log.Error(err.Error())
		metrics.DBQueryErrors.WithLabelValues("auth-service", "migration").Inc()
	}

	profileRepository := profilerepository.New(connection)
	profileUCase := profileUcase.New(profileRepository)

	// создаем gRPC сервер и регистрируем наш сервис
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(logger.GrpcLoggerInterceptor(log), metrics.MetricsInterceptor("auth-service")),
	)
	authService := authservice.New(profileUCase, SECRET)
	pb.RegisterAuthServiceServer(grpcServer, authService)

	// Запуск
	lis, err := net.Listen("tcp", cfg.AppConfig.Host+":"+cfg.AppConfig.AuthPort)
	if err != nil {
		log.Error("Failed to start server: " + err.Error())
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "server_start", "listen_error").Inc()
		os.Exit(1)
	}

	log.Info("Auth microservice: server has started working...")

	if err := grpcServer.Serve(lis); err != nil {
		log.Error("gRPC server failed: " + err.Error())
		metrics.BusinessErrorsTotal.WithLabelValues("auth-service", "grpc_server", "serve_error").Inc()
		os.Exit(1)
	}
}
