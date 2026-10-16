package noncestore

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNonceStore_BasicLifecycle(t *testing.T) {
	tempDir := t.TempDir()
	ledgerPath := filepath.Join(tempDir, "test_nonce_ledger.jsonl")

	store, err := New(ledgerPath)
	if err != nil {
		t.Fatalf("Failed to initialize NonceStore: %v", err)
	}
	defer store.Close()

	address := "0xAlice123"

	// 1. Initial latest nonce should be 0
	if latest := store.Latest(address); latest != 0 {
		t.Errorf("Expected initial nonce 0, got %d", latest)
	}

	// 2. Reject nonce 0
	if err := store.CheckAndAccept(address, 0, "trace-0"); err == nil {
		t.Errorf("Expected error for nonce 0, got nil")
	}

	// 3. Accept nonce 1
	if err := store.CheckAndAccept(address, 1, "trace-1"); err != nil {
		t.Fatalf("Failed to accept nonce 1: %v", err)
	}
	if latest := store.Latest(address); latest != 1 {
		t.Errorf("Expected latest nonce 1, got %d", latest)
	}

	// 4. Reject duplicate nonce 1 (Replay)
	if err := store.CheckAndAccept(address, 1, "trace-1-dup"); err == nil {
		t.Errorf("Expected replay rejection for duplicate nonce 1, got nil")
	}

	if !store.IsReplay(address, 1) {
		t.Errorf("Expected IsReplay(address, 1) to be true")
	}

	// 5. Accept nonce 2
	if err := store.CheckAndAccept(address, 2, "trace-2"); err != nil {
		t.Fatalf("Failed to accept nonce 2: %v", err)
	}
	if latest := store.Latest(address); latest != 2 {
		t.Errorf("Expected latest nonce 2, got %d", latest)
	}

	// 6. Verify address records
	records, err := store.AddressRecords(address)
	if err != nil {
		t.Fatalf("Failed to read address records: %v", err)
	}
	if len(records) != 2 {
		t.Errorf("Expected 2 records for address, got %d", len(records))
	}
}

func TestNonceStore_RestartPersistence(t *testing.T) {
	tempDir := t.TempDir()
	ledgerPath := filepath.Join(tempDir, "persist_nonce_ledger.jsonl")

	// Session 1: Create and write nonces
	s1, err := New(ledgerPath)
	if err != nil {
		t.Fatalf("Failed to initialize session 1: %v", err)
	}
	if err := s1.CheckAndAccept("0xBob", 1, "trace-b1"); err != nil {
		t.Fatalf("Failed to accept nonce 1: %v", err)
	}
	if err := s1.CheckAndAccept("0xBob", 2, "trace-b2"); err != nil {
		t.Fatalf("Failed to accept nonce 2: %v", err)
	}
	if err := s1.CheckAndAccept("0xCharlie", 5, "trace-c5"); err != nil {
		t.Fatalf("Failed to accept nonce 5: %v", err)
	}
	s1.Close()

	// Session 2: Reopen from same file (Restart Recovery)
	s2, err := New(ledgerPath)
	if err != nil {
		t.Fatalf("Failed to reopen store in session 2: %v", err)
	}
	defer s2.Close()

	// Verify restored state
	if latest := s2.Latest("0xBob"); latest != 2 {
		t.Errorf("Expected restored latest nonce for Bob to be 2, got %d", latest)
	}
	if latest := s2.Latest("0xCharlie"); latest != 5 {
		t.Errorf("Expected restored latest nonce for Charlie to be 5, got %d", latest)
	}

	// Verify replay rejection persisted across restart
	if err := s2.CheckAndAccept("0xBob", 2, "trace-b2-replay"); err == nil {
		t.Errorf("Expected replay rejection for Bob's nonce 2 after restart, got nil")
	}

	// Accepting new higher nonce works
	if err := s2.CheckAndAccept("0xBob", 3, "trace-b3"); err != nil {
		t.Fatalf("Failed to accept nonce 3 after restart: %v", err)
	}
}

func TestNonceStore_EmptyPathDefault(t *testing.T) {
	// Test creating store with empty path fallback
	defer os.Remove(defaultPath)
	s, err := New("")
	if err != nil {
		t.Fatalf("Failed to create store with empty path: %v", err)
	}
	defer s.Close()

	if s.path != defaultPath {
		t.Errorf("Expected path %s, got %s", defaultPath, s.path)
	}
}
