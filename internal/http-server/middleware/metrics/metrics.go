package metrics

import (
	"2025_2_a4code/internal/lib/metrics"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewStatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{w, http.StatusOK}
}

func (srw *statusResponseWriter) WriteHeader(code int) {
	srw.statusCode = code
	srw.ResponseWriter.WriteHeader(code)
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Обертка для перехвата статуса
		sw := NewStatusResponseWriter(w)

		next.ServeHTTP(sw, r)

		duration := time.Since(start).Seconds()

		path := sanitizePath(r.URL.Path)

		statusStr := strconv.Itoa(sw.statusCode)

		// Запись метрик
		metrics.HttpRequestsTotal.WithLabelValues(r.Method, path, statusStr).Inc()
		metrics.HttpRequestDuration.WithLabelValues(r.Method, path).Observe(duration)
	})
}

func sanitizePath(path string) string {
	if strings.HasPrefix(path, "/messages/") && len(path) > 10 {
		return "/messages/:id"
	}
	return path
}
