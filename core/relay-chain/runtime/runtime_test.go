package runtime

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/akashic"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/sigverify"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/truthstore"
)

type mockBlockchainWriter struct{}

func (m *mockBlockchainWriter) ProcessTransactionFromRuntime(traceID, txType, from, to, tokenID string, amount, fee, nonce uint64, timestamp int64) (string, uint64, error) {
	return "mock-tx-hash-12345", 101, nil
}

func generateValidContract(t *testing.T, traceID string, nonce uint64) (*schema.TxContract, string, string) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	pubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())
	privKeyHex := hex.EncodeToString(privKey.Serialize())

	verifyReq := sigverify.VerifyRequest{
		TraceID: traceID,
		Type:    "token_transfer",
		From:    pubKeyHex,
		To:      "0xRecipientWallet",
		Amount:  500,
		TokenID: "BHIV",
		Fee:     5,
		Nonce:   nonce,
	}

	sigHex, err := sigverify.Sign(verifyReq, privKeyHex)
	if err != nil {
		t.Fatalf("failed to sign verify request: %v", err)
	}

	contract := &schema.TxContract{
		SchemaVersion: "v1",
		TraceID:       traceID,
		Type:          "token_transfer",
		From:          pubKeyHex,
		To:            "0xRecipientWallet",
		Amount:        500,
		TokenID:       "BHIV",
		Fee:           5,
		Nonce:         nonce,
		Timestamp:     time.Now().Unix(),
		Signature:     sigHex,
	}

	return contract, pubKeyHex, privKeyHex
}

func TestRuntime_Execute_SchemaViolation(t *testing.T) {
	req := ExecutionRequest{
		Contract:   nil,
		Blockchain: &mockBlockchainWriter{},
	}

	res := Execute(req)
	if res.Allowed {
		t.Fatalf("Expected nil contract to be rejected, but Allowed was true")
	}
	if res.FailureStage != StageSchema {
		t.Errorf("Expected failure stage SCHEMA, got %s", res.FailureStage)
	}
	if res.ErrorCode != "SCHEMA_NIL" {
		t.Errorf("Expected error code SCHEMA_NIL, got %s", res.ErrorCode)
	}
}

func TestRuntime_Execute_DistributedPDVRejection(t *testing.T) {
	// Enable strict mode and point agent URLs to dead ports
	os.Setenv("PDV_STRICT_MODE", "true")
	os.Setenv("PDV_EXECUTION_AGENT_URL", "http://127.0.0.1:59991/pdv/execute")
	os.Setenv("PDV_VALIDATION_AGENT_URL", "http://127.0.0.1:59992/pdv/validate")
	os.Setenv("PDV_REPLAY_AGENT_URL", "http://127.0.0.1:59993/pdv/replay")
	defer func() {
		os.Unsetenv("PDV_STRICT_MODE")
		os.Unsetenv("PDV_EXECUTION_AGENT_URL")
		os.Unsetenv("PDV_VALIDATION_AGENT_URL")
		os.Unsetenv("PDV_REPLAY_AGENT_URL")
	}()

	contract, _, _ := generateValidContract(t, "trace-strict-fail", 1)

	req := ExecutionRequest{
		Contract:   contract,
		Blockchain: &mockBlockchainWriter{},
	}

	res := Execute(req)
	if res.Allowed {
		t.Fatalf("expected strict mode to reject when agents are unreachable")
	}
	if res.FailureStage != StagePDV {
		t.Errorf("expected StagePDV, got %s", res.FailureStage)
	}
	if res.ErrorCode != "DISTRIBUTED_PDV_REJECT" {
		t.Errorf("expected DISTRIBUTED_PDV_REJECT, got %s", res.ErrorCode)
	}
}

