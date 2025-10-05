#!/bin/bash
# ABOUTME: Script to run integration tests against live k8s deployment
# ABOUTME: Sets up port forwards and runs tests against distributed SQLite cluster

set -e

echo "🚀 Setting up port forwards for integration testing..."

# Kill any existing port forwards
pkill -f "kubectl port-forward" || true
sleep 2

# Start port forwards for all nodes
kubectl port-forward distributed-sqlite-nodes-0 8080:8080 &
PID1=$!
kubectl port-forward distributed-sqlite-nodes-1 8081:8080 &
PID2=$!
kubectl port-forward distributed-sqlite-nodes-2 8082:8080 &
PID3=$!

echo "⏳ Waiting for port forwards to be ready..."
sleep 5

# Test connectivity
echo "🔍 Testing connectivity to all nodes..."
curl -f http://localhost:8080/health || { echo "❌ Node 0 not accessible"; exit 1; }
curl -f http://localhost:8081/health || { echo "❌ Node 1 not accessible"; exit 1; }
curl -f http://localhost:8082/health || { echo "❌ Node 2 not accessible"; exit 1; }

echo "✅ All nodes accessible"

# Function to cleanup on exit
cleanup() {
    echo "🧹 Cleaning up port forwards..."
    kill $PID1 $PID2 $PID3 2>/dev/null || true
    pkill -f "kubectl port-forward" || true
}

# Set trap to cleanup on script exit
trap cleanup EXIT

echo "🧪 Running integration tests..."
cd "$(dirname "$0")/.."
go test ./test/... -v -timeout=60s

echo "✅ Integration tests completed successfully!"