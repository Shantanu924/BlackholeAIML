package noncecoord

import (
	"path/filepath"
	"testing"
)

func TestNonceCoordinator_CrossNodeCoordination(t *testing.T) {
	tempDir := t.TempDir()
	ledgerPath := filepath.Join(tempDir, "global_nonce_ledger.jsonl")

	coord, err := New(ledgerPath)
	if err != nil {
		t.Fatalf("Failed to create coordinator: %v", err)
	}
	defer coord.Close()

	addr := "0xCoordinatorUser"

	// 1. Node A submits nonce 1 -> should be accepted
	reqA := CheckRequest{
		Address: addr,
		Nonce:   1,
		TraceID: "trace-node-a-1",
		NodeID:  "relay-node-A",
	}
	respA := coord.CheckAndAccept(reqA)
	if !respA.Accepted {
		t.Fatalf("Expected Node A submission to be accepted, got rejected: %s", respA.RejectionReason)
	}

	// 2. Node B submits identical nonce 1 -> must be rejected (cross-node replay protection)
	reqB := CheckRequest{
		Address: addr,
		Nonce:   1,
		TraceID: "trace-node-b-1",
		NodeID:  "relay-node-B",
	}
	respB := coord.CheckAndAccept(reqB)
	if respB.Accepted {
		t.Fatalf("Expected Node B duplicate submission to be rejected, got accepted")
	}
	if respB.ConflictNodeID != "relay-node-A" {
		t.Errorf("Expected ConflictNodeID 'relay-node-A', got '%s'", respB.ConflictNodeID)
	}

	// 3. Node B submits nonce 2 -> should be accepted
	reqB2 := CheckRequest{
		Address: addr,
		Nonce:   2,
		TraceID: "trace-node-b-2",
		NodeID:  "relay-node-B",
	}
	respB2 := coord.CheckAndAccept(reqB2)
	if !respB2.Accepted {
		t.Fatalf("Expected Node B nonce 2 to be accepted, got rejected: %s", respB2.RejectionReason)
	}

	// 4. Verify seen count
	if seen := coord.SeenCount(); seen != 2 {
		t.Errorf("Expected 2 seen nonces, got %d", seen)
	}
}

func TestNonceCoordinator_Persistence(t *testing.T) {
	tempDir := t.TempDir()
	ledgerPath := filepath.Join(tempDir, "global_persist_ledger.jsonl")

	// Session 1: Write nonces
	c1, err := New(ledgerPath)
	if err != nil {
		t.Fatalf("Failed to initialize coordinator session 1: %v", err)
	}
	resp := c1.CheckAndAccept(CheckRequest{Address: "0xPersistentUser", Nonce: 10, TraceID: "trace-10", NodeID: "node-1"})
	if !resp.Accepted {
		t.Fatalf("Expected nonce 10 accepted in session 1")
	}
	c1.Close()

	// Session 2: Reopen from disk
	c2, err := New(ledgerPath)
	if err != nil {
		t.Fatalf("Failed to reopen coordinator in session 2: %v", err)
	}
	defer c2.Close()

	// Duplicate check across restart
	respDup := c2.CheckAndAccept(CheckRequest{Address: "0xPersistentUser", Nonce: 10, TraceID: "trace-10-dup", NodeID: "node-2"})
	if respDup.Accepted {
		t.Fatalf("Expected nonce 10 rejected after restart recovery, got accepted")
	}
}
