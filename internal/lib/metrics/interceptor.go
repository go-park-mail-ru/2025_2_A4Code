package metrics

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func NewGRPCMetricsInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start).Seconds()
		statusCode := status.Code(err).String()

		fullMethod := info.FullMethod
		methodName := fullMethod[strings.LastIndex(fullMethod, "/")+1:]

		// Запись метрик, используя переданный serviceName
		GRPCRequestsTotal.WithLabelValues(serviceName, methodName, statusCode).Inc()
		GRPCRequestDuration.WithLabelValues(serviceName, methodName).Observe(duration)

		return resp, err
	}
}
