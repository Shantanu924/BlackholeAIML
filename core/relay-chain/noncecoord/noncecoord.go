// Package noncecoord implements cross-node nonce coordination (Scope 2).
//
// Problem:
//   Each relay node has its own nonce_ledger.jsonl.
//   Without coordination, the same nonce can be accepted on node A and node B
//   independently — creating cross-node replay ambiguity.
//
// Solution:
//   A lightweight nonce coordinator service that:
//   1. Accepts nonce registration requests from relay nodes
//   2. Maintains a global seen-set across all nodes
//   3. Rejects duplicate nonces regardless of which node submitted them
//   4. Persists its own ledger for restart recovery
//   5. Exposes observability endpoints for audit
//
// Architecture:
//   Relay Node A ──┐
//   Relay Node B ──┼──► Nonce Coordinator ──► global_nonce_ledger.jsonl
//   Relay Node C ──┘
//
// Relay nodes call the coordinator BEFORE accepting a nonce locally.
// If the coordinator rejects, the relay rejects — NONCE_REPLAY.
// If the coordinator accepts, the relay also records locally (defense in depth).
//
// Configuration:
//   NONCE_COORDINATOR_URL — URL of the coordinator service
//   If not set, coordination is skipped (single-node mode, backward compatible).
//
// Constitutional boundary:
//   Nonce coordination is EXECUTION correctness — not governance legitimacy.
//   A globally unique nonce does not mean a transaction is legitimate.
package noncecoord

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const defaultLedgerPath = "global_nonce_ledger.jsonl"

// NonceRecord is one entry in the global nonce ledger.
type NonceRecord struct {
	Address   string `json:"address"`
	Nonce     uint64 `json:"nonce"`
	TraceID   string `json:"trace_id"`
	NodeID    string `json:"node_id"`   // which relay node submitted this
	Timestamp int64  `json:"timestamp"`
}

// CheckRequest is sent by relay nodes to the coordinator.
type CheckRequest struct {
	Address string `json:"address"`
	Nonce   uint64 `json:"nonce"`
	TraceID string `json:"trace_id"`
	NodeID  string `json:"node_id"`
}

