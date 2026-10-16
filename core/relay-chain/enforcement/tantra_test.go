package enforcement

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/sigverify"
)

// Helper to generate a valid secp256k1 keypair and signed TxPayload for tests
func generateSignedPayload(t *testing.T, traceID string, nonce uint64) (*TxPayload, string, string) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("failed to generate btcec keypair: %v", err)
	}

	pubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())
	privKeyHex := hex.EncodeToString(privKey.Serialize())

	req := sigverify.VerifyRequest{
		TraceID: traceID,
		Type:    "token_transfer",
		From:    pubKeyHex,
		To:      "0xRecipientAddressTest",
		Amount:  1000,
		TokenID: "BHIV",
		Fee:     10,
		Nonce:   nonce,
	}

	sigHex, err := sigverify.Sign(req, privKeyHex)
	if err != nil {
		t.Fatalf("failed to sign verify request: %v", err)
	}

	tx := &TxPayload{
		TraceID:   traceID,
		Type:      "token_transfer",
		From:      pubKeyHex,
		To:        "0xRecipientAddressTest",
		Amount:    1000,
		TokenID:   "BHIV",
		Fee:       10,
		Nonce:     nonce,
		Timestamp: time.Now().Unix(),
		Signature: sigHex,
	}

	return tx, pubKeyHex, privKeyHex
}

// TestDeterministicHash_TimestampExclusion proves that modifying the timestamp
// has zero impact on the resulting deterministic hash (crucial for multi-node and replay determinism).
func TestDeterministicHash_TimestampExclusion(t *testing.T) {
	tx1 := &TxPayload{
		TraceID:   "trace-det-1",
		Type:      "token_transfer",
		From:      "0xAlice",
		To:        "0xBob",
		Amount:    500,
		TokenID:   "BHIV",
		Fee:       5,
		Nonce:     1,
		Timestamp: 1000000000,
		Signature: "sig123",
	}

	tx2 := &TxPayload{
		TraceID:   "trace-det-1",
		Type:      "token_transfer",
		From:      "0xAlice",
		To:        "0xBob",
		Amount:    500,
		TokenID:   "BHIV",
		Fee:       5,
		Nonce:     1,
		Timestamp: 2000000000, // completely different timestamp
		Signature: "sig123",
	}

	h1 := deterministicHash(tx1)
	h2 := deterministicHash(tx2)

	if h1 != h2 {
		t.Fatalf("deterministicHash must exclude timestamp, but hashes differ:\n  h1=%s\n  h2=%s", h1, h2)
	}
}

// TestDeterministicHash_DeterministicZoneSensitivity proves that modifying any
// deterministic zone field produces a completely different hash.
func TestDeterministicHash_DeterministicZoneSensitivity(t *testing.T) {
	base := &TxPayload{
		TraceID:   "trace-det-sens",
		Type:      "token_transfer",
		From:      "0xAlice",
		To:        "0xBob",
		Amount:    100,
		TokenID:   "BHIV",
		Fee:       1,
		Nonce:     1,
		Timestamp: 1779717000,
		Signature: "sigBase",
	}
	baseHash := deterministicHash(base)

	// Mutate amount
	mutAmount := *base
	mutAmount.Amount = 101
	if deterministicHash(&mutAmount) == baseHash {
		t.Errorf("expected hash difference when mutating Amount")
	}

	// Mutate from
	mutFrom := *base
	mutFrom.From = "0xEve"
	if deterministicHash(&mutFrom) == baseHash {
		t.Errorf("expected hash difference when mutating From")
	}

	// Mutate to
	mutTo := *base
	mutTo.To = "0xCharlie"
	if deterministicHash(&mutTo) == baseHash {
		t.Errorf("expected hash difference when mutating To")
	}

	// Mutate nonce
	mutNonce := *base
	mutNonce.Nonce = 2
	if deterministicHash(&mutNonce) == baseHash {
		t.Errorf("expected hash difference when mutating Nonce")
	}

	// Mutate token_id
	mutToken := *base
	mutToken.TokenID = "USDT"
	if deterministicHash(&mutToken) == baseHash {
		t.Errorf("expected hash difference when mutating TokenID")
	}

	// Mutate fee
	mutFee := *base
	mutFee.Fee = 2
	if deterministicHash(&mutFee) == baseHash {
		t.Errorf("expected hash difference when mutating Fee")
	}

	// Mutate signature
	mutSig := *base
	mutSig.Signature = "sigDifferent"
	if deterministicHash(&mutSig) == baseHash {
		t.Errorf("expected hash difference when mutating Signature")
	}
}

// TestPDV_ThreeAgentAgreement verifies that ExecutionAgent, ValidationAgent,
// and ReplayAgent compute identical hashes on the same canonical transaction.
func TestPDV_ThreeAgentAgreement(t *testing.T) {
	tx := &TxPayload{
		TraceID:   "trace-3agent-agreement",
		Type:      "token_transfer",
		From:      "0xAlice",
		To:        "0xBob",
		Amount:    250,
		TokenID:   "BHIV",
		Fee:       2,
		Nonce:     5,
		Timestamp: time.Now().Unix(),
		Signature: "mock_signature_hex",
	}

	execHash, err := ExecutionAgent(tx)
	if err != nil {
		t.Fatalf("ExecutionAgent failed: %v", err)
	}

	valHash, err := ValidationAgent(tx)
	if err != nil {
		t.Fatalf("ValidationAgent failed: %v", err)
	}

	replayHash, err := ReplayAgent(tx)
	if err != nil {
		t.Fatalf("ReplayAgent failed: %v", err)
	}

	if execHash != valHash || valHash != replayHash {
		t.Fatalf("PDV 3-agent agreement failed: exec=%s, val=%s, replay=%s", execHash, valHash, replayHash)
	}
}

