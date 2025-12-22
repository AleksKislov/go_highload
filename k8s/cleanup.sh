#!/bin/bash

echo "=== Cleanup Kubernetes Resources ==="
echo ""

read -p "Are you sure you want to delete all resources? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "Cleanup cancelled"
    exit 0
fi

echo ""
echo "Deleting Go Service resources..."
kubectl delete -f /vagrant/go-service/k8s/deployment.yaml --ignore-not-found=true
kubectl delete -f /vagrant/go-service/k8s/service.yaml --ignore-not-found=true
kubectl delete -f /vagrant/go-service/k8s/configmap.yaml --ignore-not-found=true
kubectl delete -f /vagrant/go-service/k8s/hpa.yaml --ignore-not-found=true
kubectl delete -f /vagrant/go-service/k8s/ingress.yaml --ignore-not-found=true

echo "Deleting Redis..."
kubectl delete -f /vagrant/go-service/k8s/redis-deployment.yaml --ignore-not-found=true
kubectl delete -f /vagrant/go-service/k8s/redis-service.yaml --ignore-not-found=true

echo ""
read -p "Delete Prometheus and Grafana? (yes/no): " confirm_monitoring

if [ "$confirm_monitoring" == "yes" ]; then
    echo "Uninstalling Prometheus..."
    helm uninstall prometheus --namespace default 2>/dev/null || true
    
    echo "Uninstalling Grafana..."
    helm uninstall grafana --namespace default 2>/dev/null || true
fi

echo ""
echo "Remaining resources:"
kubectl get all
echo ""
echo "✓ Cleanup complete!"
