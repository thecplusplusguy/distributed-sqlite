#!/bin/bash
# ABOUTME: Run integration tests against a specific namespaced test cluster
# ABOUTME: Uses environment variables to target the correct cluster and ports

set -e

NAMESPACE=${NAMESPACE:-"test-$(date +%Y-%m-%d)-3n-3r"}
BASE_PORT=${BASE_PORT:-9303}
CLUSTER_SIZE=${CLUSTER_SIZE:-3}

echo "🧪 Running tests against cluster in namespace: $NAMESPACE"
echo "🔌 Using ports: $BASE_PORT - $((BASE_PORT + CLUSTER_SIZE - 1))"

# Function to setup port forwards for the namespace
setup_port_forwards() {
    echo "🔌 Setting up port forwards for namespace $NAMESPACE..."

    # Kill existing port forwards for these ports
    for ((i=0; i<CLUSTER_SIZE; i++)); do
        port=$((BASE_PORT + i))
        lsof -ti:$port | xargs -r kill -9 || true
    done
    sleep 2

    # Start port forwards
    for ((i=0; i<CLUSTER_SIZE; i++)); do
        port=$((BASE_PORT + i))
        kubectl port-forward -n "$NAMESPACE" "distributed-sqlite-nodes-$i" "$port:8080" &
        echo "  Node $i: localhost:$port"
    done

    echo "⏳ Waiting for port forwards to be ready..."
    sleep 5

    # Test connectivity
    for ((i=0; i<CLUSTER_SIZE; i++)); do
        port=$((BASE_PORT + i))
        if ! curl -f "http://localhost:$port/health" >/dev/null 2>&1; then
            echo "❌ Node $i not accessible on port $port"
            return 1
        fi
    done
    echo "✅ All $CLUSTER_SIZE nodes accessible"
}

# Function to cleanup port forwards
cleanup_port_forwards() {
    echo "🧹 Cleaning up port forwards..."
    for ((i=0; i<CLUSTER_SIZE; i++)); do
        port=$((BASE_PORT + i))
        lsof -ti:$port | xargs -r kill -9 || true
    done
    sleep 1
}

# Trap to ensure cleanup on exit
trap cleanup_port_forwards EXIT

# Setup port forwards
setup_port_forwards

# Export port configuration for tests
export TEST_BASE_PORT=$BASE_PORT
export TEST_CLUSTER_SIZE=$CLUSTER_SIZE

echo "🧪 Running integration tests..."
cd "$(dirname "$0")/.."

# Create a temporary test file with the correct URLs
cat > test/cluster_urls.go << EOF
package test

// Auto-generated cluster URLs for namespace $NAMESPACE
var ClusterNodeURLs = []string{
EOF

for ((i=0; i<CLUSTER_SIZE; i++)); do
    port=$((BASE_PORT + i))
    echo "	\"http://localhost:$port\"," >> test/cluster_urls.go
done

echo "}" >> test/cluster_urls.go

# Run tests
go test ./test/... -v -timeout=180s -run="TestNamespaced"

echo "✅ Integration tests completed for namespace $NAMESPACE!"

# Cleanup the generated file
rm -f test/cluster_urls.go