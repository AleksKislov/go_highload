# Kubernetes Manifests

Манифесты для развертывания Go сервиса в Kubernetes (Minikube).

## Структура файлов

```
k8s/
├── configmap.yaml              # Конфигурация сервиса
├── deployment.yaml             # Deployment Go сервиса
├── service.yaml                # Services (ClusterIP + NodePort)
├── hpa.yaml                    # HorizontalPodAutoscaler
├── ingress.yaml                # Ingress для внешнего доступа
├── redis-deployment.yaml       # Redis Deployment
├── redis-service.yaml          # Redis Service
├── deploy.sh                   # Скрипт автоматического развертывания
├── install-monitoring.sh       # Установка Prometheus + Grafana
├── install-metrics-server.sh  # Установка Metrics Server
├── cleanup.sh                  # Очистка ресурсов
└── README.md                   # Этот файл
```

## Быстрый старт

### Вариант 1: Автоматическое развертывание (рекомендуется)

```bash
cd <repo_directory>/k8s

# Сделать скрипты исполняемыми
chmod +x *.sh

# Развернуть все
./deploy.sh
```

### Вариант 2: Ручное развертывание

```bash
# 1. Старт Minikube
minikube start --cpus=2 --memory=4g

# 2. Включить необходимые addons
minikube addons enable ingress
minikube addons enable metrics-server

# 3. Собрать и загрузить образ
cd /vagrant/go-service
docker build -t go-service:v1.0 .
minikube image load go-service:v1.0

# 4. Применить манифесты по порядку
cd k8s
kubectl apply -f configmap.yaml
kubectl apply -f redis-deployment.yaml
kubectl apply -f redis-service.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
kubectl apply -f hpa.yaml
kubectl apply -f ingress.yaml

# 5. Проверить развертывание
kubectl get all
```

## Описание компонентов

### ConfigMap
Содержит конфигурацию:
- `SERVER_PORT`: порт сервера (8080)
- `REDIS_ADDR`: адрес Redis (redis-master:6379)
- `WINDOW_SIZE`: размер окна для Rolling Average (50)
- `ANOMALY_ZSCORE`: порог Z-Score для аномалий (2.0)

### Deployment
- **Replicas**: 2 (минимум)
- **Image**: go-service:v1.0
- **Resources**:
  - Request: 100m CPU, 128Mi Memory
  - Limit: 500m CPU, 512Mi Memory
- **Probes**: Liveness и Readiness на `/health`

### Service
Два типа сервисов:
1. **ClusterIP** (go-service): Внутренний доступ
2. **NodePort** (go-service-nodeport): Внешний доступ на порту 30080

### HPA (HorizontalPodAutoscaler)
- **Min replicas**: 2
- **Max replicas**: 5
- **Target CPU**: 70%
- **Target Memory**: 80%
- Auto-scaling при увеличении нагрузки

### Ingress
- **Host**: go-service.local
- **Paths**: /, /metrics, /analyze
- **Class**: nginx

### Redis
- **Image**: redis:7-alpine
- **Persistence**: emptyDir (не сохраняется)
- **Service**: redis-master на порту 6379

## Доступ к сервису

### Через NodePort

```bash
MINIKUBE_IP=$(minikube ip)

# Health check
curl http://${MINIKUBE_IP}:30080/health

# Отправка метрики
curl -X POST http://${MINIKUBE_IP}:30080/metrics-data \
  -H "Content-Type: application/json" \
  -d '{"timestamp":1234567890,"cpu":45.5,"rps":105.3}'

# Получение аналитики
curl http://${MINIKUBE_IP}:30080/analyze

# Статистика
curl http://${MINIKUBE_IP}:30080/stats

# Prometheus метрики
curl http://${MINIKUBE_IP}:30080/metrics
```

### Через Port-Forward

```bash
# Прокинуть порт на локальную машину
kubectl port-forward svc/go-service 8080:8080

# Использовать localhost
curl http://localhost:8080/health
```

### Через Ingress

```bash
# Получить IP Minikube
minikube ip

# Добавить в /etc/hosts на HOST машине
echo "$(minikube ip) go-service.local" | sudo tee -a /etc/hosts

# Использовать домен
curl http://go-service.local/health
```

