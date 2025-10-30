package logger

import (
	"log/slog"
	"net/http"
	"time"
)

//type Zap struct {
//	Log *zap.SugaredLogger
//}
//
//func (log *Zap) Initialize(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		start := time.Now()
//
//		next.ServeHTTP(w, r)
//
//		log.Log.Info(r.URL.Path,
//			zap.String("method", r.Method),
//			zap.String("remote_addr", r.RemoteAddr),
//			zap.String("url", r.URL.Path),
//			zap.Duration("work_time", time.Since(start)),
//		)
//	})
//}

func New(log *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		log := log.With(
			slog.String("component", "middleware/logger"),
		)

		fn := func(w http.ResponseWriter, r *http.Request) {
			entry := log.With(
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			)

			t1 := time.Now()
			defer func() {
				entry.Info("request completed",
					slog.String("duration", time.Since(t1).String()),
				)
			}()

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}
