// ABOUTME: Implements distributed storage operations with replication and fault tolerance
// ABOUTME: Coordinates reads/writes across multiple nodes with configurable replication factor
package distributed

import (
	"context"
	"fmt"
	"sync"

	"distributed-sqlite/internal/storage"
)

type ClusterManager interface {
	GetReplicationNodes(key string) ([]*storage.Node, error)
	GetHealthyNodeCount() int
}

type DistributedStorage struct {
	cluster    ClusterManager
	local      storage.Storage
	client     *NodeClient
	replFactor int
}

func NewDistributedStorage(cluster ClusterManager, local storage.Storage, replFactor int) *DistributedStorage {
	return &DistributedStorage{
		cluster:    cluster,
		local:      local,
		client:     NewNodeClient(),
		replFactor: replFactor,
	}
}

func (d *DistributedStorage) Set(ctx context.Context, key string, value []byte) error {
	// Step 1: Write to local storage first (immediate response)
	if err := d.local.Set(ctx, key, value); err != nil {
		return fmt.Errorf("local write failed: %w", err)
	}

	// Step 2: Get replication nodes
	nodes, err := d.cluster.GetReplicationNodes(key)
	if err != nil {
		// Local write succeeded, log error but don't fail
		fmt.Printf("Warning: failed to get replication nodes: %v\n", err)
		return nil
	}

	// Step 3: Async replication to other nodes (excluding self)
	go d.replicateToNodes(context.Background(), key, value, nodes)

	return nil
}

func (d *DistributedStorage) replicateToNodes(ctx context.Context, key string, value []byte, nodes []*storage.Node) {
	var wg sync.WaitGroup
	replicated := 0
	maxReplicas := d.replFactor - 1 // -1 because we already wrote locally

	for _, node := range nodes {
		if replicated >= maxReplicas {
			break
		}

		wg.Add(1)
		go func(n *storage.Node) {
			defer wg.Done()
			if err := d.client.Set(ctx, n, key, value); err != nil {
				fmt.Printf("Failed to replicate to node %s: %v\n", n.ID, err)
			} else {
				fmt.Printf("Successfully replicated key %s to node %s\n", key, n.ID)
			}
		}(node)

		replicated++
	}

	wg.Wait()
}

func (d *DistributedStorage) Get(ctx context.Context, key string) ([]byte, error) {
	// Get all available nodes for reading
	nodes, err := d.cluster.GetReplicationNodes(key)
	if err != nil {
		// Fallback to local read if cluster query fails
		return d.local.Get(ctx, key)
	}

	// Collect read results from all nodes (including local)
	resultCh := make(chan nodeResult, len(nodes)+1)
	nodeCount := 0

	// Read from local storage
	nodeCount++
	go func() {
		value, err := d.local.Get(ctx, key)
		resultCh <- nodeResult{
			nodeID: "local",
			value:  value,
			err:    err,
		}
	}()

	// Query other nodes concurrently
	for _, node := range nodes {
		nodeCount++
		go func(n *storage.Node) {
			value, err := d.client.Get(ctx, n, key)
			resultCh <- nodeResult{
				nodeID: n.ID,
				value:  value,
				err:    err,
			}
		}(node)
	}

	// Collect results and look for majority consensus
	valueCounts := make(map[string]int)
	valueData := make(map[string][]byte)
	responses := 0
	nodesWithData := 0

	for responses < nodeCount {
		select {
		case result := <-resultCh:
			responses++
			if result.err == nil && result.value != nil {
				nodesWithData++
				valueStr := string(result.value)
				valueCounts[valueStr]++
				valueData[valueStr] = result.value

				// Calculate majority threshold based on nodes that have data
				majorityThreshold := (nodesWithData / 2) + 1

				// Return immediately if we have majority of nodes with data
				if valueCounts[valueStr] >= majorityThreshold {
					return result.value, nil
				}
			}
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Final check: find majority among all nodes that had data
	if nodesWithData > 0 {
		majorityThreshold := (nodesWithData / 2) + 1
		for _, value := range valueData {
			valueStr := string(value)
			if valueCounts[valueStr] >= majorityThreshold {
				return value, nil
			}
		}
	}

	// No majority found
	return nil, nil
}

type nodeResult struct {
	nodeID string
	value  []byte
	err    error
}

func (d *DistributedStorage) Delete(ctx context.Context, key string) error {
	// For now, just delete from local storage
	return d.local.Delete(ctx, key)
}

func (d *DistributedStorage) List(ctx context.Context) ([]string, error) {
	return d.local.List(ctx)
}

func (d *DistributedStorage) Close() error {
	return d.local.Close()
}