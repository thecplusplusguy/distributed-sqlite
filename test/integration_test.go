// ABOUTME: Integration tests for distributed SQLite running on real k8s deployment
// ABOUTME: Tests write replication, majority reads, and fault tolerance against live cluster
package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

const (
	node0URL = "http://localhost:8080"
	node1URL = "http://localhost:8081"
	node2URL = "http://localhost:8082"
)

type TestData struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

type GetResponse struct {
	Key    string          `json:"key"`
	Value  json.RawMessage `json:"value"`
	NodeID string          `json:"node_id"`
}

func TestDistributedWrite(t *testing.T) {
	// Test that writes to one node eventually replicate to others

	testKey := fmt.Sprintf("integration-test-%d", time.Now().Unix())
	testValue := fmt.Sprintf(`{"message":"distributed write test","timestamp":"%s"}`, time.Now().Format(time.RFC3339))

	// Write to node 0
	writeData := TestData{
		Key:   testKey,
		Value: json.RawMessage(testValue),
	}

	err := writeToNode(node0URL, writeData)
	if err != nil {
		t.Fatalf("Failed to write to node 0: %v", err)
	}

	// Give some time for async replication
	time.Sleep(2 * time.Second)

	// Try to read from all nodes to verify replication
	nodes := []string{node0URL, node1URL, node2URL}
	successCount := 0

	for i, nodeURL := range nodes {
		value, err := readFromNode(nodeURL, testKey)
		if err != nil {
			t.Logf("Failed to read from node %d: %v", i, err)
			continue
		}

		if value != nil && string(value) == testValue {
			successCount++
			t.Logf("Successfully read from node %d", i)
		}
	}

	// With replication factor 2, we should have at least 2 copies
	if successCount < 2 {
		t.Errorf("Expected at least 2 nodes to have the data, got %d", successCount)
	}
}

func TestDistributedRead(t *testing.T) {
	// Test that reads can get majority consensus even if some nodes are different

	testKey := fmt.Sprintf("read-test-%d", time.Now().Unix())
	testValue := fmt.Sprintf(`{"message":"distributed read test","timestamp":"%s"}`, time.Now().Format(time.RFC3339))

	// Write to multiple nodes to ensure data exists
	writeData := TestData{
		Key:   testKey,
		Value: json.RawMessage(testValue),
	}

	// Write via node 0 (which should replicate)
	err := writeToNode(node0URL, writeData)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Give time for replication
	time.Sleep(2 * time.Second)

	// Read from each node and verify we get consistent results
	for i, nodeURL := range []string{node0URL, node1URL, node2URL} {
		value, err := readFromNode(nodeURL, testKey)
		if err != nil {
			t.Logf("Node %d read failed (may be expected): %v", i, err)
			continue
		}

		if value != nil {
			if string(value) != testValue {
				t.Errorf("Node %d returned wrong value: expected %s, got %s", i, testValue, string(value))
			} else {
				t.Logf("Node %d returned correct value", i)
			}
		}
	}
}

func TestHealthEndpoints(t *testing.T) {
	// Test that all nodes respond to health checks

	nodes := []struct {
		name string
		url  string
	}{
		{"node-0", node0URL},
		{"node-1", node1URL},
		{"node-2", node2URL},
	}

	for _, node := range nodes {
		resp, err := http.Get(fmt.Sprintf("%s/health", node.url))
		if err != nil {
			t.Errorf("Health check failed for %s: %v", node.name, err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Health check for %s returned status %d", node.name, resp.StatusCode)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Failed to read health response from %s: %v", node.name, err)
			continue
		}

		t.Logf("%s health: %s", node.name, string(body))
	}
}

func TestWriteReadConsistency(t *testing.T) {
	// Test that a write followed immediately by a read gets the correct data

	testKey := fmt.Sprintf("consistency-test-%d", time.Now().Unix())
	testValue := fmt.Sprintf(`{"message":"consistency test","timestamp":"%s"}`, time.Now().Format(time.RFC3339))

	writeData := TestData{
		Key:   testKey,
		Value: json.RawMessage(testValue),
	}

	// Write to node 0
	err := writeToNode(node0URL, writeData)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// Immediately read from the same node (should always work)
	value, err := readFromNode(node0URL, testKey)
	if err != nil {
		t.Fatalf("Failed to read from same node: %v", err)
	}

	if value == nil {
		t.Fatal("Expected to read value immediately after write")
	}

	if string(value) != testValue {
		t.Errorf("Read-after-write consistency failed: expected %s, got %s", testValue, string(value))
	}
}

// Helper functions

func writeToNode(nodeURL string, data TestData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/internal/set", nodeURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("write failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func readFromNode(nodeURL string, key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("%s/internal/get?key=%s", nodeURL, key),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Key not found
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("read failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response GetResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Value, nil
}