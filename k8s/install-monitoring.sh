#!/bin/bash

set -e

echo "=== Installing Prometheus and Grafana ==="
echo ""

# Добавление репозиториев Helm
echo "=== Adding Helm repositories ==="
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
echo "✓ Helm repos added"
echo ""

# Установка Prometheus
echo "=== Installing Prometheus ==="
helm install prometheus prometheus-community/prometheus \
  --namespace default \
  --set server.persistentVolume.enabled=false \
  --set alertmanager.persistentVolume.enabled=false \
  --set pushgateway.enabled=false \
  --set server.service.type=NodePort \
  --set server.service.nodePort=30090

echo "✓ Prometheus installed"
echo ""

# Ожидание готовности Prometheus
echo "Waiting for Prometheus to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus --timeout=180s
echo ""

# Установка Grafana
echo "=== Installing Grafana ==="
helm install grafana grafana/grafana \
  --namespace default \
  --set persistence.enabled=false \
  --set service.type=NodePort \
  --set service.nodePort=30300 \
  --set adminPassword=admin

echo "✓ Grafana installed"
echo ""

# Ожидание готовности Grafana
echo "Waiting for Grafana to be ready..."
kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana --timeout=180s
echo ""

# Получение IP Minikube
MINIKUBE_IP=$(minikube ip)

echo "=== Installation Complete ==="
echo ""
echo "Prometheus:"
echo "  URL: http://${MINIKUBE_IP}:30090"
echo "  Port-forward: kubectl port-forward svc/prometheus-server 9090:80"
echo ""

echo "Grafana:"
echo "  URL: http://${MINIKUBE_IP}:30300"
echo "  Port-forward: kubectl port-forward svc/grafana 3000:80"
echo "  Username: admin"
echo "  Password: admin"
echo ""

echo "Configure Grafana:"
echo "  1. Add Prometheus data source:"
echo "     URL: http://prometheus-server:80"
echo "  2. Import dashboard or create custom queries"
echo ""

echo "Example Prometheus queries for Go service:"
echo "  - Rate of requests: rate(http_requests_total[1m])"
echo "  - Anomalies: anomalies_detected_total"
echo "  - Rolling average: metrics_rolling_average"
echo "  - Request duration: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))"
echo ""
