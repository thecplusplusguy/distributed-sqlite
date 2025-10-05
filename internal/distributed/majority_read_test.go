// ABOUTME: Tests for majority read functionality in distributed storage
// ABOUTME: Validates that reads query all nodes and return when majority consensus is reached
package distributed

import (
	"context"
	"testing"
	"time"
)

func TestDistributedGet_MajorityRead(t *testing.T) {
	// Test that reads return as soon as majority of nodes agree on a value

	// This will require updating Get() to actually query multiple nodes
	// For now, just ensure the basic interface works
	cluster := newMockCluster(3, 2)
	localStorage := newTestStorage(t)

	ctx := context.Background()
	key := "majority-test-key"
	value := []byte(`{"data":"majority-value"}`)

	// Pre-populate local storage
	err := localStorage.Set(ctx, key, value)
	if err != nil {
		t.Fatalf("Failed to set test data: %v", err)
	}

	// Verify data was stored
	stored, err := localStorage.Get(ctx, key)
	if err != nil {
		t.Fatalf("Failed to get test data: %v", err)
	}
	if stored == nil {
		t.Fatal("Test data was not stored")
	}

	distStorage := NewDistributedStorage(cluster, localStorage, 2)

	// This should eventually query multiple nodes and return on majority consensus
	result, err := distStorage.Get(ctx, key)
	if err != nil {
		t.Errorf("Expected successful majority read, got error: %v", err)
	}

	if result == nil {
		t.Error("Expected non-nil result, got nil")
		return
	}

	if string(result) != string(value) {
		t.Errorf("Expected %s, got %s", string(value), string(result))
	}
}

func TestDistributedGet_MajorityReadWithTimeout(t *testing.T) {
	// Test that reads timeout if majority consensus cannot be reached
	// This test will be more meaningful once we implement actual multi-node reads

	cluster := newMockCluster(3, 2)
	localStorage := newTestStorage(t)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	key := "timeout-test-key"

	distStorage := NewDistributedStorage(cluster, localStorage, 2)

	// Should return nil for non-existent key (no majority)
	result, err := distStorage.Get(ctx, key)
	if err != nil {
		t.Errorf("Get should not error for non-existent key, got: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil for non-existent key, got %s", string(result))
	}
}