package observability

import (
	"log/slog"
	"net/http"
)

// StartMetricsServer starts a simple HTTP server for Prometheus scraping.
func StartMetricsServer(port string) {
	mux := http.NewServeMux()
	mux.Handle("/metrics", MetricsHandler())
	go func() {
		slog.Info("metrics_server_started", "port", port)
		if err := http.ListenAndServe(":"+port, mux); err != nil {
			slog.Error("metrics_server_failed", "error", err)
		}
	}()
}
