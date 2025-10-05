#!/bin/bash
# ABOUTME: Script to deploy different cluster sizes for testing
# ABOUTME: Supports 3, 4, and 5 node clusters with appropriate replication factors

set -e

CLUSTER_SIZE=${1:-3}
REPLICATION_FACTOR=${2:-2}

echo "🚀 Deploying cluster with $CLUSTER_SIZE nodes, replication factor $REPLICATION_FACTOR"

# Validate inputs
case $CLUSTER_SIZE in
    3)
        if [ "$REPLICATION_FACTOR" -gt 3 ]; then
            echo "❌ Replication factor cannot exceed cluster size"
            exit 1
        fi
        ;;
    4)
        if [ "$REPLICATION_FACTOR" -gt 4 ]; then
            echo "❌ Replication factor cannot exceed cluster size"
            exit 1
        fi
        ;;
    5)
        if [ "$REPLICATION_FACTOR" -gt 5 ]; then
            echo "❌ Replication factor cannot exceed cluster size"
            exit 1
        fi
        ;;
    *)
        echo "❌ Supported cluster sizes: 3, 4, 5"
        exit 1
        ;;
esac

# Create temporary deployment file
TEMP_DEPLOYMENT=$(mktemp)
cp k8s/deployment.yaml "$TEMP_DEPLOYMENT"

# Update replicas and environment variables
sed -i "s/replicas: .*/replicas: $CLUSTER_SIZE/" "$TEMP_DEPLOYMENT"
sed -i "s/value: \"2\"/value: \"$REPLICATION_FACTOR\"/" "$TEMP_DEPLOYMENT"
sed -i "s/value: \"3\"/value: \"$CLUSTER_SIZE\"/" "$TEMP_DEPLOYMENT"

echo "📦 Applying updated deployment..."
kubectl apply -f k8s/configmap.yaml
kubectl apply -f k8s/service.yaml
kubectl apply -f "$TEMP_DEPLOYMENT"

echo "⏳ Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=distributed-sqlite-node --timeout=120s

echo "✅ Cluster deployed successfully!"
echo ""
echo "📊 Cluster status:"
kubectl get pods -l app=distributed-sqlite-node
echo ""
echo "🔗 To access the cluster:"
echo "  Run: kubectl port-forward distributed-sqlite-nodes-0 8080:8080"
for ((i=1; i<CLUSTER_SIZE; i++)); do
    port=$((8080 + i))
    echo "       kubectl port-forward distributed-sqlite-nodes-$i $port:8080"
done

# Cleanup
rm "$TEMP_DEPLOYMENT"