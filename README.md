# Go Metrics Service with Analytics

Сервис для обработки потоковых метрик с аналитикой на основе Rolling Average и Z-Score для детекции аномалий.

## Архитектура

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   Clients   │─────▶│  Go Service  │─────▶│    Redis    │
│  (Locust)   │      │   (HTTP API) │      │   (Cache)   │
└─────────────┘      └──────────────┘      └─────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │  Prometheus  │
                     │  (Metrics)   │
                     └──────────────┘
                            │
                            ▼
                     ┌──────────────┐
                     │   Grafana    │
                     │ (Dashboard)  │
                     └──────────────┘
```

## Компоненты

### 1. Go Service
- **HTTP API** для приема метрик
- **Rolling Average** для сглаживания (окно 50 событий)
- **Z-Score** для детекции аномалий (threshold = 2.0)
- **Redis** для кэширования
- **Prometheus** метрики

### 2. Endpoints

- `POST /metrics-data` - прием метрик
- `GET /analyze` - получение аналитики
- `GET /stats` - общая статистика
- `GET /health` - health check
- `GET /metrics` - Prometheus метрики

### 3. Метрики

Формат входных данных:
```json
{
  "timestamp": 1234567890,
  "cpu": 45.5,
  "rps": 105.3,
  "memory": 50.2,
  "latency": 25.1
}
```

## Быстрый старт

### 1. Инициализация проекта

```bash
cd <repo_directory> 
go mod download
```

### 2. Локальный запуск (без Docker)

```bash
# Запустить Redis
docker run -d -p 6379:6379 --name redis redis:alpine

# Запустить сервис
go run cmd/main.go

# В другом терминале - тестовая отправка метрик
chmod +x tests/generate_metrics.py
python3 tests/generate_metrics.py
```

### 3. Сборка Docker образа

```bash
docker build -t go-service:v1.0 .
```

### 4. Запуск в Minikube

```bash
# Старт Minikube
minikube start --cpus=2 --memory=4g

# Загрузка образа в Minikube
minikube image load go-service:v1.0

# Применение манифестов
kubectl apply -f k8s/
```

## Нагрузочное тестирование

### С помощью Locust

```bash
# Запуск Locust UI
cd tests
locust -f locustfile.py --host=http://localhost:8080

# Открыть в браузере: http://localhost:8089
# Настроить: 500 users, spawn rate 10
```

### С помощью тестового скрипта

```bash
python3 tests/generate_metrics.py
```

## Структура проекта

```
go-service/
├── cmd/
│   └── main.go              # Точка входа
├── internal/
│   ├── handlers/
│   │   └── handlers.go      # HTTP обработчики
│   ├── analytics/
│   │   └── analytics.go     # Аналитика (rolling avg, z-score)
│   ├── cache/
│   │   └── cache.go         # Redis кэш
│   └── metrics/
│       └── metrics.go       # Prometheus метрики
├── k8s/                     # Kubernetes манифесты
├── tests/                   # Тесты и нагрузка
│   ├── locustfile.py
│   └── generate_metrics.py
├── Dockerfile
├── go.mod
└── README.md
```

## Алгоритмы аналитики

### Rolling Average
Скользящее среднее по окну из 50 последних значений:
```
avg = (sum of last 50 values) / 50
```

### Z-Score для аномалий
```
z-score = (value - mean) / stdDev
anomaly if |z-score| > 2.0
```

## Мониторинг

### Prometheus метрики

- `http_requests_total` - общее количество запросов
- `http_request_duration_seconds` - latency запросов
- `anomalies_detected_total` - количество обнаруженных аномалий
- `metrics_rolling_average` - текущее скользящее среднее
- `active_connections` - активные соединения

### Grafana Dashboard

После развертывания доступна по адресу: `http://localhost:3000`

## Требования

- Go 1.22+
- Docker
- Kubernetes (Minikube)
- Redis
- Python 3.8+ (для тестов)

## Переменные окружения

- `SERVER_PORT` - порт сервера (default: 8080)
- `REDIS_ADDR` - адрес Redis (default: localhost:6379)
- `REDIS_PASSWORD` - пароль Redis (default: "")

## Производительность

- **Latency**: < 50ms (p99)
- **Throughput**: 500+ RPS
- **Cache hit rate**: > 80%
- **Anomaly detection accuracy**: > 70%
- **False positive rate**: < 10%

