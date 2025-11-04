package logger

import (
	"2025_2_a4code/internal/lib/rand"
	"context"
	"log/slog"
	"net/http"
	"time"
)

type contextKey string

const loggerKey = contextKey("logger")

func GetLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log := log.With(
			slog.String("component", "middleware/logger"),
		)

		fn := func(w http.ResponseWriter, r *http.Request) {
			requestID, err := rand.GenerateRandID()
			if err != nil {
				slog.Error("Failed to generate request_id" + err.Error())
			}

			reqLog := log.With(
				slog.String("request_id", requestID),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("user_agent", r.UserAgent()),
			)

			t1 := time.Now()
			defer func() {
				reqLog.Info("request completed",
					slog.String("duration", time.Since(t1).String()),
				)
			}()
			ctx := context.WithValue(r.Context(), loggerKey, reqLog)
			next.ServeHTTP(w, r.WithContext(ctx))
		}

		return http.HandlerFunc(fn)
	}
}
