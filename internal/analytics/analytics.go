package analytics

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/yourname/go-service/internal/metrics"
)

type Metric struct {
	Timestamp time.Time
	Value     float64
	CPU       float64
	Metadata  map[string]interface{}
}

type AnalyticsService struct {
	windowSize      int
	anomalyZScore   float64
	metricsBuffer   []float64
	allMetrics      []Metric
	mu              sync.RWMutex
	metricsChan     chan Metric
	metricsService  *metrics.MetricsService
	
	// Статистика
	totalMetrics     int64
	anomaliesCount   int64
	rollingAvg       float64
	currentZScore    float64
	isCurrentAnomaly bool
	startTime        time.Time
}

type Stats struct {
	RollingAverage    float64
	CurrentZScore     float64
	IsAnomaly         bool
	TotalMetrics      int64
	AnomaliesDetected int64
}

func NewAnalyticsService(windowSize int, anomalyZScore float64, metricsService *metrics.MetricsService) *AnalyticsService {
	return &AnalyticsService{
		windowSize:     windowSize,
		anomalyZScore:  anomalyZScore,
		metricsBuffer:  make([]float64, 0, windowSize),
		allMetrics:     make([]Metric, 0),
		metricsChan:    make(chan Metric, 1000),
		metricsService: metricsService,
		startTime:      time.Now(),
	}
}

// Start запускает фоновую обработку метрик
func (a *AnalyticsService) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case metric := <-a.metricsChan:
			a.processMetric(metric)
		}
	}
}

// SendMetric отправляет метрику на обработку
func (a *AnalyticsService) SendMetric(metric Metric) {
	select {
	case a.metricsChan <- metric:
	default:
		// Канал переполнен, пропускаем метрику
	}
}

// processMetric обрабатывает метрику: вычисляет rolling average и z-score
func (a *AnalyticsService) processMetric(metric Metric) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Добавляем метрику в буфер
	a.metricsBuffer = append(a.metricsBuffer, metric.Value)
	a.allMetrics = append(a.allMetrics, metric)
	a.totalMetrics++

	// Ограничиваем размер буфера
	if len(a.metricsBuffer) > a.windowSize {
		a.metricsBuffer = a.metricsBuffer[1:]
	}

	// Вычисляем rolling average
	a.rollingAvg = a.calculateRollingAverage()

	// Обновляем Prometheus метрику
	a.metricsService.RollingAverage.Set(a.rollingAvg)

	// Вычисляем z-score и проверяем на аномалии
	if len(a.metricsBuffer) >= 10 { // Минимум 10 точек для статистики
		a.currentZScore = a.calculateZScore(metric.Value)
		a.isCurrentAnomaly = math.Abs(a.currentZScore) > a.anomalyZScore

		if a.isCurrentAnomaly {
			a.anomaliesCount++
			a.metricsService.AnomaliesTotal.Inc()
		}
	}
}

// calculateRollingAverage вычисляет скользящее среднее
func (a *AnalyticsService) calculateRollingAverage() float64 {
	if len(a.metricsBuffer) == 0 {
		return 0
	}

	sum := 0.0
	for _, value := range a.metricsBuffer {
		sum += value
	}
	return sum / float64(len(a.metricsBuffer))
}

// calculateZScore вычисляет z-score для обнаружения аномалий
func (a *AnalyticsService) calculateZScore(value float64) float64 {
	if len(a.metricsBuffer) < 2 {
		return 0
	}

	// Вычисляем среднее
	mean := a.rollingAvg

	// Вычисляем стандартное отклонение
	varianceSum := 0.0
	for _, v := range a.metricsBuffer {
		diff := v - mean
		varianceSum += diff * diff
	}
	stdDev := math.Sqrt(varianceSum / float64(len(a.metricsBuffer)))

	// Избегаем деления на ноль
	if stdDev < 0.0001 {
		return 0
	}

	// Z-score = (value - mean) / stdDev
	return (value - mean) / stdDev
}

// GetStats возвращает текущую статистику
func (a *AnalyticsService) GetStats() Stats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return Stats{
		RollingAverage:    a.rollingAvg,
		CurrentZScore:     a.currentZScore,
		IsAnomaly:         a.isCurrentAnomaly,
		TotalMetrics:      a.totalMetrics,
		AnomaliesDetected: a.anomaliesCount,
	}
}

// GetWindowSize возвращает размер окна
func (a *AnalyticsService) GetWindowSize() int {
	return a.windowSize
}

// GetStartTime возвращает время старта сервиса
func (a *AnalyticsService) GetStartTime() time.Time {
	return a.startTime
}

// PredictNext предсказывает следующее значение на основе rolling average
// Простая реализация: возвращает текущее среднее как прогноз
func (a *AnalyticsService) PredictNext() float64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	return a.rollingAvg
}

// GetRecentMetrics возвращает последние N метрик
func (a *AnalyticsService) GetRecentMetrics(n int) []Metric {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if len(a.allMetrics) == 0 {
		return []Metric{}
	}

	start := len(a.allMetrics) - n
	if start < 0 {
		start = 0
	}

	result := make([]Metric, len(a.allMetrics[start:]))
	copy(result, a.allMetrics[start:])
	return result
}
