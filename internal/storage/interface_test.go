// ABOUTME: Tests for the storage interface and node structure definitions
// ABOUTME: Validates storage interface contract and node data structure
package storage

import (
	"context"
	"testing"
)

func TestNodeStruct(t *testing.T) {
	node := Node{
		ID:      "node1",
		Address: "localhost:8080",
	}

	if node.ID != "node1" {
		t.Errorf("Expected node ID to be 'node1', got %s", node.ID)
	}

	if node.Address != "localhost:8080" {
		t.Errorf("Expected node address to be 'localhost:8080', got %s", node.Address)
	}
}

func TestNodeStructEmpty(t *testing.T) {
	node := Node{}

	if node.ID != "" {
		t.Errorf("Expected empty node ID, got %s", node.ID)
	}

	if node.Address != "" {
		t.Errorf("Expected empty node address, got %s", node.Address)
	}
}

// MockStorage implements Storage interface for testing
type MockStorage struct {
	data map[string][]byte
}

func NewMockStorage() *MockStorage {
	return &MockStorage{
		data: make(map[string][]byte),
	}
}

func (m *MockStorage) Set(ctx context.Context, key string, value []byte) error {
	m.data[key] = value
	return nil
}

func (m *MockStorage) Get(ctx context.Context, key string) ([]byte, error) {
	value, exists := m.data[key]
	if !exists {
		return nil, nil
	}
	return value, nil
}

func (m *MockStorage) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *MockStorage) List(ctx context.Context) ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys, nil
}

func (m *MockStorage) Close() error {
	return nil
}

func TestStorageInterface(t *testing.T) {
	var storage Storage = NewMockStorage()
	ctx := context.Background()

	// Test Set and Get
	err := storage.Set(ctx, "test-key", []byte("test-value"))
	if err != nil {
		t.Errorf("Set failed: %v", err)
	}

	value, err := storage.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Get failed: %v", err)
	}

	if string(value) != "test-value" {
		t.Errorf("Expected 'test-value', got %s", string(value))
	}

	// Test List
	keys, err := storage.List(ctx)
	if err != nil {
		t.Errorf("List failed: %v", err)
	}

	if len(keys) != 1 || keys[0] != "test-key" {
		t.Errorf("Expected keys ['test-key'], got %v", keys)
	}

	// Test Delete
	err = storage.Delete(ctx, "test-key")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	value, err = storage.Get(ctx, "test-key")
	if err != nil {
		t.Errorf("Get after delete failed: %v", err)
	}

	if value != nil {
		t.Errorf("Expected nil after delete, got %s", string(value))
	}

	// Test Close
	err = storage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}