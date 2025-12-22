"""
Locust файл для нагрузочного тестирования Go сервиса
Запуск: locust -f locustfile.py --host=http://localhost:8080
"""
import random
import time
from locust import HttpUser, task, between

class MetricsUser(HttpUser):
    wait_time = between(0.01, 0.05)  # Задержка между запросами 10-50ms
    
    def on_start(self):
        """Выполняется при старте каждого пользователя"""
        # Проверяем health
        self.client.get("/health")
    
    @task(10)
    def send_normal_metric(self):
        """Отправка нормальной метрики (90% запросов)"""
        metric = {
            "timestamp": int(time.time()),
            "cpu": random.uniform(30, 70),
            "rps": random.uniform(80, 120),
            "memory": random.uniform(40, 60),
            "latency": random.uniform(10, 30)
        }
        
        with self.client.post(
            "/metrics-data",
            json=metric,
            catch_response=True
        ) as response:
            if response.status_code == 202:
                response.success()
            else:
                response.failure(f"Got status code {response.status_code}")
    
    @task(1)
    def send_anomaly_metric(self):
        """Отправка аномальной метрики (10% запросов)"""
        metric = {
            "timestamp": int(time.time()),
            "cpu": random.uniform(85, 98),
            "rps": random.uniform(300, 500),
            "memory": random.uniform(80, 95),
            "latency": random.uniform(100, 200)
        }
        
        with self.client.post(
            "/metrics-data",
            json=metric,
            catch_response=True
        ) as response:
            if response.status_code == 202:
                response.success()
            else:
                response.failure(f"Got status code {response.status_code}")
    
    @task(2)
    def get_analytics(self):
        """Получение аналитики (20% запросов)"""
        with self.client.get("/analyze", catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Got status code {response.status_code}")
    
    @task(1)
    def get_stats(self):
        """Получение статистики (10% запросов)"""
        self.client.get("/stats")
    
    @task(1)
    def health_check(self):
        """Health check (10% запросов)"""
        self.client.get("/health")
