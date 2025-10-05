// ABOUTME: Tests for HTTP client used for inter-node communication
// ABOUTME: Validates client can properly communicate with node HTTP endpoints
package distributed

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"distributed-sqlite/internal/storage"
)

func TestNodeClient_Set(t *testing.T) {
	// Create test server that mimics our /internal/set endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/set" {
			t.Errorf("Expected /internal/set path, got %s", r.URL.Path)
		}

		var req struct {
			Key   string          `json:"key"`
			Value json.RawMessage `json:"value"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		if req.Key != "test-key" {
			t.Errorf("Expected key 'test-key', got %s", req.Key)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"success","node_id":"test-node","key":"test-key"}`))
	}))
	defer server.Close()

	client := NewNodeClient()
	node := &storage.Node{
		ID:      "test-node",
		Address: server.URL[7:], // Remove "http://" prefix
	}

	testValue := []byte(`{"message":"test value"}`)
	err := client.Set(context.Background(), node, "test-key", testValue)

	if err != nil {
		t.Errorf("Set failed: %v", err)
	}
}

func TestNodeClient_Get(t *testing.T) {
	// Create test server that mimics our /internal/get endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/get" {
			t.Errorf("Expected /internal/get path, got %s", r.URL.Path)
		}

		key := r.URL.Query().Get("key")
		if key != "test-key" {
			t.Errorf("Expected key 'test-key', got %s", key)
		}

		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Key    string          `json:"key"`
			Value  json.RawMessage `json:"value"`
			NodeID string          `json:"node_id"`
		}{
			Key:    "test-key",
			Value:  json.RawMessage(`{"message":"test value"}`),
			NodeID: "test-node",
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewNodeClient()
	node := &storage.Node{
		ID:      "test-node",
		Address: server.URL[7:], // Remove "http://" prefix
	}

	value, err := client.Get(context.Background(), node, "test-key")

	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	expected := `{"message":"test value"}`
	if string(value) != expected {
		t.Errorf("Expected value %s, got %s", expected, string(value))
	}
}

func TestNodeClient_Get_NotFound(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Key not found"))
	}))
	defer server.Close()

	client := NewNodeClient()
	node := &storage.Node{
		ID:      "test-node",
		Address: server.URL[7:], // Remove "http://" prefix
	}

	value, err := client.Get(context.Background(), node, "nonexistent-key")

	if err != nil {
		t.Errorf("Get should not error on 404, got: %v", err)
	}

	if value != nil {
		t.Errorf("Expected nil value for nonexistent key, got %s", string(value))
	}
}

func TestNodeClient_Delete(t *testing.T) {
	// Create test server that mimics our /internal/delete endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE method, got %s", r.Method)
		}
		if r.URL.Path != "/internal/delete" {
			t.Errorf("Expected /internal/delete path, got %s", r.URL.Path)
		}

		key := r.URL.Query().Get("key")
		if key != "test-key" {
			t.Errorf("Expected key 'test-key', got %s", key)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"deleted","node_id":"test-node","key":"test-key"}`))
	}))
	defer server.Close()

	client := NewNodeClient()
	node := &storage.Node{
		ID:      "test-node",
		Address: server.URL[7:], // Remove "http://" prefix
	}

	err := client.Delete(context.Background(), node, "test-key")

	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}
}