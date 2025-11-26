package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HttpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HttpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

var (
	// === gRPC метрики ===
	GRPCRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"service", "method", "status"},
	)

	GRPCRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "Duration of gRPC requests",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"service", "method"},
	)

	GRPCRequestsInProgress = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_requests_in_progress",
			Help: "Number of gRPC requests currently in progress",
		},
		[]string{"service", "method"},
	)

	AuthLoginAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_login_attempts_total",
			Help: "Total number of login attempts",
		},
		[]string{"status"},
	)

	AuthSignupAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_signup_attempts_total",
			Help: "Total number of signup attempts",
		},
		[]string{"status"},
	)

	AuthTokenRefreshes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_token_refreshes_total",
			Help: "Total number of token refresh attempts",
		},
		[]string{"status"},
	)

	AuthLogoutsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_logouts_total",
			Help: "Total number of logout operations",
		},
		[]string{"status"},
	)

	// === Метрики токенов ===
	TokenGenerations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_token_generations_total",
			Help: "Total number of token generations",
		},
		[]string{"token_type", "status"},
	)

	TokenGenerationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_token_generation_errors_total",
			Help: "Total number of token generation errors",
		},
		[]string{"token_type"},
	)

	TokenGenerationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_token_generation_duration_seconds",
			Help:    "Duration of token generation operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"token_type"},
	)

	TokenValidationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "auth_token_validation_duration_seconds",
			Help:    "Duration of token validation operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"token_type"},
	)

	JWTValidationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_jwt_validation_errors_total",
			Help: "Total number of JWT validation errors",
		},
		[]string{"token_type", "error_type"},
	)

	// === Бизнес-метрики для messages ===

	MessagesSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_service_sent_total",
			Help: "Total number of successfully sent messages by type (send, reply, draft_send)",
		},
		[]string{"type"},
	)

	MessagesOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "messages_operations_total",
			Help: "Total number of messages operations",
		},
		[]string{"service", "operation", "status"},
	)

	MessagesOperationsDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "messages_operations_duration_seconds",
			Help:    "Duration of messages operations",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1, 2, 5},
		},
		[]string{"service", "operation"},
	)

	MessagesCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "messages_count",
			Help: "Number of messages in the system",
		},
		[]string{"service", "type"},
	)

	// === Метрики файлов ===
	FileOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "file_operations_total",
			Help: "Total number of file operations",
		},
		[]string{"service", "operation", "status"},
	)

	FileSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "file_size_bytes",
			Help:    "Size of uploaded files",
			Buckets: []float64{1024, 10240, 102400, 1048576, 10485760},
		},
		[]string{"service", "file_type"},
	)

	// === Метрики потоков (threads) ===
	ThreadOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "thread_operations_total",
			Help: "Total number of thread operations",
		},
		[]string{"service", "operation", "status"},
	)

	// === Метрики аватаров ===
	AvatarOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "avatar_operations_total",
			Help: "Total number of avatar operations",
		},
		[]string{"service", "operation", "status"},
	)

	// === Метрики ошибок ===
	BusinessErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "business_errors_total",
			Help: "Total number of business logic errors",
		},
		[]string{"service", "operation", "error_type"},
	)

	// === Метрики базы данных ===
	DBConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_connections",
			Help: "Number of database connections",
		},
		[]string{"state"},
	)

	DBQueryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_query_errors_total",
			Help: "Total number of database query errors",
		},
		[]string{"service", "operation"},
	)

	// === Метрики MinIO ===
	MinIOOperations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "minio_operations_total",
			Help: "Total number of MinIO operations",
		},
		[]string{"service", "operation", "status"},
	)

	MinIOOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "minio_operation_duration_seconds",
			Help:    "Duration of MinIO operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
		},
		[]string{"service", "operation"},
	)
)
