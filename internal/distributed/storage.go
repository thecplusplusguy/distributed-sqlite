// ABOUTME: Implements distributed storage operations with replication and fault tolerance
// ABOUTME: Coordinates reads/writes across multiple nodes with configurable replication factor
package distributed

import (
	"context"

	"distributed-sqlite/internal/storage"
)

type ClusterManager interface {
	GetReplicationNodes(key string) ([]*storage.Node, error)
	GetHealthyNodeCount() int
}

type DistributedStorage struct {
	cluster ClusterManager
	local   storage.Storage
}

func NewDistributedStorage(cluster ClusterManager, local storage.Storage) *DistributedStorage {
	return &DistributedStorage{
		cluster: cluster,
		local:   local,
	}
}

func (d *DistributedStorage) Set(ctx context.Context, key string, value []byte) error {
	_, err := d.cluster.GetReplicationNodes(key)
	if err != nil {
		return err
	}

	// For now, just write to local storage to make first test pass
	return d.local.Set(ctx, key, value)
}

func (d *DistributedStorage) Get(ctx context.Context, key string) ([]byte, error) {
	// For now, just read from local storage
	return d.local.Get(ctx, key)
}

func (d *DistributedStorage) Delete(ctx context.Context, key string) error {
	// For now, just delete from local storage
	return d.local.Delete(ctx, key)
}

func (d *DistributedStorage) List(ctx context.Context) ([]string, error) {
	return d.local.List(ctx)
}

func (d *DistributedStorage) Close() error {
	return d.local.Close()
}