// ABOUTME: Configuration management for distributed SQLite with strict validation
// ABOUTME: All config values are required from environment variables - no defaults allowed
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	NodeID            string
	Port              string
	ReplicationFactor int
	ClusterSize       int
	Namespace         string
}

func LoadConfig() (*Config, error) {
	config := &Config{}

	// All config values are required
	var err error

	config.NodeID = os.Getenv("NODE_ID")
	if config.NodeID == "" {
		return nil, fmt.Errorf("NODE_ID environment variable is required")
	}

	config.Port = os.Getenv("PORT")
	if config.Port == "" {
		return nil, fmt.Errorf("PORT environment variable is required")
	}

	// Parse replication factor with validation
	replFactorStr := os.Getenv("REPLICATION_FACTOR")
	if replFactorStr == "" {
		return nil, fmt.Errorf("REPLICATION_FACTOR environment variable is required")
	}

	config.ReplicationFactor, err = strconv.Atoi(replFactorStr)
	if err != nil {
		return nil, fmt.Errorf("invalid replication factor '%s': must be a number", replFactorStr)
	}

	if config.ReplicationFactor <= 0 {
		return nil, fmt.Errorf("replication factor must be positive, got %d", config.ReplicationFactor)
	}

	// Parse cluster size with validation
	clusterSizeStr := os.Getenv("CLUSTER_SIZE")
	if clusterSizeStr == "" {
		return nil, fmt.Errorf("CLUSTER_SIZE environment variable is required")
	}

	config.ClusterSize, err = strconv.Atoi(clusterSizeStr)
	if err != nil {
		return nil, fmt.Errorf("invalid cluster size '%s': must be a number", clusterSizeStr)
	}

	if config.ClusterSize <= 0 {
		return nil, fmt.Errorf("cluster size must be positive, got %d", config.ClusterSize)
	}

	// Validate replication factor doesn't exceed cluster size
	if config.ReplicationFactor > config.ClusterSize {
		return nil, fmt.Errorf("replication factor (%d) cannot exceed cluster size (%d)", config.ReplicationFactor, config.ClusterSize)
	}

	config.Namespace = os.Getenv("POD_NAMESPACE")
	if config.Namespace == "" {
		return nil, fmt.Errorf("POD_NAMESPACE environment variable is required")
	}

	return config, nil
}