// ABOUTME: HTTP client for inter-node communication in distributed SQLite system
// ABOUTME: Handles set/get/delete operations across cluster nodes with majority consensus
package distributed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"distributed-sqlite/internal/storage"
)

type NodeClient struct {
	httpClient *http.Client
	timeout    time.Duration
}

func NewNodeClient() *NodeClient {
	return &NodeClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		timeout: 3 * time.Second,
	}
}

func (c *NodeClient) Set(ctx context.Context, node *storage.Node, key string, value []byte) error {
	reqBody := struct {
		Key   string          `json:"key"`
		Value json.RawMessage `json:"value"`
	}{
		Key:   key,
		Value: value,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("http://%s/internal/set", node.Address)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", node.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node %s returned status %d: %s", node.ID, resp.StatusCode, string(body))
	}

	return nil
}

func (c *NodeClient) Get(ctx context.Context, node *storage.Node, key string) ([]byte, error) {
	url := fmt.Sprintf("http://%s/internal/get?key=%s", node.Address, key)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", node.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // Key not found
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("node %s returned status %d: %s", node.ID, resp.StatusCode, string(body))
	}

	var response struct {
		Key    string          `json:"key"`
		Value  json.RawMessage `json:"value"`
		NodeID string          `json:"node_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response from %s: %w", node.ID, err)
	}

	return response.Value, nil
}

func (c *NodeClient) Delete(ctx context.Context, node *storage.Node, key string) error {
	url := fmt.Sprintf("http://%s/internal/delete?key=%s", node.Address, key)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", node.ID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node %s returned status %d: %s", node.ID, resp.StatusCode, string(body))
	}

	return nil
}