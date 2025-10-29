package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type Logger struct {
	Zlog *zap.SugaredLogger
}

func (log *Logger) Initialize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Zlog.Info(r.URL.Path,
			zap.String("method", r.Method),
			zap.String("remote_addr", r.RemoteAddr),
			zap.String("url", r.URL.Path),
			zap.Duration("work_time", time.Since(start)),
		)
	})
}