## Мониторинг

### Установка Prometheus и Grafana

```bash
./install-monitoring.sh
```

После установки:
- **Prometheus**: http://$(minikube ip):30090
- **Grafana**: http://$(minikube ip):30300 (admin/admin)

### Метрики в Prometheus

Запросы для мониторинга:

```promql
# RPS (requests per second)
rate(http_requests_total[1m])

# Количество аномалий
anomalies_detected_total

# Rolling average
metrics_rolling_average

# P99 latency
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# Активные соединения
active_connections

# CPU usage by pod
container_cpu_usage_seconds_total{pod=~"go-service.*"}
```

### Настройка Grafana

1. Открыть Grafana (admin/admin)
2. Add Data Source → Prometheus
3. URL: `http://prometheus-server:80`
4. Save & Test
5. Create Dashboard с запросами выше

## Нагрузочное тестирование

### С помощью Locust

```bash
# Port-forward для сервиса
kubectl port-forward svc/go-service 8080:8080 &

# Запуск Locust
cd /vagrant/go-service/tests
locust -f locustfile.py --host=http://localhost:8080

# Открыть в браузере на HOST машине: http://localhost:8089
# Настроить: 500 users, spawn rate 10
```

### Мониторинг HPA

```bash
# Смотреть метрики в реальном времени
watch kubectl get hpa

# Смотреть количество подов
watch kubectl get pods

# Метрики подов
kubectl top pods
```

## Проверка работы

### 1. Проверка подов

```bash
kubectl get pods
# Ожидается: 2+ пода go-service в статусе Running
```

### 2. Проверка логов

```bash
kubectl logs -f deployment/go-service
```

### 3. Проверка HPA

```bash
kubectl get hpa go-service-hpa
# Смотрим на TARGETS (должно быть меньше 70%/80%)
```

### 4. Тестирование масштабирования

```bash
# Запустить нагрузку
kubectl port-forward svc/go-service 8080:8080 &
cd /vagrant/go-service/tests
locust -f locustfile.py --host=http://localhost:8080 --headless \
  -u 500 -r 50 -t 5m

# В другом терминале смотреть масштабирование
watch kubectl get pods
```

## Troubleshooting

### Pod не запускается

```bash
# Описание пода
kubectl describe pod <pod-name>

# Логи
kubectl logs <pod-name>

# Events
kubectl get events --sort-by='.lastTimestamp'
```

### Образ не найден

```bash
# Проверить образы в Minikube
minikube ssh docker images

# Перезагрузить образ
minikube image load go-service:v1.0
```

### HPA не работает

```bash
# Проверить Metrics Server
kubectl get apiservice v1beta1.metrics.k8s.io -o yaml

# Включить если отключен
minikube addons enable metrics-server

# Проверить метрики
kubectl top nodes
kubectl top pods
```

### Redis недоступен

```bash
# Проверить статус
kubectl get pods -l app=redis

# Логи Redis
kubectl logs deployment/redis

# Проверить сервис
kubectl get svc redis-master
```

## Очистка ресурсов

```bash
# Удалить все ресурсы проекта
./cleanup.sh

# Или вручную
kubectl delete -f .

# Удалить Prometheus и Grafana
helm uninstall prometheus
helm uninstall grafana

# Остановить Minikube
minikube stop

# Удалить Minikube (полная очистка)
minikube delete
```

## Полезные команды

```bash
# Все ресурсы
kubectl get all

# Описание deployment
kubectl describe deployment go-service

# Масштабирование вручную
kubectl scale deployment go-service --replicas=3

# Рестарт подов
kubectl rollout restart deployment go-service

# Откат deployment
kubectl rollout undo deployment go-service

# История rollout
kubectl rollout history deployment go-service

# Port-forward для всех сервисов
kubectl port-forward svc/go-service 8080:8080 &
kubectl port-forward svc/prometheus-server 9090:80 &
kubectl port-forward svc/grafana 3000:80 &

# Dashboard Minikube
minikube dashboard
```
