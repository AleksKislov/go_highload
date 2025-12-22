package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type MetricsService struct {
	RequestsTotal      *prometheus.CounterVec
	RequestDuration    *prometheus.HistogramVec
	AnomaliesTotal     prometheus.Counter
	RollingAverage     prometheus.Gauge
	ActiveConnections  prometheus.Gauge
}

func NewMetricsService() *MetricsService {
	return &MetricsService{
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "endpoint", "status"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request latencies in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "endpoint"},
		),
		AnomaliesTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "anomalies_detected_total",
				Help: "Total number of anomalies detected",
			},
		),
		RollingAverage: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "metrics_rolling_average",
				Help: "Current rolling average of RPS metrics",
			},
		),
		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_connections",
				Help: "Number of active HTTP connections",
			},
		),
	}
}

// MetricsMiddleware - middleware для подсчёта метрик запросов
func (m *MetricsService) MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Увеличиваем счётчик активных соединений
		m.ActiveConnections.Inc()
		defer m.ActiveConnections.Dec()

		// Обёртка для ResponseWriter для захвата статуса
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)

		// Записываем метрики
		duration := time.Since(start).Seconds()
		m.RequestDuration.WithLabelValues(r.Method, r.URL.Path).Observe(duration)
		m.RequestsTotal.WithLabelValues(r.Method, r.URL.Path, strconv.Itoa(wrapped.statusCode)).Inc()
	})
}

// responseWriter обёртка для захвата статус-кода
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
