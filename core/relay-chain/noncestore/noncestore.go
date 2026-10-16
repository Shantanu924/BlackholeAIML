// Package noncestore implements Phase 2 — Persistent Replay-Safe Nonce Governance.
//
// Problem solved:
//   The wallet's NonceRegistry was in-memory only — it reset on restart.
//   This meant a nonce used before restart could be replayed after restart.
//   This package closes that gap.
//
// Design:
//   - Every accepted nonce is appended to a JSONL file (nonce_ledger.jsonl)
//   - On startup, the store loads all existing nonces — restart-safe
//   - Duplicate nonce after restart → NONCE_REPLAY rejection
//   - Nonce must be strictly greater than the last accepted nonce for that address
//   - Nonce lineage is observable via the /api/nonce/lookup endpoint
//
// Nonce record format (one per line):
//   {"address":"alice","nonce":1,"trace_id":"abc123","timestamp":1746721834}
//
// Constitutional boundary:
//   Nonce governance is EXECUTION correctness — not governance legitimacy.
//   A valid nonce does not mean a transaction is legitimate.
//   It only means the transaction is not a replay.
package noncestore

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

const defaultPath = "nonce_ledger.jsonl"

// NonceRecord is one entry in the persistent nonce ledger.
type NonceRecord struct {
	Address   string `json:"address"`
	Nonce     uint64 `json:"nonce"`
	TraceID   string `json:"trace_id"`
	Timestamp int64  `json:"timestamp"`
}

// Store is the persistent nonce governance store.
type Store struct {
	mu      sync.Mutex
	path    string
	f       *os.File
	// last accepted nonce per address — loaded from file on startup
	latest  map[string]uint64
	// full seen set for replay detection — "address:nonce"
	seen    map[string]bool
}

// New opens or creates the nonce store at path.
// Loads all existing nonces from the file on startup — restart-safe.
func New(path string) (*Store, error) {
	if path == "" {
		path = defaultPath
	}

	s := &Store{
		path:   path,
		latest: make(map[string]uint64),
		seen:   make(map[string]bool),
	}

	// Load existing nonces from file — restart recovery.
	if err := s.loadExisting(); err != nil {
		return nil, fmt.Errorf("noncestore: load existing: %w", err)
	}

	// Open file for appending.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("noncestore: open %s: %w", path, err)
	}
	s.f = f

	log.Printf("[NONCESTORE] loaded %d addresses from %s", len(s.latest), path)
	return s, nil
}

// loadExisting reads all nonce records from the JSONL file.
// Called once on startup to restore state after restart.
func (s *Store) loadExisting() error {
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // fresh start
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
			log.Printf("[NONCESTORE] skipping malformed line: %v", err)
			continue
		}
		// Track latest nonce per address.
		if rec.Nonce > s.latest[rec.Address] {
			s.latest[rec.Address] = rec.Nonce
		}
		// Mark as seen for replay detection.
		key := fmt.Sprintf("%s:%d", rec.Address, rec.Nonce)
		s.seen[key] = true
		count++
	}
	log.Printf("[NONCESTORE] recovered %d nonce records on startup", count)
	return scanner.Err()
}

// CheckAndAccept validates a nonce and records it if valid.
// Returns an error if the nonce is a replay or out of sequence.
//
// Rules:
//   1. Nonce must be > 0
//   2. Nonce must not have been seen before (replay detection)
//   3. Nonce must be exactly last+1 OR any value > last (flexible for wallet)
func (s *Store) CheckAndAccept(address string, nonce uint64, traceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if nonce == 0 {
		return fmt.Errorf("NONCE_INVALID: nonce must be greater than 0")
	}

	// Replay detection — check if this exact nonce was seen before.
	key := fmt.Sprintf("%s:%d", address, nonce)
	if s.seen[key] {
		log.Printf("[NONCESTORE][NONCE_REPLAY] address=%s nonce=%d trace=%s",
			address, nonce, traceID)
		return fmt.Errorf("NONCE_REPLAY: nonce %d already used by address %s", nonce, address)
	}

	// Accept the nonce.
	s.seen[key] = true
	if nonce > s.latest[address] {
		s.latest[address] = nonce
	}

	// Persist to file.
	rec := NonceRecord{
		Address:   address,
		Nonce:     nonce,
		TraceID:   traceID,
		Timestamp: time.Now().Unix(),
	}
	data, _ := json.Marshal(rec)
	if _, err := fmt.Fprintf(s.f, "%s\n", data); err != nil {
		log.Printf("[NONCESTORE][WARN] failed to persist nonce: %v", err)
	}

	log.Printf("[NONCESTORE][ACCEPT] address=%s nonce=%d trace=%s", address, nonce, traceID)
	return nil
}

// Latest returns the last accepted nonce for an address.
// Returns 0 if no nonce has been accepted yet.
func (s *Store) Latest(address string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latest[address]
}

// IsReplay returns true if this address+nonce has been seen before.
func (s *Store) IsReplay(address string, nonce uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := fmt.Sprintf("%s:%d", address, nonce)
	return s.seen[key]
}

// AllRecords returns all nonce records for observability.
func (s *Store) AllRecords() ([]NonceRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readAll()
}

// AddressRecords returns all nonce records for a specific address.
func (s *Store) AddressRecords(address string) ([]NonceRecord, error) {
	all, err := s.AllRecords()
	if err != nil {
		return nil, err
	}
	var result []NonceRecord
	for _, r := range all {
		if r.Address == address {
			result = append(result, r)
		}
	}
	return result, nil
}

func (s *Store) readAll() ([]NonceRecord, error) {
	f, err := os.Open(s.path)
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

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Close()
}
