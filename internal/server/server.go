// ABOUTME: HTTP server exposing public key-value API and internal replication endpoints
// ABOUTME: Public routes go through DistributedStorage; internal routes touch local storage only
package server

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"distributed-sqlite/internal/distributed"
	"distributed-sqlite/internal/storage"
)

// Server wires the local SQLite storage and the distributed coordinator
// into a set of HTTP handlers. The local storage backs the /internal/*
// endpoints (called peer-to-peer during replication) while the distributed
// storage backs the public API that clients use.
type Server struct {
	nodeID string
	local  storage.Storage
	dist   *distributed.DistributedStorage
	router *mux.Router
}

func New(nodeID string, local storage.Storage, dist *distributed.DistributedStorage) *Server {
	s := &Server{
		nodeID: nodeID,
		local:  local,
		dist:   dist,
		router: mux.NewRouter(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.router
}

func (s *Server) routes() {
	s.router.HandleFunc("/health", s.handleHealth).Methods(http.MethodGet)

	s.router.HandleFunc("/set", s.handlePublicSet).Methods(http.MethodPost)
	s.router.HandleFunc("/get", s.handlePublicGet).Methods(http.MethodGet)
	s.router.HandleFunc("/delete", s.handlePublicDelete).Methods(http.MethodDelete)

	s.router.HandleFunc("/internal/set", s.handleInternalSet).Methods(http.MethodPost)
	s.router.HandleFunc("/internal/get", s.handleInternalGet).Methods(http.MethodGet)
	s.router.HandleFunc("/internal/delete", s.handleInternalDelete).Methods(http.MethodDelete)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"node_id": s.nodeID,
	})
}

// setRequest is the wire format for both /set and /internal/set.
// Value is a raw JSON message so the storage layer keeps native JSON.
type setRequest struct {
	Key   string          `json:"key"`
	Value json.RawMessage `json:"value"`
}

func (s *Server) handleInternalSet(w http.ResponseWriter, r *http.Request) {
	var req setRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	if err := s.local.Set(r.Context(), req.Key, req.Value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"node_id": s.nodeID,
		"key":     req.Key,
	})
}

func (s *Server) handleInternalGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	value, err := s.local.Get(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if value == nil {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"value":   json.RawMessage(value),
		"node_id": s.nodeID,
	})
}

func (s *Server) handlePublicSet(w http.ResponseWriter, r *http.Request) {
	var req setRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	if err := s.dist.Set(r.Context(), req.Key, req.Value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "success",
		"node_id": s.nodeID,
		"key":     req.Key,
	})
}

func (s *Server) handlePublicGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	value, err := s.dist.Get(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if value == nil {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"key":     key,
		"value":   json.RawMessage(value),
		"node_id": s.nodeID,
	})
}

func (s *Server) handlePublicDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	if err := s.dist.Delete(r.Context(), key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "deleted",
		"node_id": s.nodeID,
		"key":     key,
	})
}

func (s *Server) handleInternalDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "missing key", http.StatusBadRequest)
		return
	}

	if err := s.local.Delete(r.Context(), key); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "deleted",
		"node_id": s.nodeID,
		"key":     key,
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
