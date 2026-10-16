// Package akashic implements the AKASHIC lineage store.
//
// AKASHIC is the append-only execution memory of TANTRA.
// Every PDV-passed, Bucket-accepted transaction appends a lineage entry.
// Entries are chain-linked (prev_hash → entry_hash) for tamper detection.
// Reconstruct() replays all entries to verify final state root equality.
//
// Write order enforced by server.go:
//   PDV PASS → Blockchain Write → Bucket Write → AKASHIC Append
//
// No direct writes without PDV PASS are allowed.
package akashic

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const defaultPath = "akashic_lineage.jsonl"

// LineageEntry is one immutable record in the AKASHIC lineage.
type LineageEntry struct {
	TraceID        string `json:"trace_id"`        // immutable across entire flow
	TxHash         string `json:"tx_hash"`         // blockchain transaction ID
	ExecutionHash  string `json:"execution_hash"`  // PDV execution hash
	ValidationHash string `json:"validation_hash"` // PDV validation hash
	ReplayHash     string `json:"replay_hash"`     // PDV replay hash
	FraudDecision  string `json:"fraud_decision"`  // Sarathi/DGIC decision
	BlockHeight    uint64 `json:"block_height"`    // blockchain height at write time
	StateRootHash  string `json:"state_root_hash"` // hash of state after this tx
	Timestamp      int64  `json:"timestamp"`
	PrevHash       string `json:"prev_hash"`  // chain link to previous entry
	EntryHash      string `json:"entry_hash"` // SHA-256 of this entry's content
}

// ReconstructionResult is returned by Reconstruct.
type ReconstructionResult struct {
	TotalEntries   int    `json:"total_entries"`
	ChainIntact    bool   `json:"chain_intact"`
	BrokenAt       int    `json:"broken_at,omitempty"`
	FinalStateRoot string `json:"final_state_root"`
	Verified       bool   `json:"verified"`
	Message        string `json:"message"`
}

// Store is the AKASHIC lineage store.
type Store struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

// New opens or creates the AKASHIC lineage store at path.
func New(path string) (*Store, error) {
	if path == "" {
		path = defaultPath
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("akashic: open %s: %w", path, err)
	}
	return &Store{path: path, f: f}, nil
}

// entryChainHash computes SHA-256 over an entry's core fields (excluding entry_hash).
func entryChainHash(e LineageEntry) string {
	core := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%s|%d|%s",
		e.TraceID, e.TxHash, e.ExecutionHash, e.ValidationHash,
		e.ReplayHash, e.FraudDecision, e.BlockHeight,
		e.StateRootHash, e.Timestamp, e.PrevHash,
	)
	sum := sha256.Sum256([]byte(core))
	return hex.EncodeToString(sum[:])
}

// Append adds a new lineage entry. Called only after PDV PASS + Bucket acceptance.
func (s *Store) Append(e LineageEntry) error {
	if e.Timestamp == 0 {
		e.Timestamp = time.Now().Unix()
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	records, _ := s.readAll()
	if len(records) > 0 {
		e.PrevHash = records[len(records)-1].EntryHash
	} else {
		e.PrevHash = "genesis"
	}
	e.EntryHash = entryChainHash(e)

	data, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("akashic: marshal: %w", err)
	}
	_, err = fmt.Fprintf(s.f, "%s\n", data)
	if err != nil {
		return fmt.Errorf("akashic: write: %w", err)
	}
	log.Printf("[AKASHIC][APPEND] trace=%s tx=%s entry_hash=%s", e.TraceID, e.TxHash, e.EntryHash)
	return nil
}

// FindByTraceID returns the lineage entry for a given trace_id.
func (s *Store) FindByTraceID(traceID string) (LineageEntry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, _ := s.readAll()
	for _, r := range records {
		if r.TraceID == traceID {
			return r, true
		}
	}
	return LineageEntry{}, false
}

// All returns all lineage entries.
func (s *Store) All() ([]LineageEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readAll()
}

