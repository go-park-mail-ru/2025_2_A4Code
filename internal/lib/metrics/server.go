package metrics

import (
	"log/slog"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func StartMetricsServer(port string, log *slog.Logger) {
	http.Handle("/metrics", promhttp.Handler())
	addr := ":" + port
	log.Info("Starting metrics server on " + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Error("Failed to start metrics server: " + err.Error())
	}
}
