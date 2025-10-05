// ABOUTME: Tests that run against namespaced test clusters
// ABOUTME: Uses dynamically generated cluster URLs for isolated testing
package test

import (
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestNamespacedCluster(t *testing.T) {
	// Get cluster configuration from environment
	basePortStr := os.Getenv("TEST_BASE_PORT")
	clusterSizeStr := os.Getenv("TEST_CLUSTER_SIZE")

	if basePortStr == "" || clusterSizeStr == "" {
		t.Skip("Skipping namespaced test - requires TEST_BASE_PORT and TEST_CLUSTER_SIZE env vars")
	}

	basePort, err := strconv.Atoi(basePortStr)
	if err != nil {
		t.Fatalf("Invalid TEST_BASE_PORT: %v", err)
	}

	clusterSize, err := strconv.Atoi(clusterSizeStr)
	if err != nil {
		t.Fatalf("Invalid TEST_CLUSTER_SIZE: %v", err)
	}

	// Generate node URLs
	nodeURLs := make([]string, clusterSize)
	for i := 0; i < clusterSize; i++ {
		nodeURLs[i] = fmt.Sprintf("http://localhost:%d", basePort+i)
	}

	t.Logf("Testing namespaced cluster with %d nodes", clusterSize)

	// Run comprehensive tests
	t.Run("health-checks", func(t *testing.T) {
		testHealthChecks(t, nodeURLs)
	})

	t.Run("write-replication", func(t *testing.T) {
		// For namespaced tests, we expect full replication
		expectedReplicas := clusterSize
		testWriteReplication(t, nodeURLs, expectedReplicas)
	})

	t.Run("majority-reads", func(t *testing.T) {
		testMajorityReads(t, nodeURLs)
	})

	t.Run("fault-tolerance", func(t *testing.T) {
		// Test with expected replication factor
		testFaultTolerance(t, nodeURLs, clusterSize)
	})

	t.Run("concurrent-operations", func(t *testing.T) {
		testConcurrentOperations(t, nodeURLs)
	})
}

func testConcurrentOperations(t *testing.T, nodeURLs []string) {
	// Test concurrent writes and reads
	concurrency := 5
	results := make(chan error, concurrency)

	baseKey := fmt.Sprintf("concurrent-test-%d", time.Now().Unix())

	// Start concurrent operations
	for i := 0; i < concurrency; i++ {
		go func(id int) {
			testKey := fmt.Sprintf("%s-%d", baseKey, id)
			testValue := fmt.Sprintf(`{"message":"concurrent test %d","timestamp":"%s"}`,
				id, time.Now().Format(time.RFC3339))

			writeData := TestData{
				Key:   testKey,
				Value: []byte(testValue),
			}

			// Write to random node
			nodeURL := nodeURLs[id%len(nodeURLs)]
			err := writeToNode(nodeURL, writeData)
			if err != nil {
				results <- fmt.Errorf("write %d failed: %v", id, err)
				return
			}

			// Give time for replication
			time.Sleep(1 * time.Second)

			// Read from different node
			readNodeURL := nodeURLs[(id+1)%len(nodeURLs)]
			value, err := readFromNode(readNodeURL, testKey)
			if err != nil {
				results <- fmt.Errorf("read %d failed: %v", id, err)
				return
			}

			if value == nil {
				results <- fmt.Errorf("read %d got nil value", id)
				return
			}

			if string(value) != testValue {
				results <- fmt.Errorf("read %d got wrong value", id)
				return
			}

			results <- nil
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < concurrency; i++ {
		err := <-results
		if err != nil {
			t.Logf("Concurrent operation failed: %v", err)
		} else {
			successCount++
		}
	}

	t.Logf("Concurrent operations: %d/%d succeeded", successCount, concurrency)

	// We expect most operations to succeed
	if successCount < concurrency/2 {
		t.Errorf("Too many concurrent operations failed: %d/%d succeeded", successCount, concurrency)
	}
}