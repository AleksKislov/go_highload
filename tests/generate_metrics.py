#!/usr/bin/env python3
"""
Скрипт для генерации синтетических метрик и тестирования сервиса
"""
import json
import random
import time
import requests
from datetime import datetime

SERVICE_URL = "http://localhost:8080"

def generate_normal_metric():
    """Генерирует нормальную метрику"""
    return {
        "timestamp": int(time.time()),
        "cpu": random.uniform(30, 70),
        "rps": random.uniform(80, 120),
        "memory": random.uniform(40, 60),
        "latency": random.uniform(10, 30)
    }

def generate_anomaly_metric():
    """Генерирует аномальную метрику"""
    return {
        "timestamp": int(time.time()),
        "cpu": random.uniform(85, 98),
        "rps": random.uniform(300, 500),  # Аномально высокий RPS
        "memory": random.uniform(80, 95),
        "latency": random.uniform(100, 200)
    }

def send_metric(metric):
    """Отправляет метрику на сервер"""
    try:
        response = requests.post(
            f"{SERVICE_URL}/metrics-data",
            json=metric,
            timeout=5
        )
        return response.status_code == 202
    except Exception as e:
        print(f"Error sending metric: {e}")
        return False

def main():
    print("Starting metric generator...")
    print(f"Target: {SERVICE_URL}")
    print("Press Ctrl+C to stop\n")
    
    sent_count = 0
    error_count = 0
    anomaly_count = 0
    
    try:
        while True:
            # 90% нормальных метрик, 10% аномалий
            if random.random() < 0.9:
                metric = generate_normal_metric()
            else:
                metric = generate_anomaly_metric()
                anomaly_count += 1
            
            if send_metric(metric):
                sent_count += 1
                if sent_count % 10 == 0:
                    print(f"Sent: {sent_count}, Errors: {error_count}, Anomalies: {anomaly_count}")
            else:
                error_count += 1
            
            # Задержка между отправками
            time.sleep(0.1)  # 10 RPS
            
    except KeyboardInterrupt:
        print(f"\n\nStopping...")
        print(f"Total sent: {sent_count}")
        print(f"Total errors: {error_count}")
        print(f"Total anomalies: {anomaly_count}")
        
        # Получаем статистику
        try:
            response = requests.get(f"{SERVICE_URL}/stats", timeout=5)
            if response.status_code == 200:
                stats = response.json()
                print(f"\nServer stats:")
                print(json.dumps(stats, indent=2))
        except:
            pass

if __name__ == "__main__":
    main()
