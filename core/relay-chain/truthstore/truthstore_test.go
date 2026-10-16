package truthstore

import (
	"path/filepath"
	"testing"
)

type mockBlockchainReader struct {
	txs map[string]bool
}

func (m *mockBlockchainReader) FindTransactionByID(txID string) bool {
	return m.txs[txID]
}

func TestTruthStore_ChainingAndVerification(t *testing.T) {
	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "test_truth.jsonl")

	store, err := New(storePath)
	if err != nil {
		t.Fatalf("Failed to initialize truthstore: %v", err)
	}
	defer store.Close()

	// 1. Append record 1
	r1 := Record{
		TraceID:        "trace-1",
		ExecutionHash:  "exec-hash-1",
		ValidationHash: "exec-hash-1",
		ReplayHash:     "exec-hash-1",
		TxHash:         "tx-hash-1",
		FraudDecision:  "APPROVED",
		Timestamp:      1000,
	}
	if err := store.Append(r1); err != nil {
		t.Fatalf("Failed to append r1: %v", err)
	}

	// 2. Append record 2
	r2 := Record{
		TraceID:        "trace-2",
		ExecutionHash:  "exec-hash-2",
		ValidationHash: "exec-hash-2",
		ReplayHash:     "exec-hash-2",
		TxHash:         "tx-hash-2",
		FraudDecision:  "APPROVED",
		Timestamp:      1001,
	}
	if err := store.Append(r2); err != nil {
		t.Fatalf("Failed to append r2: %v", err)
	}

	// 3. Verify unbroken chain
	ok, brokenIdx, err := store.VerifyChain()
	if err != nil {
		t.Fatalf("VerifyChain error: %v", err)
	}
	if !ok {
		t.Errorf("Expected chain to be valid, broken at index %d", brokenIdx)
	}

	// 4. Find record by TxHash and TraceID
	foundR1, ok1 := store.FindByTxHash("tx-hash-1")
	if !ok1 || foundR1.TraceID != "trace-1" {
		t.Errorf("Expected to find record by TxHash")
	}

	foundR2, ok2 := store.FindByTraceID("trace-2")
	if !ok2 || foundR2.TxHash != "tx-hash-2" {
		t.Errorf("Expected to find record by TraceID")
	}

	// 5. Test Verify against mock blockchain
	mockBC := &mockBlockchainReader{txs: map[string]bool{"tx-hash-1": true}}
	vRes1 := store.Verify("tx-hash-1", mockBC)
	if !vRes1.Found || !vRes1.OnChain || !vRes1.HashesMatch {
		t.Errorf("Expected verified transaction on chain with matching hashes")
	}
}
