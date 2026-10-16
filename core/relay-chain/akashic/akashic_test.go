package akashic

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAkashic_AppendAndFind(t *testing.T) {
	tmpDir := t.TempDir()
	lineagePath := filepath.Join(tmpDir, "test_akashic.jsonl")

	store, err := New(lineagePath)
	if err != nil {
		t.Fatalf("failed to create akashic store: %v", err)
	}
	defer store.Close()

	entry1 := LineageEntry{
		TraceID:        "trace-001",
		TxHash:         "0xtx1",
		ExecutionHash:  "0xhash_a",
		ValidationHash: "0xhash_a",
		ReplayHash:     "0xhash_a",
		FraudDecision:  "LEGITIMATE",
		BlockHeight:    1,
		StateRootHash:  "0xroot1",
	}

	if err := store.Append(entry1); err != nil {
		t.Fatalf("failed to append entry 1: %v", err)
	}

	found, ok := store.FindByTraceID("trace-001")
	if !ok {
		t.Fatalf("expected to find trace-001")
	}
	if found.TxHash != "0xtx1" {
		t.Fatalf("expected tx 0xtx1, got %s", found.TxHash)
	}
	if found.PrevHash != "genesis" {
		t.Fatalf("expected genesis prev_hash, got %s", found.PrevHash)
	}

	entry2 := LineageEntry{
		TraceID:        "trace-002",
		TxHash:         "0xtx2",
		ExecutionHash:  "0xhash_b",
		ValidationHash: "0xhash_b",
		ReplayHash:     "0xhash_b",
		FraudDecision:  "LEGITIMATE",
		BlockHeight:    2,
		StateRootHash:  "0xroot2",
	}

	if err := store.Append(entry2); err != nil {
		t.Fatalf("failed to append entry 2: %v", err)
	}

	found2, ok := store.FindByTraceID("trace-002")
	if !ok {
		t.Fatalf("expected to find trace-002")
	}
	if found2.PrevHash != found.EntryHash {
		t.Fatalf("expected entry2 prev_hash (%s) to match entry1 entry_hash (%s)", found2.PrevHash, found.EntryHash)
	}
}

func TestAkashic_Reconstruction_Success(t *testing.T) {
	tmpDir := t.TempDir()
	lineagePath := filepath.Join(tmpDir, "test_reconstruct.jsonl")

	store, err := New(lineagePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	entries := []LineageEntry{
		{
			TraceID:        "trace-r1",
			TxHash:         "0xtx_r1",
			ExecutionHash:  "0xsame_hash_1",
			ValidationHash: "0xsame_hash_1",
			ReplayHash:     "0xsame_hash_1",
			FraudDecision:  "LEGITIMATE",
			BlockHeight:    1,
			StateRootHash:  "0xroot1",
		},
		{
			TraceID:        "trace-r2",
			TxHash:         "0xtx_r2",
			ExecutionHash:  "0xsame_hash_2",
			ValidationHash: "0xsame_hash_2",
			ReplayHash:     "0xsame_hash_2",
			FraudDecision:  "LEGITIMATE",
			BlockHeight:    2,
			StateRootHash:  "0xroot2",
		},
	}

	for _, e := range entries {
		if err := store.Append(e); err != nil {
			t.Fatalf("failed to append: %v", err)
		}
	}

	res := store.Reconstruct()
	if !res.Verified {
		t.Fatalf("expected reconstruction to be verified, got message: %s", res.Message)
	}
	if !res.ChainIntact {
		t.Fatalf("expected chain to be intact")
	}
	if res.TotalEntries != 2 {
		t.Fatalf("expected 2 total entries, got %d", res.TotalEntries)
	}
	if res.FinalStateRoot == "" {
		t.Fatalf("expected non-empty final state root")
	}
}

func TestAkashic_TamperDetection(t *testing.T) {
	tmpDir := t.TempDir()
	lineagePath := filepath.Join(tmpDir, "test_tamper.jsonl")

	store, err := New(lineagePath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	e := LineageEntry{
		TraceID:        "trace-tamper",
		TxHash:         "0xtx_orig",
		ExecutionHash:  "0xhash_t",
		ValidationHash: "0xhash_t",
		ReplayHash:     "0xhash_t",
		FraudDecision:  "LEGITIMATE",
		BlockHeight:    1,
		StateRootHash:  "0xroot_t",
	}
	if err := store.Append(e); err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	// Close store handle before modifying file
	store.Close()

	// Corrupt the file content
	content, err := os.ReadFile(lineagePath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	// Tamper a character
	content[15] = 'X'
	if err := os.WriteFile(lineagePath, content, 0644); err != nil {
		t.Fatalf("failed to write tampered file: %v", err)
	}

	// Reopen store
	reopenedStore, err := New(lineagePath)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer reopenedStore.Close()

	res := reopenedStore.Reconstruct()
	if res.Verified && res.ChainIntact {
		t.Fatalf("expected tamper detection to fail verification")
	}
}
