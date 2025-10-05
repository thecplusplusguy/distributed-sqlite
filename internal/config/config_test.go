// ABOUTME: Tests for configuration management including replication factor validation
// ABOUTME: Validates config fails fast on invalid values rather than using unsafe defaults
package config

import (
	"os"
	"testing"
)

func TestConfig_MissingReplicationFactorFails(t *testing.T) {
	// Clear all env vars to test required validation
	os.Unsetenv("REPLICATION_FACTOR")
	os.Unsetenv("NODE_ID")
	os.Unsetenv("PORT")
	os.Unsetenv("CLUSTER_SIZE")

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for missing required config, got nil")
	}

	// Should fail on first missing required field
	if err != nil && !containsString(err.Error(), "NODE_ID environment variable is required") {
		t.Errorf("Expected 'NODE_ID environment variable is required' error, got: %v", err)
	}
}

func TestConfig_ValidReplicationFactorFromEnv(t *testing.T) {
	// Set all required environment variables
	os.Setenv("NODE_ID", "test-node")
	os.Setenv("PORT", "8080")
	os.Setenv("REPLICATION_FACTOR", "3")
	os.Setenv("CLUSTER_SIZE", "5")
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("PORT")
		os.Unsetenv("REPLICATION_FACTOR")
		os.Unsetenv("CLUSTER_SIZE")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	expected := 3
	if config.ReplicationFactor != expected {
		t.Errorf("Expected replication factor %d from env, got %d", expected, config.ReplicationFactor)
	}
}

func TestConfig_InvalidReplicationFactorFails(t *testing.T) {
	// Set all required env vars, but make replication factor invalid
	os.Setenv("NODE_ID", "test-node")
	os.Setenv("PORT", "8080")
	os.Setenv("REPLICATION_FACTOR", "invalid")
	os.Setenv("CLUSTER_SIZE", "3")
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("PORT")
		os.Unsetenv("REPLICATION_FACTOR")
		os.Unsetenv("CLUSTER_SIZE")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for invalid replication factor, got nil")
	}

	if err != nil && !containsString(err.Error(), "invalid replication factor") {
		t.Errorf("Expected 'invalid replication factor' error, got: %v", err)
	}
}

func TestConfig_ZeroReplicationFactorFails(t *testing.T) {
	// Set all required env vars, but make replication factor zero
	os.Setenv("NODE_ID", "test-node")
	os.Setenv("PORT", "8080")
	os.Setenv("REPLICATION_FACTOR", "0")
	os.Setenv("CLUSTER_SIZE", "3")
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("PORT")
		os.Unsetenv("REPLICATION_FACTOR")
		os.Unsetenv("CLUSTER_SIZE")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for zero replication factor, got nil")
	}

	if err != nil && !containsString(err.Error(), "replication factor must be positive") {
		t.Errorf("Expected 'replication factor must be positive' error, got: %v", err)
	}
}

func TestConfig_NegativeReplicationFactorFails(t *testing.T) {
	// Set all required env vars, but make replication factor negative
	os.Setenv("NODE_ID", "test-node")
	os.Setenv("PORT", "8080")
	os.Setenv("REPLICATION_FACTOR", "-1")
	os.Setenv("CLUSTER_SIZE", "3")
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("PORT")
		os.Unsetenv("REPLICATION_FACTOR")
		os.Unsetenv("CLUSTER_SIZE")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for negative replication factor, got nil")
	}

	if err != nil && !containsString(err.Error(), "replication factor must be positive") {
		t.Errorf("Expected 'replication factor must be positive' error, got: %v", err)
	}
}

func TestConfig_ReplicationFactorExceedsClusterSizeFails(t *testing.T) {
	// Set all required env vars, but make replication factor exceed cluster size
	os.Setenv("NODE_ID", "test-node")
	os.Setenv("PORT", "8080")
	os.Setenv("REPLICATION_FACTOR", "10")
	os.Setenv("CLUSTER_SIZE", "3")
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("PORT")
		os.Unsetenv("REPLICATION_FACTOR")
		os.Unsetenv("CLUSTER_SIZE")
	}()

	_, err := LoadConfig()
	if err == nil {
		t.Error("Expected error for replication factor exceeding cluster size, got nil")
	}

	if err != nil && !containsString(err.Error(), "cannot exceed cluster size") {
		t.Errorf("Expected 'cannot exceed cluster size' error, got: %v", err)
	}
}

func TestConfig_ValidConfigWithAllFields(t *testing.T) {
	// Set valid environment variables
	os.Setenv("NODE_ID", "test-node-123")
	os.Setenv("PORT", "9090")
	os.Setenv("REPLICATION_FACTOR", "2")
	os.Setenv("CLUSTER_SIZE", "3")
	defer func() {
		os.Unsetenv("NODE_ID")
		os.Unsetenv("PORT")
		os.Unsetenv("REPLICATION_FACTOR")
		os.Unsetenv("CLUSTER_SIZE")
	}()

	config, err := LoadConfig()
	if err != nil {
		t.Errorf("Expected no error for valid config, got: %v", err)
	}

	if config.NodeID != "test-node-123" {
		t.Errorf("Expected NodeID 'test-node-123', got '%s'", config.NodeID)
	}

	if config.Port != "9090" {
		t.Errorf("Expected Port '9090', got '%s'", config.Port)
	}

	if config.ReplicationFactor != 2 {
		t.Errorf("Expected ReplicationFactor 2, got %d", config.ReplicationFactor)
	}

	if config.ClusterSize != 3 {
		t.Errorf("Expected ClusterSize 3, got %d", config.ClusterSize)
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