// Reconstruct replays all lineage entries, verifies chain integrity,
// and computes the final state root. This is the Phase 6 reconstruction proof.
func (s *Store) Reconstruct() ReconstructionResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.readAll()
	if err != nil {
		return ReconstructionResult{Verified: false, Message: "read error: " + err.Error()}
	}
	if len(records) == 0 {
		return ReconstructionResult{
			TotalEntries: 0,
			ChainIntact:  true,
			Verified:     true,
			Message:      "lineage is empty — nothing to reconstruct",
		}
	}

	for i, r := range records {
		// Verify entry_hash integrity.
		expected := entryChainHash(r)
		if expected != r.EntryHash {
			log.Printf("[AKASHIC][TAMPER] entry %d entry_hash mismatch", i)
			return ReconstructionResult{
				TotalEntries: len(records),
				ChainIntact:  false,
				BrokenAt:     i,
				Verified:     false,
				Message:      fmt.Sprintf("chain broken at entry %d: entry_hash mismatch", i),
			}
		}
		// Verify chain link.
		if i > 0 && r.PrevHash != records[i-1].EntryHash {
			log.Printf("[AKASHIC][TAMPER] entry %d prev_hash broken", i)
			return ReconstructionResult{
				TotalEntries: len(records),
				ChainIntact:  false,
				BrokenAt:     i,
				Verified:     false,
				Message:      fmt.Sprintf("chain broken at entry %d: prev_hash mismatch", i),
			}
		}
		// Verify PDV equality within each entry.
		if r.ExecutionHash != r.ValidationHash || r.ValidationHash != r.ReplayHash {
			log.Printf("[AKASHIC][TAMPER] entry %d PDV hash mismatch", i)
			return ReconstructionResult{
				TotalEntries: len(records),
				ChainIntact:  false,
				BrokenAt:     i,
				Verified:     false,
				Message:      fmt.Sprintf("PDV hash mismatch at entry %d: exec=%s val=%s replay=%s", i, r.ExecutionHash, r.ValidationHash, r.ReplayHash),
			}
		}
	}

	finalStateRoot := computeStateRoot(records)
	log.Printf("[AKASHIC][RECONSTRUCT] entries=%d chain=intact final_state_root=%s", len(records), finalStateRoot)

	return ReconstructionResult{
		TotalEntries:   len(records),
		ChainIntact:    true,
		FinalStateRoot: finalStateRoot,
		Verified:       true,
		Message:        "reconstruction successful — lineage intact",
	}
}

// computeStateRoot produces a rolling SHA-256 over all entry hashes in order.
func computeStateRoot(records []LineageEntry) string {
	h := sha256.New()
	for _, r := range records {
		h.Write([]byte(r.EntryHash))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (s *Store) readAll() ([]LineageEntry, error) {
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var records []LineageEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var e LineageEntry
		if err := json.Unmarshal(line, &e); err != nil {
			log.Printf("[AKASHIC] skipping malformed line: %v", err)
			continue
		}
		records = append(records, e)
	}
	return records, scanner.Err()
}

// SimulateCorruption intentionally corrupts the last entry for testing Phase 4.
// Returns the entry_hash that was corrupted so tests can verify detection.
func (s *Store) SimulateCorruption() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.readAll()
	if err != nil || len(records) == 0 {
		return "", fmt.Errorf("no records to corrupt")
	}

	// Read all lines, corrupt the last one.
	data, err := os.ReadFile(s.path)
	if err != nil {
		return "", err
	}

	originalHash := records[len(records)-1].EntryHash
	// Replace last entry_hash with a corrupted value.
	corrupted := string(data)
	corrupted = strings.Replace(corrupted, originalHash, "CORRUPTED"+originalHash[:8], 1)

	if err := os.WriteFile(s.path, []byte(corrupted), 0644); err != nil {
		return "", err
	}

	log.Printf("[AKASHIC][CORRUPTION_SIMULATED] original_hash=%s", originalHash)
	return originalHash, nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Close()
}
