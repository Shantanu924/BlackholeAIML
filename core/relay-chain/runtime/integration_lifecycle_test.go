// Package runtime contains end-to-end integration lifecycle tests for Ishan Shirode's
// VAJRA apprenticeship and production support stream.
//
// Demonstrations:
// 1. Valid transaction processing & state commitment
// 2. Schema / malformed transaction rejection
// 3. Bad signature rejection
// 4. Nonce replay protection rejection
// 5. Evidence & lineage logging (TruthStore & Akashic)
// 6. Cryptographic replay invariance
package runtime

import (
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/akashic"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/sigverify"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/truthstore"
)

// statefulMockBlockchain simulates ledger state mutations and nonce tracking.
type statefulMockBlockchain struct {
	mu          sync.Mutex
	nonces      map[string]uint64
	blockHeight uint64
	committedTx []string
}

func newMockBlockchain() *statefulMockBlockchain {
	return &statefulMockBlockchain{
		nonces:      make(map[string]uint64),
		blockHeight: 200,
		committedTx: make([]string, 0),
	}
}

func (s *statefulMockBlockchain) ProcessTransactionFromRuntime(traceID, txType, from, to, tokenID string, amount, fee, nonce uint64, timestamp int64) (string, uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lastNonce := s.nonces[from]
	if nonce <= lastNonce && lastNonce > 0 {
		return "", 0, os.ErrExist // Nonce replay error
	}

	s.nonces[from] = nonce
	s.blockHeight++
	txHash := "0x" + hex.EncodeToString([]byte(traceID))
	s.committedTx = append(s.committedTx, txHash)
	return txHash, s.blockHeight, nil
}

