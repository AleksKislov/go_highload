package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go-service/internal/analytics"
	"go-service/internal/cache"
	"go-service/internal/metrics"
)

type Handler struct {
	cache     *cache.RedisCache
	analytics *analytics.AnalyticsService
	metrics   *metrics.MetricsService
}

type MetricData struct {
	Timestamp int64   `json:"timestamp"`
	CPU       float64 `json:"cpu"`
	RPS       float64 `json:"rps"`
	Memory    float64 `json:"memory,omitempty"`
	Latency   float64 `json:"latency,omitempty"`
}

type AnalyticsResponse struct {
	RollingAverage  float64   `json:"rolling_average"`
	ZScore          float64   `json:"z_score"`
	IsAnomaly       bool      `json:"is_anomaly"`
	TotalMetrics    int       `json:"total_metrics"`
	AnomaliesCount  int       `json:"anomalies_count"`
	Timestamp       time.Time `json:"timestamp"`
	WindowSize      int       `json:"window_size"`
}

type StatsResponse struct {
	TotalRequests   int64     `json:"total_requests"`
	TotalAnomalies  int64     `json:"total_anomalies"`
	RollingAverage  float64   `json:"rolling_average"`
	Uptime          string    `json:"uptime"`
	Timestamp       time.Time `json:"timestamp"`
}

func NewHandler(cache *cache.RedisCache, analytics *analytics.AnalyticsService, metrics *metrics.MetricsService) *Handler {
	return &Handler{
		cache:     cache,
		analytics: analytics,
		metrics:   metrics,
	}
}

// ReceiveMetrics принимает метрики от клиентов
func (h *Handler) ReceiveMetrics(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	var metricData MetricData
	if err := json.NewDecoder(r.Body).Decode(&metricData); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Валидация данных
	if metricData.RPS < 0 || metricData.CPU < 0 || metricData.CPU > 100 {
		http.Error(w, "Invalid metric values", http.StatusBadRequest)
		return
	}

	// Если timestamp не указан, используем текущее время
	if metricData.Timestamp == 0 {
		metricData.Timestamp = time.Now().Unix()
	}

	// Кэширование в Redis
	cacheKey := fmt.Sprintf("metric:%d", metricData.Timestamp)
	if err := h.cache.Set(r.Context(), cacheKey, metricData, 5*time.Minute); err != nil {
		// Логируем ошибку, но продолжаем работу
		fmt.Printf("Cache error: %v\n", err)
	}

	// Отправка метрики в аналитический движок
	metric := analytics.Metric{
		Timestamp: time.Unix(metricData.Timestamp, 0),
		Value:     metricData.RPS,
		CPU:       metricData.CPU,
		Metadata: map[string]interface{}{
			"memory":  metricData.Memory,
			"latency": metricData.Latency,
		},
	}

	h.analytics.SendMetric(metric)

	// Обновление Prometheus метрик
	duration := time.Since(startTime).Seconds()
	h.metrics.RequestDuration.WithLabelValues("POST", "/metrics-data").Observe(duration)

	// Ответ клиенту
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "accepted",
		"message": "Metric received successfully",
	})
}

// GetAnalytics возвращает текущую аналитику
func (h *Handler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	// Попытка получить из кэша
	var cachedResponse AnalyticsResponse
	cacheKey := "analytics:latest"
	
	if err := h.cache.Get(r.Context(), cacheKey, &cachedResponse); err == nil {
		// Если кэш валиден (не старше 5 секунд), возвращаем его
		if time.Since(cachedResponse.Timestamp) < 5*time.Second {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			json.NewEncoder(w).Encode(cachedResponse)
			return
		}
	}

	// Получение свежих данных
	stats := h.analytics.GetStats()
	
	response := AnalyticsResponse{
		RollingAverage: stats.RollingAverage,
		ZScore:         stats.CurrentZScore,
		IsAnomaly:      stats.IsAnomaly,
		TotalMetrics:   stats.TotalMetrics,
		AnomaliesCount: stats.AnomaliesDetected,
		Timestamp:      time.Now(),
		WindowSize:     h.analytics.GetWindowSize(),
	}

	// Сохранение в кэш
	h.cache.Set(r.Context(), cacheKey, response, 10*time.Second)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	json.NewEncoder(w).Encode(response)
}

// GetStats возвращает общую статистику
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.analytics.GetStats()
	
	response := StatsResponse{
		TotalRequests:  stats.TotalMetrics,
		TotalAnomalies: stats.AnomaliesDetected,
		RollingAverage: stats.RollingAverage,
		Uptime:         time.Since(h.analytics.GetStartTime()).String(),
		Timestamp:      time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HealthCheck проверяет состояние сервиса
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "go-metrics-service",
	}

	// Проверка Redis
	if err := h.cache.Ping(r.Context()); err != nil {
		health["redis"] = "unhealthy"
		health["redis_error"] = err.Error()
	} else {
		health["redis"] = "healthy"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}
