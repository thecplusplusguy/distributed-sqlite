#!/bin/bash
# ABOUTME: Deploy test clusters in separate namespaces with unique ports
# ABOUTME: Creates isolated test environments for different cluster configurations

set -e

CLUSTER_SIZE=${1:-3}
REPLICATION_FACTOR=${2:-3}
DATE=$(date +%Y-%m-%d)
NAMESPACE="test-${DATE}-${CLUSTER_SIZE}n-${REPLICATION_FACTOR}r"

# Calculate base port for this configuration to avoid conflicts
BASE_PORT=$((9000 + (CLUSTER_SIZE * 100) + REPLICATION_FACTOR))

echo "🚀 Deploying test cluster: ${CLUSTER_SIZE} nodes, RF=${REPLICATION_FACTOR}"
echo "📦 Namespace: ${NAMESPACE}"
echo "🔌 Base port: ${BASE_PORT}"

# Validate inputs
if [ "$REPLICATION_FACTOR" -gt "$CLUSTER_SIZE" ]; then
    echo "❌ Replication factor ($REPLICATION_FACTOR) cannot exceed cluster size ($CLUSTER_SIZE)"
    exit 1
fi

# Create namespace if it doesn't exist
kubectl create namespace "$NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -

# Create temporary deployment files
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT

# Generate deployment with updated values
cat > "$TEMP_DIR/deployment.yaml" << EOF
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: distributed-sqlite-nodes
  namespace: $NAMESPACE
spec:
  serviceName: "distributed-sqlite-headless"
  replicas: $CLUSTER_SIZE
  selector:
    matchLabels:
      app: distributed-sqlite-node
  template:
    metadata:
      labels:
        app: distributed-sqlite-node
    spec:
      containers:
      - name: node
        image: distributed-sqlite-node:latest
        imagePullPolicy: Never
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: PORT
          value: "8080"
        - name: NODE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: REPLICATION_FACTOR
          value: "$REPLICATION_FACTOR"
        - name: CLUSTER_SIZE
          value: "$CLUSTER_SIZE"
        resources:
          requests:
            memory: "32Mi"
            cpu: "25m"
          limits:
            memory: "64Mi"
            cpu: "50m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 2
          periodSeconds: 5
        volumeMounts:
        - name: data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 100Mi
EOF

# Generate headless service
cat > "$TEMP_DIR/service-headless.yaml" << EOF
apiVersion: v1
kind: Service
metadata:
  name: distributed-sqlite-headless
  namespace: $NAMESPACE
spec:
  clusterIP: None
  selector:
    app: distributed-sqlite-node
  ports:
  - port: 8080
    targetPort: 8080
    name: http
EOF

# Generate external service with unique ports
cat > "$TEMP_DIR/service-external.yaml" << EOF
apiVersion: v1
kind: Service
metadata:
  name: distributed-sqlite-external
  namespace: $NAMESPACE
spec:
  type: LoadBalancer
  selector:
    app: distributed-sqlite-node
  ports:
  - port: $BASE_PORT
    targetPort: 8080
    name: http
EOF

echo "📦 Applying test cluster manifests..."
kubectl apply -f "$TEMP_DIR/"

echo "⏳ Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=distributed-sqlite-node -n "$NAMESPACE" --timeout=120s

echo "✅ Test cluster deployed successfully!"
echo ""
echo "📊 Cluster status:"
kubectl get pods -l app=distributed-sqlite-node -n "$NAMESPACE"
echo ""
echo "🔗 Port forward commands for testing:"
for ((i=0; i<CLUSTER_SIZE; i++)); do
    port=$((BASE_PORT + i))
    echo "  kubectl port-forward -n $NAMESPACE distributed-sqlite-nodes-$i $port:8080"
done
echo ""
echo "🧪 To run tests against this cluster:"
echo "  NAMESPACE=$NAMESPACE BASE_PORT=$BASE_PORT CLUSTER_SIZE=$CLUSTER_SIZE ./test/run_namespace_tests.sh"