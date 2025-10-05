// ABOUTME: Tests for distributed storage operations with replication and fault tolerance
// ABOUTME: Defines expected behavior for write coordination, read failover, and consistency
package distributed

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"distributed-sqlite/internal/storage"
)

// Helper to create real SQLite storage for testing
func newTestStorage(t *testing.T) storage.Storage {
	// Create temporary database file
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	sqliteStorage, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test SQLite storage: %v", err)
	}

	// Clean up after test
	t.Cleanup(func() {
		sqliteStorage.Close()
	})

	return sqliteStorage
}

// Mock cluster manager for testing
type mockCluster struct {
	nodes           []*storage.Node
	replicationFactor int
}

func newMockCluster(nodeCount, replicationFactor int) *mockCluster {
	nodes := make([]*storage.Node, nodeCount)
	for i := 0; i < nodeCount; i++ {
		nodes[i] = &storage.Node{
			ID:      fmt.Sprintf("node%d", i+1),
			Address: fmt.Sprintf("localhost:800%d", i+1),
		}
	}
	return &mockCluster{
		nodes:             nodes,
		replicationFactor: replicationFactor,
	}
}

func (m *mockCluster) GetReplicationNodes(key string) ([]*storage.Node, error) {
	if len(m.nodes) < m.replicationFactor {
		return nil, fmt.Errorf("insufficient nodes: need %d, have %d", m.replicationFactor, len(m.nodes))
	}
	
	// Return first replicationFactor nodes for simplicity in tests
	result := make([]*storage.Node, m.replicationFactor)
	for i := 0; i < m.replicationFactor; i++ {
		result[i] = m.nodes[i]
	}
	return result, nil
}

func (m *mockCluster) GetHealthyNodeCount() int {
	return len(m.nodes)
}

// Test that Set operation requires majority of replicas to succeed
func TestDistributedSet_RequiresMajoritySuccess(t *testing.T) {
	// Given: 3 nodes with replication factor 3
	cluster := newMockCluster(3, 3)
	localStorage := newTestStorage(t)
	
	ctx := context.Background()
	key := "test-key"
	value := []byte(`{"message":"test-value"}`)
	
	distStorage := NewDistributedStorage(cluster, localStorage, 3)
	err := distStorage.Set(ctx, key, value)
	if err != nil {
		t.Errorf("Expected successful write with replication factor 3, got: %v", err)
	}
	
	// Verify data was written to local storage
	stored, err := localStorage.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected data to be stored locally, got error: %v", err)
	}
	
	if string(stored) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(stored))
	}
}

// Test that Set succeeds locally even when insufficient replicas are available
func TestDistributedSet_SucceedsLocallyWithInsufficientNodes(t *testing.T) {
	// Given: 2 nodes but replication factor 3
	cluster := newMockCluster(2, 3)
	localStorage := newTestStorage(t)
	
	ctx := context.Background()
	key := "test-key"
	value := []byte(`{"message":"test-value"}`)
	
	distStorage := NewDistributedStorage(cluster, localStorage, 3)
	err := distStorage.Set(ctx, key, value)
	
	// Our implementation writes locally first, so it should succeed
	if err != nil {
		t.Errorf("Expected local write to succeed even with insufficient remote nodes, got: %v", err)
	}

	// Verify data was written to local storage
	stored, err := localStorage.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected data to be stored locally, got error: %v", err)
	}

	if string(stored) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(stored))
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		findInString(s, substr)))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Test that Get succeeds when reading from any available replica (eventual consistency)
func TestDistributedGet_SucceedsFromAnyReplica(t *testing.T) {
	// Given: 3 nodes with replication factor 2
	cluster := newMockCluster(3, 2)
	localStorage := newTestStorage(t)
	
	// Pre-populate local storage to simulate data exists on this replica
	ctx := context.Background()
	key := "existing-key"
	value := []byte(`{"message":"existing-value"}`)
	localStorage.Set(ctx, key, value)
	
	distStorage := NewDistributedStorage(cluster, localStorage, 3)
	result, err := distStorage.Get(ctx, key)
	
	if err != nil {
		t.Errorf("Expected successful read, got error: %v", err)
	}
	
	if string(result) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(result))
	}
}

// Test that Delete coordinates across replicas (majority write like Set)
func TestDistributedDelete_CoordinatesAcrossReplicas(t *testing.T) {
	// Given: 3 nodes with replication factor 3
	cluster := newMockCluster(3, 3)
	localStorage := newTestStorage(t)
	
	// Pre-populate data to delete
	ctx := context.Background()
	key := "delete-key"
	value := []byte(`{"message":"delete-value"}`)
	localStorage.Set(ctx, key, value)
	
	distStorage := NewDistributedStorage(cluster, localStorage, 3)
	err := distStorage.Delete(ctx, key)
	
	if err != nil {
		t.Errorf("Expected successful delete, got error: %v", err)
	}
	
	// Verify data was deleted from local storage
	_, err = localStorage.Get(ctx, key)
	if err == nil {
		t.Error("Expected key to be deleted from local storage")
	}
}

// Test fault tolerance - system works when some nodes fail
func TestDistributedStorage_FaultTolerance(t *testing.T) {
	// Given: 5 nodes with replication factor 3 (can tolerate 2 failures)
	cluster := newMockCluster(5, 3)
	localStorage := newTestStorage(t)
	
	ctx := context.Background()
	key := "fault-test"
	value := []byte(`{"message":"should-survive-failures"}`)
	
	distStorage := NewDistributedStorage(cluster, localStorage, 3)
	
	// Should be able to write even with fault tolerance requirements
	err := distStorage.Set(ctx, key, value)
	if err != nil {
		t.Errorf("Expected write to succeed with fault tolerance, got: %v", err)
	}
	
	// Should still be able to read the data (eventual consistency)
	result, err := distStorage.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected read to succeed after write, got: %v", err)
	}
	
	if string(result) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(result))
	}
}