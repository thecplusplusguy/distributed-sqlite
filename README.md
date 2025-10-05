# Distributed SQLite

A distributed SQLite system with configurable replication factor where N-R nodes can fail without data loss (N=total nodes, R=replication factor).

## Architecture

- **Write Model**: Write-local-first with async replication to RF nodes
- **Read Model**: Majority consensus reads for consistency
- **Deployment**: Kubernetes StatefulSet with persistent volumes
- **Communication**: HTTP-based inter-node coordination
- **Storage**: SQLite with JSON support for native JSON storage

## Features

- ✅ Configurable replication factor
- ✅ Fault tolerance (N-R node failures)
- ✅ Majority consensus reads
- ✅ Kubernetes-native deployment
- ✅ Automated integration testing
- ✅ Multi-cluster size validation

## Quick Start

### Prerequisites

- Docker
- Kubernetes cluster (kind, minikube, or full cluster)
- kubectl configured
- Go 1.21+

### Build and Deploy

1. **Build the container image**:
   ```bash
   docker build -t distributed-sqlite-node:latest .
   ```

2. **Deploy to Kubernetes**:
   ```bash
   kubectl apply -f k8s/
   ```

3. **Wait for pods to be ready**:
   ```bash
   kubectl wait --for=condition=ready pod -l app=distributed-sqlite-node --timeout=120s
   ```

## Running Integration Tests

The system includes comprehensive integration tests that validate distributed operations across different cluster configurations.

### Single Cluster Testing

To test against the default 3-node cluster:

```bash
# Setup port forwards
kubectl port-forward distributed-sqlite-nodes-0 8080:8080 &
kubectl port-forward distributed-sqlite-nodes-1 8081:8080 &
kubectl port-forward distributed-sqlite-nodes-2 8082:8080 &

# Run integration tests
go test ./test/... -v -timeout=120s
```

### Multi-Cluster Testing (Recommended)

For comprehensive testing across different cluster sizes with namespace isolation:

#### Test 3-Node Cluster (RF=3)
```bash
# Deploy isolated test cluster
./test/deploy_test_cluster.sh 3 3

# Run tests against this cluster
NAMESPACE=test-$(date +%Y-%m-%d)-3n-3r BASE_PORT=9303 CLUSTER_SIZE=3 ./test/run_namespace_tests.sh

# Cleanup
kubectl delete namespace test-$(date +%Y-%m-%d)-3n-3r
```

#### Test 4-Node Cluster (RF=3)
```bash
# Deploy isolated test cluster
./test/deploy_test_cluster.sh 4 3

# Run tests against this cluster
NAMESPACE=test-$(date +%Y-%m-%d)-4n-3r BASE_PORT=9403 CLUSTER_SIZE=4 ./test/run_namespace_tests.sh

# Cleanup
kubectl delete namespace test-$(date +%Y-%m-%d)-4n-3r
```

#### Test 5-Node Cluster (RF=4)
```bash
# Deploy isolated test cluster
./test/deploy_test_cluster.sh 5 4

# Run tests against this cluster
NAMESPACE=test-$(date +%Y-%m-%d)-5n-4r BASE_PORT=9504 CLUSTER_SIZE=5 ./test/run_namespace_tests.sh

# Cleanup
kubectl delete namespace test-$(date +%Y-%m-%d)-5n-4r
```

### Automated Test Scripts

The repository includes several test automation scripts:

- **`test/deploy_test_cluster.sh`**: Deploy clusters in isolated namespaces
- **`test/run_namespace_tests.sh`**: Run tests against specific namespaced clusters
- **`test/run_cluster_tests.sh`**: Comprehensive multi-cluster testing (all sizes)

#### Run All Cluster Configurations
```bash
# Test all supported configurations automatically
./test/run_cluster_tests.sh
```

This will test:
- 3-node cluster with RF=3
- 4-node cluster with RF=3
- 5-node cluster with RF=4

### Test Coverage

The integration tests validate:

1. **Health Checks**: All nodes respond to health endpoints
2. **Write Replication**: Data replicates to all nodes according to RF
3. **Majority Reads**: Consistent reads across multiple nodes
4. **Fault Tolerance**: System remains operational with node failures
5. **Concurrent Operations**: Behavior under concurrent load

### Expected Test Results

✅ **3-node cluster (RF=3)**:
- Full replication to all 3 nodes
- Fault tolerance with 1 node failure
- Majority consensus reads

✅ **4-node cluster (RF=3)**:
- Replication to 3 out of 4 nodes
- Fault tolerance with 1 node failure
- Majority consensus reads

✅ **5-node cluster (RF=4)**:
- Replication to 4 out of 5 nodes
- Fault tolerance with 1 node failure
- Majority consensus reads

### Known Limitations

- **SQLite Concurrency**: High concurrent write operations may experience database locking
- **Solution**: Operation queue implementation (planned)

## Configuration

Set these environment variables in your deployment:

- `NODE_ID`: Unique identifier for the node
- `PORT`: HTTP server port (default: 8080)
- `REPLICATION_FACTOR`: Number of replicas (required)
- `CLUSTER_SIZE`: Total number of nodes (required)

## API Endpoints

### Public API
- `GET /health` - Health check
- `POST /set` - Store key-value pair
- `GET /get?key=<key>` - Retrieve value
- `DELETE /delete?key=<key>` - Delete key

### Internal API (Inter-node)
- `POST /internal/set` - Replication endpoint
- `GET /internal/get?key=<key>` - Internal read endpoint
- `DELETE /internal/delete?key=<key>` - Internal delete endpoint

## Development

### Test-Driven Development

This project follows TDD principles:

```bash
# Run unit tests
go test ./internal/... -v

# Run integration tests
go test ./test/... -v

# Run all tests
go test ./... -v
```

### Adding New Features

1. Write failing tests first
2. Implement minimal code to pass tests
3. Refactor while keeping tests green
4. Validate with integration tests

## Architecture Details

### Replication Strategy

1. **Write-local-first**: Writes succeed locally immediately
2. **Async replication**: Background replication to RF-1 other nodes
3. **Best-effort delivery**: Log failures but don't block writes

### Read Strategy

1. **Concurrent reads**: Query all available nodes simultaneously
2. **Majority consensus**: Return value when majority of responding nodes agree
3. **Fast response**: Return as soon as majority threshold is reached

### Fault Tolerance

- **N-R failures tolerated**: Where N=cluster size, R=replication factor
- **Graceful degradation**: System continues operating with reduced capacity
- **Automatic recovery**: Nodes rejoin cluster when healthy

## Contributing

1. Follow TDD approach
2. Ensure all tests pass
3. Update documentation
4. Add integration tests for new features

## License

[Add your license here]