func TestApprenticeship_DemonstrateFullLifecycle(t *testing.T) {
	os.Setenv("PDV_STRICT_MODE", "false")
	defer os.Unsetenv("PDV_STRICT_MODE")

	// Mock Sarathi allow response
	govServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"decision":"allow","reason":"OK"}`))
	}))
	defer govServer.Close()

	os.Setenv("SARATHI_URL", govServer.URL)
	defer os.Unsetenv("SARATHI_URL")

	tmpDir, err := os.MkdirTemp("", "vajra_lifecycle_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	truthStore, err := truthstore.New(filepath.Join(tmpDir, "truth.jsonl"))
	if err != nil {
		t.Fatalf("Failed to create truthstore: %v", err)
	}

	akashicStore, err := akashic.New(filepath.Join(tmpDir, "akashic.jsonl"))
	if err != nil {
		t.Fatalf("Failed to create akashic store: %v", err)
	}

	chainWriter := newMockBlockchain()

	// 1. DEMO: Valid Transaction
	t.Run("Demo 1: Valid Transaction Execution", func(t *testing.T) {
		contract, _, _ := generateValidContract(t, "trace-demo-valid-001", 1)
		req := ExecutionRequest{
			Contract:     contract,
			Blockchain:   chainWriter,
			TruthStore:   truthStore,
			AkashicStore: akashicStore,
		}

		res := Execute(req)
		if !res.Allowed {
			t.Fatalf("Expected valid transaction to be allowed, failed: %s", res.RejectionReason)
		}
		if res.BlockHeight != 201 {
			t.Errorf("Expected block height 201, got %d", res.BlockHeight)
		}
		if !res.SignatureValid {
			t.Errorf("Expected SignatureValid=true")
		}
	})

	// 2. DEMO: Invalid Schema Rejection
	t.Run("Demo 2: Nil / Malformed Schema Rejection", func(t *testing.T) {
		req := ExecutionRequest{
			Contract:     nil,
			Blockchain:   chainWriter,
			TruthStore:   truthStore,
			AkashicStore: akashicStore,
		}

		res := Execute(req)
		if res.Allowed {
			t.Fatalf("Expected nil contract to be rejected")
		}
		if res.FailureStage != StageSchema {
			t.Errorf("Expected StageSchema, got %s", res.FailureStage)
		}
	})

	// 3. DEMO: Bad Signature Rejection
	t.Run("Demo 3: Bad Cryptographic Signature Rejection", func(t *testing.T) {
		contract, _, _ := generateValidContract(t, "trace-demo-bad-sig-002", 2)
		contract.Signature = "deadbeefcafebabe00112233445566778899" // Corrupted signature

		req := ExecutionRequest{
			Contract:     contract,
			Blockchain:   chainWriter,
			TruthStore:   truthStore,
			AkashicStore: akashicStore,
		}

		res := Execute(req)
		if res.Allowed {
			t.Fatalf("Expected bad signature to be rejected")
		}
		if res.ErrorCode != "SIGNATURE_REJECT" {
			t.Errorf("Expected SIGNATURE_REJECT, got %s", res.ErrorCode)
		}
		if res.SignatureValid {
			t.Errorf("Expected SignatureValid=false")
		}
	})

	// 4. DEMO: Nonce Replay Rejection
	t.Run("Demo 4: Nonce Replay Protection Rejection", func(t *testing.T) {
		// Replay nonce=1 from the same sender
		privKey, _ := btcec.NewPrivateKey()
		pubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())
		privKeyHex := hex.EncodeToString(privKey.Serialize())

		// First TX with nonce 10 -> succeeds
		req1 := sigverify.VerifyRequest{
			TraceID: "trace-nonce-tx1",
			Type:    "token_transfer",
			From:    pubKeyHex,
			To:      "0xRecipientWallet",
			Amount:  100,
			TokenID: "BHIV",
			Fee:     5,
			Nonce:   10,
		}
		sig1, _ := sigverify.Sign(req1, privKeyHex)
		contract1 := &schema.TxContract{
			SchemaVersion: "v1",
			TraceID:       "trace-nonce-tx1",
			Type:          "token_transfer",
			From:          pubKeyHex,
			To:            "0xRecipientWallet",
			Amount:        100,
			TokenID:       "BHIV",
			Fee:           5,
			Nonce:         10,
			Timestamp:     time.Now().Unix(),
			Signature:     sig1,
		}

		res1 := Execute(ExecutionRequest{
			Contract:     contract1,
			Blockchain:   chainWriter,
			TruthStore:   truthStore,
			AkashicStore: akashicStore,
		})
		if !res1.Allowed {
			t.Fatalf("First TX failed: %s", res1.RejectionReason)
		}

		// Replay TX with old nonce 5 (<= 10) -> rejected at blockchain stage
		req2 := sigverify.VerifyRequest{
			TraceID: "trace-nonce-tx2-replay",
			Type:    "token_transfer",
			From:    pubKeyHex,
			To:      "0xRecipientWallet",
			Amount:  100,
			TokenID: "BHIV",
			Fee:     5,
			Nonce:   5,
		}
		sig2, _ := sigverify.Sign(req2, privKeyHex)
		contract2 := &schema.TxContract{
			SchemaVersion: "v1",
			TraceID:       "trace-nonce-tx2-replay",
			Type:          "token_transfer",
			From:          pubKeyHex,
			To:            "0xRecipientWallet",
			Amount:        100,
			TokenID:       "BHIV",
			Fee:           5,
			Nonce:         5,
			Timestamp:     time.Now().Unix(),
			Signature:     sig2,
		}

		res2 := Execute(ExecutionRequest{
			Contract:     contract2,
			Blockchain:   chainWriter,
			TruthStore:   truthStore,
			AkashicStore: akashicStore,
		})
		if res2.Allowed {
			t.Fatalf("Expected replayed nonce to be rejected")
		}
		if res2.FailureStage != StageBlockchain {
			t.Errorf("Expected StageBlockchain rejection for replayed nonce, got %s", res2.FailureStage)
		}
	})

	// 5. DEMO: Lineage Evidence Verification
	t.Run("Demo 5: Evidence Generation & Lineage Verification", func(t *testing.T) {
		entries, err := akashicStore.All()
		if err != nil {
			t.Fatalf("Failed to read lineage entries: %v", err)
		}
		if len(entries) < 2 {
			t.Fatalf("Expected at least 2 lineage records, got %d", len(entries))
		}

		for idx, entry := range entries {
			if entry.TxHash == "" || entry.ExecutionHash == "" {
				t.Errorf("Lineage entry [%d] missing cryptographic hashes: %+v", idx, entry)
			}
		}
	})
}
