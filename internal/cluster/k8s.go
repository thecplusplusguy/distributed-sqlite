// ABOUTME: Kubernetes-aware cluster manager for distributed SQLite nodes
// ABOUTME: Discovers other nodes via k8s service and provides replication node selection
package cluster

import (
	"fmt"

	"distributed-sqlite/internal/storage"
)

type K8sClusterManager struct {
	currentNodeID     string
	namespace         string
	serviceName       string
	clusterSize       int
	replicationFactor int
}

func NewK8sClusterManager(nodeID, namespace, serviceName string, clusterSize, replicationFactor int) *K8sClusterManager {
	return &K8sClusterManager{
		currentNodeID:     nodeID,
		namespace:         namespace,
		serviceName:       serviceName,
		clusterSize:       clusterSize,
		replicationFactor: replicationFactor,
	}
}

func (k *K8sClusterManager) GetReplicationNodes(key string) ([]*storage.Node, error) {
	// For now, create static node list based on k8s StatefulSet naming
	// In production, this would use k8s API to discover actual pods

	var nodes []*storage.Node

	// Create nodes for other pods in the StatefulSet (excluding current node)
	for i := 0; i < k.clusterSize; i++ {
		nodeID := fmt.Sprintf("distributed-sqlite-nodes-%d", i)

		// Skip current node
		if nodeID == k.currentNodeID {
			continue
		}

		// Create node with k8s service address (pod.service.namespace.svc.cluster.local)
		address := fmt.Sprintf("%s.%s.%s.svc.cluster.local:8080", nodeID, k.serviceName, k.namespace)

		nodes = append(nodes, &storage.Node{
			ID:      nodeID,
			Address: address,
		})

		// Only return replicationFactor-1 nodes (excluding current node)
		if len(nodes) >= k.replicationFactor-1 {
			break
		}
	}

	if len(nodes) < k.replicationFactor-1 {
		return nodes, fmt.Errorf("insufficient nodes: need %d, have %d", k.replicationFactor-1, len(nodes))
	}

	return nodes, nil
}

func (k *K8sClusterManager) GetHealthyNodeCount() int {
	// For now, assume all nodes are healthy
	// In production, this would check pod status via k8s API
	return k.clusterSize
}