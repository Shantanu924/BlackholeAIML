package tantra

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTantra_BuildIntent(t *testing.T) {
	rt := NewRuntime("http://localhost:8080")

	req := IntentRequest{
		From:    "0xAlicePubkey",
		To:      "0xBobPubkey",
		Amount:  500,
		TokenID: "BHIV",
		Type:    "token_transfer",
	}

	contract, err := rt.BuildIntent(req)
	if err != nil {
		t.Fatalf("BuildIntent failed: %v", err)
	}

	if contract.SchemaVersion != SchemaVersion {
		t.Errorf("Expected schema_version %s, got %s", SchemaVersion, contract.SchemaVersion)
	}
	if contract.TraceID == "" {
		t.Errorf("Expected non-empty trace_id")
	}
	if contract.Nonce != 1 {
		t.Errorf("Expected initial nonce 1, got %d", contract.Nonce)
	}
}

func TestTantra_Submit_Success(t *testing.T) {
	// Mock relay server endpoint /api/relay/submit
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/relay/submit" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(SubmissionResult{
				Success:        true,
				TraceID:        "trace-wallet-submit",
				TransactionID:  "tx-hash-12345",
				ExecutionHash:  "exec-hash-1",
				ValidationHash: "exec-hash-1",
				ReplayHash:     "exec-hash-1",
				FraudDecision:  "APPROVED",
				BlockHeight:    10,
			})
		}
	}))
	defer server.Close()

	rt := NewRuntime(server.URL)
	contract := &IntentContract{
		SchemaVersion: "v1",
		TraceID:       "trace-wallet-submit",
		Type:          "token_transfer",
		From:          "0xAlicePubkey",
		To:            "0xBobPubkey",
		Amount:        100,
		TokenID:       "BHIV",
		Nonce:         1,
		Signature:     "sig-hex",
	}

	resp, err := rt.Submit(contract)
	if err != nil {
		t.Fatalf("Submit failed: %v", err)
	}
	if !resp.Success || resp.TransactionID != "tx-hash-12345" {
		t.Errorf("Expected successful submitted transaction, got %+v", resp)
	}
}