// TestEnforce_SignatureVerificationGate verifies that invalid or tampered signatures
// are rejected at the very first gate before PDV computation.
func TestEnforce_SignatureVerificationGate(t *testing.T) {
	tx, _, _ := generateSignedPayload(t, "trace-sig-gate", 1)

	// Tamper signature
	tx.Signature = "bad_signature_deadbeef"

	res := Enforce(tx)
	if res.Allowed {
		t.Fatalf("expected Enforce to reject tampered signature, but Allowed was true")
	}
	if res.SignatureValid {
		t.Errorf("expected SignatureValid to be false")
	}
	if res.RejectionReason == "" {
		t.Errorf("expected a descriptive RejectionReason")
	}
}

// TestEnforce_SarathiFailClosedDefault verifies fail-closed security posture
// when the Sarathi/DGIC fraud service is unreachable.
func TestEnforce_SarathiFailClosedDefault(t *testing.T) {
	tx, _, _ := generateSignedPayload(t, "trace-fail-closed", 2)

	// Point Sarathi to dead port and ensure fail-closed is active
	os.Setenv("SARATHI_URL", "http://127.0.0.1:59999/api/fraud/check")
	os.Setenv("SARATHI_FAIL_CLOSED", "true")
	defer func() {
		os.Unsetenv("SARATHI_URL")
		os.Unsetenv("SARATHI_FAIL_CLOSED")
	}()

	res := Enforce(tx)
	if res.Allowed {
		t.Fatalf("expected Enforce to block when Sarathi is unreachable in fail-closed mode")
	}
	if res.FraudDecision != "block" {
		t.Errorf("expected FraudDecision 'block', got '%s'", res.FraudDecision)
	}
}

// TestEnforce_SarathiFailOpenConfig verifies fail-open capability when explicitly configured.
func TestEnforce_SarathiFailOpenConfig(t *testing.T) {
	tx, _, _ := generateSignedPayload(t, "trace-fail-open", 3)

	os.Setenv("SARATHI_URL", "http://127.0.0.1:59999/api/fraud/check")
	os.Setenv("SARATHI_FAIL_CLOSED", "false") // explicit dev opt-out
	defer func() {
		os.Unsetenv("SARATHI_URL")
		os.Unsetenv("SARATHI_FAIL_CLOSED")
	}()

	res := Enforce(tx)
	if !res.Allowed {
		t.Fatalf("expected Enforce to allow when SARATHI_FAIL_CLOSED=false and service unreachable, got: %s", res.RejectionReason)
	}
	if res.FraudDecision != "allow" {
		t.Errorf("expected FraudDecision 'allow', got '%s'", res.FraudDecision)
	}
}

// TestEnforce_SarathiDecisionBlock verifies transaction rejection when Sarathi returns "block".
func TestEnforce_SarathiDecisionBlock(t *testing.T) {
	// Create mock Sarathi server returning "block"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fraudResponse{
			Decision: "block",
			Reason:   "Address flagged for suspicious illicit velocity",
		})
	}))
	defer server.Close()

	os.Setenv("SARATHI_URL", server.URL)
	defer os.Unsetenv("SARATHI_URL")

	tx, _, _ := generateSignedPayload(t, "trace-fraud-block", 4)

	res := Enforce(tx)
	if res.Allowed {
		t.Fatalf("expected Enforce to reject when Sarathi returns block")
	}
	if res.FraudDecision != "block" {
		t.Errorf("expected FraudDecision 'block', got '%s'", res.FraudDecision)
	}
}

// TestEnforce_FullPipelineSuccess tests end-to-end execution through the complete pipeline
// with valid signature, 3-agent agreement, and Sarathi "allow" response.
func TestEnforce_FullPipelineSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(fraudResponse{
			Decision: "allow",
		})
	}))
	defer server.Close()

	os.Setenv("SARATHI_URL", server.URL)
	defer os.Unsetenv("SARATHI_URL")

	tx, _, _ := generateSignedPayload(t, "trace-full-pass", 5)

	res := Enforce(tx)
	if !res.Allowed {
		t.Fatalf("expected Enforce to succeed, but got rejected: %s", res.RejectionReason)
	}
	if !res.SignatureValid {
		t.Errorf("expected SignatureValid to be true")
	}
	if res.ExecutionHash == "" || res.ValidationHash == "" || res.ReplayHash == "" {
		t.Errorf("expected all 3 PDV hashes to be populated")
	}
	if res.ExecutionHash != res.ValidationHash || res.ValidationHash != res.ReplayHash {
		t.Errorf("expected all 3 PDV hashes to match")
	}
	if res.FraudDecision != "allow" {
		t.Errorf("expected FraudDecision 'allow', got '%s'", res.FraudDecision)
	}
}

// TestInjectTraceID_PreservesAndGenerates verifies that injectTraceID preserves
// pre-existing trace IDs and deterministically generates new ones when empty.
func TestInjectTraceID_PreservesAndGenerates(t *testing.T) {
	// Case 1: Pre-existing trace ID preserved
	tx1 := &TxPayload{
		TraceID: "custom-trace-12345",
		From:    "0xAlice",
		To:      "0xBob",
		Amount:  50,
	}
	injectTraceID(tx1)
	if tx1.TraceID != "custom-trace-12345" {
		t.Errorf("expected custom trace ID to be preserved, got %s", tx1.TraceID)
	}

	// Case 2: Missing trace ID generated
	tx2 := &TxPayload{
		From:   "0xAlice",
		To:     "0xBob",
		Amount: 50,
	}
	injectTraceID(tx2)
	if tx2.TraceID == "" || len(tx2.TraceID) != 16 {
		t.Errorf("expected generated 16-character trace ID, got '%s'", tx2.TraceID)
	}
}