// CheckResponse is returned by the coordinator.
type CheckResponse struct {
	Accepted        bool   `json:"accepted"`
	Address         string `json:"address"`
	Nonce           uint64 `json:"nonce"`
	TraceID         string `json:"trace_id"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	ConflictNodeID  string `json:"conflict_node_id,omitempty"` // which node already used this nonce
}

// Coordinator is the global nonce coordination service.
type Coordinator struct {
	mu   sync.Mutex
	path string
	f    *os.File
	// seen: "address:nonce" -> nodeID that first accepted it
	seen   map[string]string
	latest map[string]uint64
}

// New creates a new coordinator, loading existing records from disk.
func New(ledgerPath string) (*Coordinator, error) {
	if ledgerPath == "" {
		ledgerPath = defaultLedgerPath
	}
	c := &Coordinator{
		path:   ledgerPath,
		seen:   make(map[string]string),
		latest: make(map[string]uint64),
	}
	if err := c.loadExisting(); err != nil {
		return nil, fmt.Errorf("noncecoord: load: %w", err)
	}
	f, err := os.OpenFile(ledgerPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("noncecoord: open %s: %w", ledgerPath, err)
	}
	c.f = f
	log.Printf("[NONCECOORD] loaded %d addresses from %s", len(c.latest), ledgerPath)
	return c, nil
}

func (c *Coordinator) loadExisting() error {
	f, err := os.Open(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec NonceRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		key := fmt.Sprintf("%s:%d", rec.Address, rec.Nonce)
		c.seen[key] = rec.NodeID
		if rec.Nonce > c.latest[rec.Address] {
			c.latest[rec.Address] = rec.Nonce
		}
		count++
	}
	log.Printf("[NONCECOORD] recovered %d nonce records on startup", count)
	return scanner.Err()
}

// CheckAndAccept validates a nonce globally and records it if valid.
// Returns an error if the nonce was already used by ANY node.
func (c *Coordinator) CheckAndAccept(req CheckRequest) CheckResponse {
	c.mu.Lock()
	defer c.mu.Unlock()

	if req.Nonce == 0 {
		return CheckResponse{
			Accepted:        false,
			Address:         req.Address,
			Nonce:           req.Nonce,
			TraceID:         req.TraceID,
			RejectionReason: "NONCE_INVALID: nonce must be greater than 0",
		}
	}

	key := fmt.Sprintf("%s:%d", req.Address, req.Nonce)
	if existingNode, seen := c.seen[key]; seen {
		log.Printf("[NONCECOORD][NONCE_REPLAY] address=%s nonce=%d trace=%s conflict_node=%s submitting_node=%s",
			req.Address, req.Nonce, req.TraceID, existingNode, req.NodeID)
		return CheckResponse{
			Accepted:        false,
			Address:         req.Address,
			Nonce:           req.Nonce,
			TraceID:         req.TraceID,
			RejectionReason: fmt.Sprintf("NONCE_REPLAY: nonce %d already used by address %s on node %s", req.Nonce, req.Address, existingNode),
			ConflictNodeID:  existingNode,
		}
	}

	// Accept globally.
	c.seen[key] = req.NodeID
	if req.Nonce > c.latest[req.Address] {
		c.latest[req.Address] = req.Nonce
	}

	// Persist.
	rec := NonceRecord{
		Address:   req.Address,
		Nonce:     req.Nonce,
		TraceID:   req.TraceID,
		NodeID:    req.NodeID,
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(rec)
	fmt.Fprintf(c.f, "%s\n", data)

	log.Printf("[NONCECOORD][ACCEPT] address=%s nonce=%d trace=%s node=%s",
		req.Address, req.Nonce, req.TraceID, req.NodeID)
	return CheckResponse{
		Accepted: true,
		Address:  req.Address,
		Nonce:    req.Nonce,
		TraceID:  req.TraceID,
	}
}

// Latest returns the latest accepted nonce for an address across all nodes.
func (c *Coordinator) Latest(address string) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.latest[address]
}

// SeenCount returns the number of unique nonces tracked across all nodes.
func (c *Coordinator) SeenCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.seen)
}

// Close closes the coordinator ledger file handle.
func (c *Coordinator) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.f != nil {
		return c.f.Close()
	}
	return nil
}

// AllRecords returns all nonce records for observability.
func (c *Coordinator) AllRecords() ([]NonceRecord, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.readAll()
}

func (c *Coordinator) readAll() ([]NonceRecord, error) {
	f, err := os.Open(c.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var records []NonceRecord
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec NonceRecord
		if err := json.Unmarshal(line, &rec); err != nil {
			continue
		}
		records = append(records, rec)
	}
	return records, scanner.Err()
}

// StartServer starts the nonce coordinator HTTP server.
func StartServer(port int, ledgerPath string) error {
	coord, err := New(ledgerPath)
	if err != nil {
		return fmt.Errorf("noncecoord: init: %w", err)
	}

	mux := http.NewServeMux()

	// POST /nonce/check — relay nodes call this before accepting a nonce
	mux.HandleFunc("/nonce/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req CheckRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(CheckResponse{
				Accepted:        false,
				RejectionReason: "invalid request: " + err.Error(),
			})
			return
		}
		resp := coord.CheckAndAccept(req)
		w.Header().Set("Content-Type", "application/json")
		if !resp.Accepted {
			w.WriteHeader(http.StatusConflict)
		}
		json.NewEncoder(w).Encode(resp)
	})

	// GET /nonce/lookup?address=<addr> — observability
	mux.HandleFunc("/nonce/lookup", func(w http.ResponseWriter, r *http.Request) {
		address := r.URL.Query().Get("address")
		if address == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": "address required"})
			return
		}
		latest := coord.Latest(address)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"address":      address,
			"latest_nonce": latest,
			"next_nonce":   latest + 1,
		})
	})

	// GET /nonce/records — full global ledger
	mux.HandleFunc("/nonce/records", func(w http.ResponseWriter, r *http.Request) {
		records, err := coord.AllRecords()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
			return
		}
		if records == nil {
			records = []NonceRecord{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":   len(records),
			"records": records,
		})
	})

	// GET /health
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "ok",
			"service": "nonce-coordinator",
		})
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("[NONCECOORD] starting on %s ledger=%s", addr, ledgerPath)
	return http.ListenAndServe(addr, mux)
}

// ── RELAY-SIDE CLIENT ─────────────────────────────────────────────────────────

// coordinatorURL returns the configured coordinator URL.
// If not set, returns "" — coordination is skipped (single-node mode).
func CoordinatorURL() string {
	return os.Getenv("NONCE_COORDINATOR_URL")
}

// CheckWithCoordinator calls the coordinator before accepting a nonce locally.
// Returns nil if accepted or if coordinator is not configured.
// Returns an error with NONCE_REPLAY if the coordinator rejects.
func CheckWithCoordinator(address string, nonce uint64, traceID, nodeID string) error {
	url := CoordinatorURL()
	if url == "" {
		// No coordinator configured — single-node mode, skip.
		return nil
	}

	req := CheckRequest{
		Address: address,
		Nonce:   nonce,
		TraceID: traceID,
		NodeID:  nodeID,
	}
	body, _ := json.Marshal(req)

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(url+"/nonce/check", "application/json",
		bytes.NewReader(body))
	if err != nil {
		// Coordinator unreachable — fail-closed: reject the nonce.
		// This prevents cross-node ambiguity when coordinator is down.
		log.Printf("[NONCECOORD][FAIL_CLOSED] coordinator unreachable: %v — rejecting nonce", err)
		return fmt.Errorf("NONCE_COORD_UNAVAILABLE: coordinator unreachable — nonce rejected for safety")
	}
	defer resp.Body.Close()

	var cr CheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		log.Printf("[NONCECOORD][WARN] bad coordinator response: %v — rejecting nonce", err)
		return fmt.Errorf("NONCE_COORD_ERROR: bad coordinator response")
	}

	if !cr.Accepted {
		log.Printf("[NONCECOORD][NONCE_REPLAY] address=%s nonce=%d conflict_node=%s",
			address, nonce, cr.ConflictNodeID)
		return fmt.Errorf("%s", cr.RejectionReason)
	}

	log.Printf("[NONCECOORD][ACCEPTED] address=%s nonce=%d trace=%s", address, nonce, traceID)
	return nil
}

// bufioReader is no longer needed — using bytes.NewReader directly.
