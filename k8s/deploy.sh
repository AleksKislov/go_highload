#!/bin/bash

set -e

echo "=== Go Service Kubernetes Deployment Script ==="
echo ""

# Проверка Minikube
if ! minikube status &> /dev/null; then
    echo "❌ Minikube is not running!"
    echo "Start it with: minikube start --cpus=2 --memory=4g"
    exit 1
fi

echo "✓ Minikube is running"
echo ""

# Проверка Docker образа
echo "=== Checking Docker image ==="
if ! docker images | grep -q "go-service.*v1.0"; then
    echo "⚠️  Docker image 'go-service:v1.0' not found"
    echo "Building image..."
    cd /vagrant/go-service
    docker build -t go-service:v1.0 .
fi

echo "✓ Docker image exists"
echo ""

# Загрузка образа в Minikube
echo "=== Loading image to Minikube ==="
minikube image load go-service:v1.0
echo "✓ Image loaded to Minikube"
echo ""

# Включение Ingress
echo "=== Enabling Ingress ==="
minikube addons enable ingress
echo "✓ Ingress enabled"
echo ""

# Включение Metrics Server для HPA
echo "=== Enabling Metrics Server ==="
minikube addons enable metrics-server
echo "✓ Metrics Server enabled"
echo ""

# Применение манифестов
echo "=== Applying Kubernetes manifests ==="

cd /vagrant/go-service/k8s

# 1. ConfigMap
echo "  → Applying ConfigMap..."
kubectl apply -f configmap.yaml

# 2. Redis
echo "  → Deploying Redis..."
kubectl apply -f redis-deployment.yaml
kubectl apply -f redis-service.yaml

# Ждем Redis
echo "  → Waiting for Redis to be ready..."
kubectl wait --for=condition=ready pod -l app=redis --timeout=120s

# 3. Go Service
echo "  → Deploying Go Service..."
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml

# Ждем Go Service
echo "  → Waiting for Go Service to be ready..."
kubectl wait --for=condition=ready pod -l app=go-service --timeout=120s

# 4. HPA
echo "  → Applying HPA..."
kubectl apply -f hpa.yaml

# 5. Ingress
echo "  → Applying Ingress..."
kubectl apply -f ingress.yaml

echo ""
echo "=== Deployment Summary ==="
echo ""

# Показываем статус
echo "Pods:"
kubectl get pods
echo ""

echo "Services:"
kubectl get svc
echo ""

echo "HPA:"
kubectl get hpa
echo ""

echo "Ingress:"
kubectl get ingress
echo ""

# Получаем URL
MINIKUBE_IP=$(minikube ip)
echo "=== Access Information ==="
echo ""
echo "Service URL (NodePort): http://${MINIKUBE_IP}:30080"
echo "Service URL (Ingress):  http://go-service.local"
echo ""
echo "Add to /etc/hosts on your HOST machine:"
echo "  ${MINIKUBE_IP} go-service.local"
echo ""

echo "Health check:"
echo "  curl http://${MINIKUBE_IP}:30080/health"
echo ""

echo "Send metric:"
echo '  curl -X POST http://'"${MINIKUBE_IP}"':30080/metrics-data \\'
echo '    -H "Content-Type: application/json" \\'
echo "    -d '{\"timestamp\":$(date +%s),\"cpu\":45.5,\"rps\":105.3}'"
echo ""

echo "Get analytics:"
echo "  curl http://${MINIKUBE_IP}:30080/analyze"
echo ""

echo "Prometheus metrics:"
echo "  curl http://${MINIKUBE_IP}:30080/metrics"
echo ""

echo "=== Port Forwarding Commands ==="
echo "For direct access from host machine:"
echo "  kubectl port-forward svc/go-service 8080:8080"
echo ""

echo "✓ Deployment complete!"
