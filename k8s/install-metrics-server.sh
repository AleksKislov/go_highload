#!/bin/bash
# Скрипт для установки Metrics Server в Minikube

echo "=== Installing Metrics Server for HPA ==="

# В Minikube можно просто включить addon
minikube addons enable metrics-server

# Или установить вручную
# kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

echo "Waiting for metrics-server to be ready..."
kubectl wait --for=condition=ready pod -l k8s-app=metrics-server -n kube-system --timeout=120s

echo ""
echo "✓ Metrics Server installed successfully!"
echo ""
echo "Check metrics with:"
echo "  kubectl top nodes"
echo "  kubectl top pods"
