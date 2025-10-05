#!/bin/bash
# ABOUTME: Script to test different cluster configurations
# ABOUTME: Tests 3, 4, and 5 node clusters with varying replication factors

set -e

echo "🧪 Running comprehensive cluster testing..."

# Test configurations: cluster_size:replication_factor
CONFIGS=(
    "3:3"
    "4:3"
    "5:4"
)

# Function to setup port forwards for a given cluster size
setup_port_forwards() {
    local cluster_size=$1
    echo "🔌 Setting up port forwards for $cluster_size nodes..."

    # Kill existing port forwards
    pkill -f "kubectl port-forward" || true
    sleep 2

    # Start port forwards
    for ((i=0; i<cluster_size; i++)); do
        port=$((8080 + i))
        kubectl port-forward "distributed-sqlite-nodes-$i" "$port:8080" &
        echo "  Node $i: localhost:$port"
    done

    echo "⏳ Waiting for port forwards to be ready..."
    sleep 5

    # Test connectivity
    for ((i=0; i<cluster_size; i++)); do
        port=$((8080 + i))
        if ! curl -f "http://localhost:$port/health" >/dev/null 2>&1; then
            echo "❌ Node $i not accessible on port $port"
            return 1
        fi
    done
    echo "✅ All $cluster_size nodes accessible"
}

# Function to cleanup port forwards
cleanup_port_forwards() {
    echo "🧹 Cleaning up port forwards..."
    pkill -f "kubectl port-forward" || true
    sleep 2
}

# Trap to ensure cleanup on exit
trap cleanup_port_forwards EXIT

# Run tests for each configuration
for config in "${CONFIGS[@]}"; do
    IFS=':' read -r cluster_size replication_factor <<< "$config"

    echo ""
    echo "================================================================"
    echo "🎯 Testing: $cluster_size nodes, replication factor $replication_factor"
    echo "================================================================"

    # Deploy cluster configuration
    echo "📦 Deploying cluster..."
    ./test/deploy_cluster.sh "$cluster_size" "$replication_factor"

    # Setup port forwards
    setup_port_forwards "$cluster_size"

    # Run tests
    echo "🧪 Running tests for $cluster_size-node cluster..."
    cd "$(dirname "$0")/.."

    # Run the specific cluster size test
    go test ./test/... -v -timeout=120s -run="TestMultipleClusterSizes/$cluster_size-node"

    echo "✅ Tests completed for $cluster_size-node cluster"

    # Cleanup port forwards before next iteration
    cleanup_port_forwards
    sleep 2
done

echo ""
echo "🎉 All cluster configuration tests completed successfully!"
echo ""
echo "📊 Summary:"
echo "  ✅ 3-node cluster (RF=3)"
echo "  ✅ 4-node cluster (RF=3)"
echo "  ✅ 5-node cluster (RF=4)"