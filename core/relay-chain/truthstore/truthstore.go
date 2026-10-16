// Package truthstore implements Phase 5 — append-only JSONL truth store.
// Every accepted transaction is persisted with all 4 hashes + txHash.
// Verify() checks stored records against the real blockchain state.
package truthstore

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

const defaultPath = "tantra_truth.jsonl"

type Record struct {
	TraceID        string `json:"trace_id"`
	ExecutionHash  string `json:"execution_hash"`
	ValidationHash string `json:"validation_hash"`
	ReplayHash     string `json:"replay_hash"`
	TxHash         string `json:"tx_hash"`
	FraudDecision  string `json:"fraud_decision"`
	Timestamp      int64  `json:"timestamp"`
	// PrevHash chains this record to the previous one.
	// Any tampering with a past record breaks the chain — detectable on verify.
	PrevHash       string `json:"prev_hash"`
	// EntryHash is SHA-256 of this record's content (excluding entry_hash itself).
	EntryHash      string `json:"entry_hash"`
}

// VerifyResult is returned by Verify — shows whether the stored record
// matches what is actually on the blockchain.
type VerifyResult struct {
	Found          bool   `json:"found"`
	OnChain        bool   `json:"on_chain"`
	HashesMatch    bool   `json:"hashes_match"`
	Record         Record `json:"record"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// BlockchainReader is the minimal interface needed to verify against chain state.
type BlockchainReader interface {
	FindTransactionByID(txID string) bool
}

// bucketURL is the remote endpoint to POST records to (Phase 5 — no local-only storage).
// Set env var TANTRA_BUCKET_URL to enable, e.g. https://your-bucket-api/records
var bucketURL = os.Getenv("TANTRA_BUCKET_URL")

type Store struct {
	mu   sync.Mutex
	path string
	f    *os.File
}

func New(path string) (*Store, error) {
	if path == "" {
		path = defaultPath
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("truthstore: open %s: %w", path, err)
	}
	return &Store{path: path, f: f}, nil
}

// chainHash computes SHA-256 over a record's core fields (excluding entry_hash/prev_hash).
func chainHash(r Record) string {
	core := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%s",
		r.TraceID, r.ExecutionHash, r.ValidationHash, r.ReplayHash,
		r.TxHash, r.FraudDecision, r.Timestamp, r.PrevHash,
	)
	sum := sha256.Sum256([]byte(core))
	return hex.EncodeToString(sum[:])
}

func (s *Store) Append(r Record) error {
	if r.Timestamp == 0 {
		r.Timestamp = time.Now().Unix()
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// Chain: set PrevHash to the EntryHash of the last record.
	records, _ := s.readAll()
	if len(records) > 0 {
		r.PrevHash = records[len(records)-1].EntryHash
	} else {
		r.PrevHash = "genesis"
	}
	r.EntryHash = chainHash(r)

	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("truthstore: marshal: %w", err)
	}
	// Local append (primary).
	_, err = fmt.Fprintf(s.f, "%s\n", data)
	if err != nil {
		return err
	}
	// Remote bucket write (Phase 5 — no local-only storage).
	if bucketURL != "" {
		go func() {
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Post(bucketURL, "application/json", bytes.NewReader(data))
			if err != nil {
				log.Printf("[truthstore] bucket write failed: %v", err)
				return
			}
			resp.Body.Close()
			if resp.StatusCode >= 300 {
				log.Printf("[truthstore] bucket returned HTTP %d", resp.StatusCode)
			}
		}()
	}
	return nil
}

// VerifyChain walks every record and confirms the chain is unbroken.
// Returns false + the index of the first broken link if tampered.
func (s *Store) VerifyChain() (bool, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, err := s.readAll()
	if err != nil {
		return false, -1, err
	}
	for i, r := range records {
		expected := chainHash(r)
		if expected != r.EntryHash {
			log.Printf("[truthstore][TAMPER] record %d entry_hash mismatch", i)
			return false, i, nil
		}
		if i > 0 && r.PrevHash != records[i-1].EntryHash {
			log.Printf("[truthstore][TAMPER] record %d prev_hash broken", i)
			return false, i, nil
		}
	}
	return true, -1, nil
}

// readAll reads all records from the JSONL file.
func (s *Store) readAll() ([]Record, error) {
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var records []Record
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(line, &r); err != nil {
			log.Printf("[truthstore] skipping malformed line: %v", err)
			continue
		}
		records = append(records, r)
	}
	return records, scanner.Err()
}

// FindByTxHash looks up a stored record by blockchain txHash.
func (s *Store) FindByTxHash(txHash string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, _ := s.readAll()
	for _, r := range records {
		if r.TxHash == txHash {
			return r, true
		}
	}
	return Record{}, false
}

// FindByTraceID looks up a stored record by trace_id.
func (s *Store) FindByTraceID(traceID string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	records, _ := s.readAll()
	for _, r := range records {
		if r.TraceID == traceID {
			return r, true
		}
	}
	return Record{}, false
}

// Verify checks a stored record against the real blockchain state.
// It confirms:
//  1. The record exists in the truth store.
//  2. The txHash exists on-chain (not local-only).
//  3. The three hashes are still equal (replay integrity).
func (s *Store) Verify(txHash string, bc BlockchainReader) VerifyResult {
	record, found := s.FindByTxHash(txHash)
	if !found {
		return VerifyResult{
			Found:           false,
			RejectionReason: fmt.Sprintf("tx_hash %s not found in truth store", txHash),
		}
	}

	// Check against real blockchain state — not local-only.
	onChain := bc.FindTransactionByID(txHash)
	if !onChain {
		log.Printf("[truthstore][Verify] tx=%s found in store but NOT on chain", txHash)
		return VerifyResult{
			Found:           true,
			OnChain:         false,
			Record:          record,
			RejectionReason: "transaction exists in truth store but not found on blockchain",
		}
	}

	// Replay integrity: all three hashes must still match.
	hashesMatch := record.ExecutionHash != "" &&
		record.ExecutionHash == record.ValidationHash &&
		record.ValidationHash == record.ReplayHash

	if !hashesMatch {
		return VerifyResult{
			Found:           true,
			OnChain:         true,
			HashesMatch:     false,
			Record:          record,
			RejectionReason: fmt.Sprintf("hash mismatch in stored record: exec=%s val=%s replay=%s",
				record.ExecutionHash, record.ValidationHash, record.ReplayHash),
		}
	}

	log.Printf("[truthstore][Verify] tx=%s trace=%s VERIFIED on-chain", txHash, record.TraceID)
	return VerifyResult{
		Found:       true,
		OnChain:     true,
		HashesMatch: true,
		Record:      record,
	}
}

// All returns every record in the store (for audit/debug).
func (s *Store) All() ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.readAll()
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.f.Close()
}
