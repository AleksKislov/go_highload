package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/yourname/go-service/internal/analytics"
	"github.com/yourname/go-service/internal/cache"
	"github.com/yourname/go-service/internal/handlers"
	"github.com/yourname/go-service/internal/metrics"
)

type Config struct {
	ServerPort     string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	WindowSize     int
	AnomalyZScore  float64
}

func main() {
	// Конфигурация
	config := Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       0,
		WindowSize:    50,
		AnomalyZScore: 2.0,
	}

	log.Printf("Starting Go Service with config: %+v", config)

	// Инициализация Redis
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     config.RedisAddr,
		Password: config.RedisPassword,
		DB:       config.RedisDB,
	})

	// Проверка подключения к Redis
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v. Continuing without cache...", err)
	} else {
		log.Println("Successfully connected to Redis")
	}

	// Инициализация компонентов
	cacheService := cache.NewRedisCache(redisClient)
	metricsService := metrics.NewMetricsService()
	analyticsService := analytics.NewAnalyticsService(config.WindowSize, config.AnomalyZScore, metricsService)

	// Запуск фоновой обработки аналитики
	go analyticsService.Start(ctx)

	// Регистрация Prometheus метрик
	prometheus.MustRegister(metricsService.RequestsTotal)
	prometheus.MustRegister(metricsService.RequestDuration)
	prometheus.MustRegister(metricsService.AnomaliesTotal)
	prometheus.MustRegister(metricsService.RollingAverage)
	prometheus.MustRegister(metricsService.ActiveConnections)

	// Создание обработчиков
	handler := handlers.NewHandler(cacheService, analyticsService, metricsService)

	// Настройка роутера
	router := mux.NewRouter()
	
	// Эндпоинты для метрик
	router.HandleFunc("/metrics-data", handler.ReceiveMetrics).Methods("POST")
	router.HandleFunc("/health", handler.HealthCheck).Methods("GET")
	router.HandleFunc("/analyze", handler.GetAnalytics).Methods("GET")
	router.HandleFunc("/stats", handler.GetStats).Methods("GET")
	
	// Prometheus метрики
	router.Handle("/metrics", promhttp.Handler()).Methods("GET")

	// Middleware для подсчёта активных соединений
	router.Use(metricsService.MetricsMiddleware)

	// Создание HTTP сервера
	srv := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server starting on port %s", config.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown с таймаутом
	ctxShutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Закрытие Redis соединения
	if err := redisClient.Close(); err != nil {
		log.Printf("Error closing Redis connection: %v", err)
	}

	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
