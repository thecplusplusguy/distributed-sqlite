// ABOUTME: Tests for the HTTP server that exposes public and internal endpoints
// ABOUTME: Uses real SQLite storage and an in-process cluster stub to validate handler behavior
package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"distributed-sqlite/internal/distributed"
	"distributed-sqlite/internal/storage"
)

// testCluster is an in-process ClusterManager that returns no peers, so
// DistributedStorage will only touch the local store during handler tests.
type testCluster struct{}

func (testCluster) GetReplicationNodes(key string) ([]*storage.Node, error) {
	return nil, nil
}

func (testCluster) GetAllPeers() ([]*storage.Node, error) { return nil, nil }

func (testCluster) GetHealthyNodeCount() int { return 1 }

func newTestServer(t *testing.T, nodeID string) (*Server, storage.Storage) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	local, err := storage.NewSQLiteStorage(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	t.Cleanup(func() { local.Close() })

	dist := distributed.NewDistributedStorage(testCluster{}, local, 1)
	return New(nodeID, local, dist), local
}

func TestHealth(t *testing.T) {
	srv, _ := newTestServer(t, "node-test")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var body struct {
		Status string `json:"status"`
		NodeID string `json:"node_id"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode health body: %v", err)
	}
	if body.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", body.Status)
	}
	if body.NodeID != "node-test" {
		t.Errorf("expected node_id 'node-test', got %q", body.NodeID)
	}
}

func TestInternalSet_WritesLocalOnly(t *testing.T) {
	srv, local := newTestServer(t, "node-a")

	payload := []byte(`{"key":"k1","value":{"hello":"world"}}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/set", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	stored, err := local.Get(context.Background(), "k1")
	if err != nil {
		t.Fatalf("local.Get returned error: %v", err)
	}
	if string(stored) != `{"hello":"world"}` {
		t.Errorf("expected stored value %q, got %q", `{"hello":"world"}`, string(stored))
	}
}

func TestInternalSet_RejectsBadJSON(t *testing.T) {
	srv, _ := newTestServer(t, "node-a")

	req := httptest.NewRequest(http.MethodPost, "/internal/set", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for malformed body, got %d", rr.Code)
	}
}

func TestInternalGet_ReturnsValue(t *testing.T) {
	srv, local := newTestServer(t, "node-a")
	if err := local.Set(context.Background(), "k2", []byte(`{"n":42}`)); err != nil {
		t.Fatalf("seed local.Set failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/internal/get?key=k2", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var body struct {
		Key    string          `json:"key"`
		Value  json.RawMessage `json:"value"`
		NodeID string          `json:"node_id"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Key != "k2" {
		t.Errorf("expected key 'k2', got %q", body.Key)
	}
	if string(body.Value) != `{"n":42}` {
		t.Errorf("expected value %q, got %q", `{"n":42}`, string(body.Value))
	}
	if body.NodeID != "node-a" {
		t.Errorf("expected node_id 'node-a', got %q", body.NodeID)
	}
}

func TestInternalGet_MissingKey404(t *testing.T) {
	srv, _ := newTestServer(t, "node-a")

	req := httptest.NewRequest(http.MethodGet, "/internal/get?key=nope", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestPublicSet_WritesThroughDistributed(t *testing.T) {
	srv, local := newTestServer(t, "node-a")

	payload := []byte(`{"key":"pk1","value":{"hello":"public"}}`)
	req := httptest.NewRequest(http.MethodPost, "/set", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	stored, err := local.Get(context.Background(), "pk1")
	if err != nil {
		t.Fatalf("local.Get error: %v", err)
	}
	if string(stored) != `{"hello":"public"}` {
		t.Errorf("expected %q, got %q", `{"hello":"public"}`, string(stored))
	}
}

func TestPublicGet_ReadsThroughDistributed(t *testing.T) {
	srv, local := newTestServer(t, "node-a")
	if err := local.Set(context.Background(), "pk2", []byte(`{"answer":42}`)); err != nil {
		t.Fatalf("seed local.Set failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/get?key=pk2", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", rr.Code, rr.Body.String())
	}

	var body struct {
		Key    string          `json:"key"`
		Value  json.RawMessage `json:"value"`
		NodeID string          `json:"node_id"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if string(body.Value) != `{"answer":42}` {
		t.Errorf("expected value %q, got %q", `{"answer":42}`, string(body.Value))
	}
}

func TestPublicGet_MissingKey404(t *testing.T) {
	srv, _ := newTestServer(t, "node-a")

	req := httptest.NewRequest(http.MethodGet, "/get?key=missing", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestPublicDelete_RemovesValue(t *testing.T) {
	srv, local := newTestServer(t, "node-a")
	if err := local.Set(context.Background(), "pk3", []byte(`{"x":1}`)); err != nil {
		t.Fatalf("seed local.Set failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/delete?key=pk3", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	stored, err := local.Get(context.Background(), "pk3")
	if err != nil {
		t.Fatalf("local.Get error: %v", err)
	}
	if stored != nil {
		t.Errorf("expected key removed, got %q", string(stored))
	}
}

func TestInternalDelete_RemovesValue(t *testing.T) {
	srv, local := newTestServer(t, "node-a")
	if err := local.Set(context.Background(), "k3", []byte(`{"x":1}`)); err != nil {
		t.Fatalf("seed local.Set failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/internal/delete?key=k3", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	stored, err := local.Get(context.Background(), "k3")
	if err != nil {
		t.Fatalf("local.Get error: %v", err)
	}
	if stored != nil {
		t.Errorf("expected key removed, got %q", string(stored))
	}
}
