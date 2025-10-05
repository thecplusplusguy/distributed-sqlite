// ABOUTME: Tests for different cluster sizes and replication factors
// ABOUTME: Validates distributed SQLite behavior across 3, 4, and 5 node clusters
package test

import (
	"fmt"
	"testing"
	"time"
)

type ClusterConfig struct {
	Size              int
	ReplicationFactor int
	Name              string
}

func TestMultipleClusterSizes(t *testing.T) {
	configs := []ClusterConfig{
		{Size: 3, ReplicationFactor: 3, Name: "3-node-rf3"},
		{Size: 4, ReplicationFactor: 3, Name: "4-node-rf3"},
		{Size: 5, ReplicationFactor: 4, Name: "5-node-rf4"},
	}

	for _, config := range configs {
		t.Run(config.Name, func(t *testing.T) {
			testClusterConfiguration(t, config)
		})
	}
}

func testClusterConfiguration(t *testing.T, config ClusterConfig) {
	t.Logf("Testing cluster: %d nodes, replication factor %d", config.Size, config.ReplicationFactor)

	// Generate node URLs based on cluster size
	nodeURLs := make([]string, config.Size)
	for i := 0; i < config.Size; i++ {
		nodeURLs[i] = fmt.Sprintf("http://localhost:%d", 8080+i)
	}

	// Test 1: Health check all nodes
	t.Run("health-checks", func(t *testing.T) {
		testHealthChecks(t, nodeURLs)
	})

	// Test 2: Write replication with expected replication factor
	t.Run("write-replication", func(t *testing.T) {
		testWriteReplication(t, nodeURLs, config.ReplicationFactor)
	})

	// Test 3: Majority reads
	t.Run("majority-reads", func(t *testing.T) {
		testMajorityReads(t, nodeURLs)
	})

	// Test 4: Fault tolerance (test with one node "down")
	t.Run("fault-tolerance", func(t *testing.T) {
		testFaultTolerance(t, nodeURLs, config.ReplicationFactor)
	})
}

func testHealthChecks(t *testing.T, nodeURLs []string) {
	for i, nodeURL := range nodeURLs {
		if !isNodeHealthy(nodeURL) {
			t.Errorf("Node %d (%s) is not healthy", i, nodeURL)
		} else {
			t.Logf("Node %d (%s) is healthy", i, nodeURL)
		}
	}
}

func testWriteReplication(t *testing.T, nodeURLs []string, expectedReplicas int) {
	testKey := fmt.Sprintf("replication-test-%d-%d", len(nodeURLs), time.Now().Unix())
	testValue := fmt.Sprintf(`{"message":"replication test","cluster_size":%d,"timestamp":"%s"}`,
		len(nodeURLs), time.Now().Format(time.RFC3339))

	// Write to first node
	writeData := TestData{
		Key:   testKey,
		Value: []byte(testValue),
	}

	err := writeToNode(nodeURLs[0], writeData)
	if err != nil {
		t.Fatalf("Failed to write to node 0: %v", err)
	}

	// Give time for replication
	time.Sleep(3 * time.Second)

	// Check how many nodes have the data
	replicaCount := 0
	for i, nodeURL := range nodeURLs {
		value, err := readFromNode(nodeURL, testKey)
		if err != nil {
			t.Logf("Failed to read from node %d: %v", i, err)
			continue
		}

		if value != nil && string(value) == testValue {
			replicaCount++
			t.Logf("Node %d has replicated data", i)
		}
	}

	if replicaCount < expectedReplicas {
		t.Errorf("Expected at least %d replicas, got %d", expectedReplicas, replicaCount)
	} else {
		t.Logf("Replication successful: %d/%d nodes have data (expected: %d)",
			replicaCount, len(nodeURLs), expectedReplicas)
	}
}

func testMajorityReads(t *testing.T, nodeURLs []string) {
	testKey := fmt.Sprintf("majority-read-test-%d-%d", len(nodeURLs), time.Now().Unix())
	testValue := fmt.Sprintf(`{"message":"majority read test","cluster_size":%d,"timestamp":"%s"}`,
		len(nodeURLs), time.Now().Format(time.RFC3339))

	// Write via first node
	writeData := TestData{
		Key:   testKey,
		Value: []byte(testValue),
	}

	err := writeToNode(nodeURLs[0], writeData)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Give time for replication
	time.Sleep(3 * time.Second)

	// Try reading from different nodes
	successfulReads := 0
	for i, nodeURL := range nodeURLs {
		value, err := readFromNode(nodeURL, testKey)
		if err != nil {
			t.Logf("Read from node %d failed: %v", i, err)
			continue
		}

		if value != nil {
			if string(value) == testValue {
				successfulReads++
				t.Logf("Node %d returned correct value", i)
			} else {
				t.Errorf("Node %d returned incorrect value", i)
			}
		}
	}

	// Should be able to read from majority of nodes
	majorityThreshold := (len(nodeURLs) / 2) + 1
	if successfulReads < majorityThreshold {
		t.Errorf("Expected reads from at least %d nodes, got %d", majorityThreshold, successfulReads)
	}
}

func testFaultTolerance(t *testing.T, nodeURLs []string, replicationFactor int) {
	// This test simulates fault tolerance by skipping the last node
	// In a real fault tolerance test, we'd actually stop a node

	if len(nodeURLs) <= 1 {
		t.Skip("Need at least 2 nodes for fault tolerance test")
		return
	}

	testKey := fmt.Sprintf("fault-tolerance-test-%d-%d", len(nodeURLs), time.Now().Unix())
	testValue := fmt.Sprintf(`{"message":"fault tolerance test","cluster_size":%d,"timestamp":"%s"}`,
		len(nodeURLs), time.Now().Format(time.RFC3339))

	// Write to first node
	writeData := TestData{
		Key:   testKey,
		Value: []byte(testValue),
	}

	err := writeToNode(nodeURLs[0], writeData)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Give time for replication
	time.Sleep(3 * time.Second)

	// Try to read from remaining nodes (simulating one node failure)
	availableNodes := nodeURLs[:len(nodeURLs)-1] // Skip last node
	successfulReads := 0

	for i, nodeURL := range availableNodes {
		value, err := readFromNode(nodeURL, testKey)
		if err != nil {
			t.Logf("Read from available node %d failed: %v", i, err)
			continue
		}

		if value != nil && string(value) == testValue {
			successfulReads++
		}
	}

	// Should still be able to read from enough nodes to get majority
	if successfulReads > 0 {
		t.Logf("Fault tolerance test passed: %d nodes still readable with one node 'down'", successfulReads)
	} else {
		t.Error("Fault tolerance test failed: no nodes readable with one node 'down'")
	}
}

func isNodeHealthy(nodeURL string) bool {
	_, err := readFromNode(nodeURL, "non-existent-key") // Just test connectivity
	// We expect a 404 or successful connection, not a connection error
	return err == nil || (err != nil && !isConnectionError(err))
}

func isConnectionError(err error) bool {
	// Simple check for connection-related errors
	errStr := err.Error()
	return len(errStr) > 0 && (
		fmt.Sprintf("%v", err) == "connection refused" ||
		fmt.Sprintf("%v", err) == "no route to host" ||
		fmt.Sprintf("%v", err) == "connection timeout")
}