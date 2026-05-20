// ABOUTME: Tests for the Kubernetes-aware cluster manager
// ABOUTME: Validates peer DNS construction and replication node selection
package cluster

import (
	"strings"
	"testing"
)

func TestGetReplicationNodes_UsesProvidedNamespace(t *testing.T) {
	mgr := NewK8sClusterManager(
		"distributed-sqlite-nodes-0",
		"test-2025-10-05-3n-3r",
		"distributed-sqlite-headless",
		3,
		3,
	)

	nodes, err := mgr.GetReplicationNodes("any-key")
	if err != nil {
		t.Fatalf("GetReplicationNodes returned error: %v", err)
	}
	if len(nodes) == 0 {
		t.Fatal("expected at least one replication node, got none")
	}

	for _, n := range nodes {
		if !strings.Contains(n.Address, ".test-2025-10-05-3n-3r.svc.cluster.local") {
			t.Errorf("expected address in namespace test-2025-10-05-3n-3r, got %q", n.Address)
		}
		if strings.Contains(n.Address, ".default.svc.cluster.local") {
			t.Errorf("address should not reference 'default' namespace, got %q", n.Address)
		}
	}
}

func TestGetReplicationNodes_ExcludesSelfAndReturnsCorrectCount(t *testing.T) {
	mgr := NewK8sClusterManager(
		"distributed-sqlite-nodes-1",
		"default",
		"distributed-sqlite-headless",
		3,
		3,
	)

	nodes, err := mgr.GetReplicationNodes("any-key")
	if err != nil {
		t.Fatalf("GetReplicationNodes returned error: %v", err)
	}

	// RF=3 in a 3-node cluster: 2 peers (everyone except self).
	if len(nodes) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(nodes))
	}
	for _, n := range nodes {
		if n.ID == "distributed-sqlite-nodes-1" {
			t.Errorf("self should be excluded, got %q in peer list", n.ID)
		}
	}
}
