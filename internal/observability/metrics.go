package observability

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Kafka metrics
	KafkaMessagesProduced = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_produced_total",
			Help: "Total Kafka messages produced",
		},
		[]string{"topic"},
	)
	KafkaMessagesConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_messages_consumed_total",
			Help: "Total Kafka messages consumed",
		},
		[]string{"topic", "consumer_group"},
	)
	KafkaProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kafka_message_processing_seconds",
			Help:    "Kafka message processing duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic", "consumer_group"},
	)
	KafkaProcessingErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kafka_message_processing_errors_total",
			Help: "Total Kafka message processing errors",
		},
		[]string{"topic", "consumer_group"},
	)

	// Business metrics
	BetsPlacedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "bets_placed_total",
			Help: "Total bets placed",
		},
	)
	BetsRejectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bets_rejected_total",
			Help: "Total bets rejected",
		},
		[]string{"reason"},
	)
	BetsSettledTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "bets_settled_total",
			Help: "Total bets settled",
		},
		[]string{"result"},
	)
	OddsUpdatesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "odds_updates_total",
			Help: "Total odds updates processed",
		},
	)
	OddsChangePercent = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "odds_change_percent",
			Help:    "Odds change percentage distribution",
			Buckets: []float64{1, 2, 5, 10, 20, 50},
		},
	)
	FraudAlertsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fraud_alerts_total",
			Help: "Total fraud alerts triggered",
		},
		[]string{"alert_type"},
	)
	MarketSuspensionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "market_suspensions_total",
			Help: "Total market suspensions",
		},
	)
	GameEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "game_events_total",
			Help: "Total game events produced",
		},
		[]string{"type"},
	)

	// API metrics
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"method", "path", "status"},
	)
)

func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// MetricsMiddleware wraps an http.Handler with request duration and count tracking.
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(rw.status)
		HTTPRequestDuration.WithLabelValues(r.Method, r.URL.Path, status).Observe(duration)
		HTTPRequestsTotal.WithLabelValues(r.Method, r.URL.Path, status).Inc()
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
