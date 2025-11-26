package app

import (
	authservice "2025_2_a4code/auth-service"
	pb "2025_2_a4code/auth-service/pkg/authproto"
	"2025_2_a4code/internal/config"
	"2025_2_a4code/internal/http-server/middleware/logger"
	in "2025_2_a4code/internal/lib/init"
	"2025_2_a4code/internal/lib/metrics"
	profilerepository "2025_2_a4code/internal/storage/postgres/profile-repository"
	profileUcase "2025_2_a4code/internal/usecase/profile"
	"context"
	"database/sql"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v4/stdlib"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	go startMetricsServer(cfg.AppConfig.AuthMetricsPort, log)

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

	go monitorDBConnections(connection)

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
		grpc.ChainUnaryInterceptor(logger.GrpcLoggerInterceptor(log), metricsInterceptor("auth-service")),
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

func startMetricsServer(port string, log *slog.Logger) {
	http.Handle("/metrics", promhttp.Handler())
	addr := ":" + port
	log.Info("Starting metrics server on " + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Error("Failed to start metrics server: " + err.Error())
	}
}

func metricsInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// увеличение счетчика активных запросов
		metrics.GRPCRequestsInProgress.WithLabelValues(serviceName, info.FullMethod).Inc()
		defer metrics.GRPCRequestsInProgress.WithLabelValues(serviceName, info.FullMethod).Dec()

		resp, err := handler(ctx, req)
		duration := time.Since(start).Seconds()

		status := "success"
		if err != nil {
			status = "error"
			// сбор ошибок по методам
			metrics.BusinessErrorsTotal.WithLabelValues(serviceName, info.FullMethod, "grpc_error").Inc()
		}

		// сбор метрик запросов и времени выполнения
		metrics.GRPCRequestsTotal.WithLabelValues(serviceName, info.FullMethod, status).Inc()
		metrics.GRPCRequestDuration.WithLabelValues(serviceName, info.FullMethod).Observe(duration)

		return resp, err
	}
}

func monitorDBConnections(connection *sql.DB) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := connection.Stats()
		// установка метрик состояния соединений БД
		metrics.DBConnections.WithLabelValues("idle").Set(float64(stats.Idle))
		metrics.DBConnections.WithLabelValues("active").Set(float64(stats.InUse))
		metrics.DBConnections.WithLabelValues("open").Set(float64(stats.OpenConnections))
		metrics.DBConnections.WithLabelValues("max_open").Set(float64(stats.MaxOpenConnections))
	}
}