func TestRuntime_Execute_SignatureRejection(t *testing.T) {
	os.Setenv("PDV_STRICT_MODE", "false")
	defer os.Unsetenv("PDV_STRICT_MODE")

	contract, _, _ := generateValidContract(t, "trace-sig-rej", 2)
	contract.Signature = "bad_deadbeef_signature"

	req := ExecutionRequest{
		Contract:   contract,
		Blockchain: &mockBlockchainWriter{},
	}

	res := Execute(req)
	if res.Allowed {
		t.Fatalf("expected signature rejection, but Allowed was true")
	}
	if res.ErrorCode != "SIGNATURE_REJECT" {
		t.Errorf("expected ErrorCode SIGNATURE_REJECT, got %s", res.ErrorCode)
	}
	if res.FailureStage != StageSchema {
		t.Errorf("expected StageSchema, got %s", res.FailureStage)
	}
}

func TestRuntime_Execute_GovernanceRejection(t *testing.T) {
	os.Setenv("PDV_STRICT_MODE", "false")
	defer os.Unsetenv("PDV_STRICT_MODE")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"decision": "block",
			"reason":   "SARATHI_POLICY_VIOLATION",
		})
	}))
	defer server.Close()

	os.Setenv("SARATHI_URL", server.URL)
	defer os.Unsetenv("SARATHI_URL")

	contract, _, _ := generateValidContract(t, "trace-gov-rej", 3)

	req := ExecutionRequest{
		Contract:   contract,
		Blockchain: &mockBlockchainWriter{},
	}

	res := Execute(req)
	if res.Allowed {
		t.Fatalf("expected governance rejection, but Allowed was true")
	}
	if res.FailureStage != StageGovernance {
		t.Errorf("expected StageGovernance, got %s", res.FailureStage)
	}
	if res.ErrorCode != "GOVERNANCE_REJECT" {
		t.Errorf("expected GOVERNANCE_REJECT, got %s", res.ErrorCode)
	}
}

func TestRuntime_Execute_FullCommitWithStores(t *testing.T) {
	os.Setenv("PDV_STRICT_MODE", "false")
	defer os.Unsetenv("PDV_STRICT_MODE")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"decision": "allow",
		})
	}))
	defer server.Close()

	os.Setenv("SARATHI_URL", server.URL)
	defer os.Unsetenv("SARATHI_URL")

	tmpDir, err := os.MkdirTemp("", "runtime_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	truthPath := filepath.Join(tmpDir, "test_truth.jsonl")
	akashicPath := filepath.Join(tmpDir, "test_akashic.jsonl")

	ts, err := truthstore.New(truthPath)
	if err != nil {
		t.Fatalf("failed to create TruthStore: %v", err)
	}

	ak, err := akashic.New(akashicPath)
	if err != nil {
		t.Fatalf("failed to create AkashicStore: %v", err)
	}

	contract, _, _ := generateValidContract(t, "trace-full-commit", 4)

	req := ExecutionRequest{
		Contract:     contract,
		Blockchain:   &mockBlockchainWriter{},
		TruthStore:   ts,
		AkashicStore: ak,
	}

	res := Execute(req)
	if !res.Allowed {
		t.Fatalf("expected successful execution, but failed: %s", res.RejectionReason)
	}
	if res.TxHash != "mock-tx-hash-12345" {
		t.Errorf("expected tx hash 'mock-tx-hash-12345', got '%s'", res.TxHash)
	}
	if res.BlockHeight != 101 {
		t.Errorf("expected block height 101, got %d", res.BlockHeight)
	}

	// Verify TruthStore record was appended
	rec, found := ts.FindByTraceID("trace-full-commit")
	if !found {
		t.Fatalf("expected TruthStore record with trace_id 'trace-full-commit' to be found")
	}
	if rec.TxHash != "mock-tx-hash-12345" {
		t.Errorf("expected TruthStore record tx_hash 'mock-tx-hash-12345', got '%s'", rec.TxHash)
	}

	// Verify AkashicStore record was appended
	entries, err := ak.All()
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 AkashicStore entry, got %d (err: %v)", len(entries), err)
	}
	if entries[0].TraceID != "trace-full-commit" {
		t.Errorf("expected AkashicStore entry trace_id 'trace-full-commit', got '%s'", entries[0].TraceID)
	}
}
