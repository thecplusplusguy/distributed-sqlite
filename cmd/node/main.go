// ABOUTME: Entry point for the distributed SQLite node binary
// ABOUTME: Loads config, opens SQLite, builds the cluster + distributed coordinator, and serves HTTP
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"distributed-sqlite/internal/cluster"
	"distributed-sqlite/internal/config"
	"distributed-sqlite/internal/distributed"
	"distributed-sqlite/internal/server"
	"distributed-sqlite/internal/storage"
)

const (
	dataDir              = "/data"
	dbFilename           = "node.db"
	headlessServiceName  = "distributed-sqlite-headless"
	shutdownGracePeriod  = 15 * time.Second
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	dbPath := filepath.Join(dataDir, dbFilename)
	local, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		log.Fatalf("failed to open sqlite at %s: %v", dbPath, err)
	}
	defer local.Close()

	clusterMgr := cluster.NewK8sClusterManager(cfg.NodeID, headlessServiceName, cfg.ClusterSize, cfg.ReplicationFactor)
	dist := distributed.NewDistributedStorage(clusterMgr, local, cfg.ReplicationFactor)
	srv := server.New(cfg.NodeID, local, dist)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: srv.Handler(),
	}

	serverErrCh := make(chan error, 1)
	go func() {
		log.Printf("node %s listening on :%s (cluster_size=%d, replication_factor=%d)",
			cfg.NodeID, cfg.Port, cfg.ClusterSize, cfg.ReplicationFactor)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrCh:
		log.Fatalf("http server error: %v", err)
	case sig := <-signalCh:
		log.Printf("received signal %s, shutting down", sig)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownGracePeriod)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
}
