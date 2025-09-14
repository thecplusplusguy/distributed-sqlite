#!/bin/bash
# ABOUTME: Deployment script for distributed SQLite cluster in Kubernetes
# ABOUTME: Builds Docker image and deploys to local Kubernetes cluster

set -e

echo "🐳 Building Docker image..."
docker build -t distributed-sqlite-node:latest .

echo "📦 Applying Kubernetes manifests..."
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f k8s/deployment.yaml

echo "⏳ Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=distributed-sqlite-node --timeout=300s

echo "✅ Deployment complete!"
echo ""
echo "📊 Cluster status:"
kubectl get pods -l app=distributed-sqlite-node
echo ""
echo "🌐 Services:"
kubectl get svc | grep distributed-sqlite
echo ""
echo "🔗 Access the cluster:"
echo "  External: kubectl port-forward svc/distributed-sqlite-external 8080:80"
echo "  Direct pod: kubectl port-forward distributed-sqlite-nodes-0 8080:8080"