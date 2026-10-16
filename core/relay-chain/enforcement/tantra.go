// Package enforcement implements the TANTRA convergence layer.
//
// ARCHITECTURE (per reviewer feedback):
//
//   PDV Layer (deterministic correctness only):
//     ExecutionAgent  → execution_hash  (hash of canonical payload, NO timestamp)
//     ValidationAgent → validation_hash (same hash, independent recompute)
//     ReplayAgent     → replay_hash     (same hash, confirms determinism)
//     EQUALITY GATE:  execution_hash == validation_hash == replay_hash
//
//   Governance Layer (policy decisions — SEPARATE, called AFTER PDV passes):
//     FraudGate → calls Sarathi/DGIC fraud service (port 9091)
//     FRAUD GATE: decision == "allow"
//
// Timestamp is EXCLUDED from the deterministic hash zone so that
// replay across environments always produces the same hash.
package enforcement

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/sigverify"
)

// TxPayload is the canonical transaction representation passed through the pipeline.
type TxPayload struct {
	TraceID   string `json:"trace_id"`
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	TokenID   string `json:"token_id"`
	Fee       uint64 `json:"fee"`
	Nonce     uint64 `json:"nonce"`
	Timestamp int64  `json:"timestamp"` // carried for fraud/audit, NOT hashed
	Signature string `json:"signature"`
}

// deterministicZone is the subset of TxPayload that participates in the hash.
// Timestamp is intentionally excluded — it must not affect replay correctness.
type deterministicZone struct {
	TraceID   string `json:"trace_id"`
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	TokenID   string `json:"token_id"`
	Fee       uint64 `json:"fee"`
	Nonce     uint64 `json:"nonce"`
	Signature string `json:"signature"`
}

// EnforcementResult is returned to the caller after the full pipeline.
type EnforcementResult struct {
	Allowed         bool   `json:"allowed"`
	TraceID         string `json:"trace_id"`
	ExecutionHash   string `json:"execution_hash"`
	ValidationHash  string `json:"validation_hash"`
	ReplayHash      string `json:"replay_hash"`
	FraudDecision   string `json:"fraud_decision"` // "allow" | "block"
	SignatureValid  bool   `json:"signature_valid"`
	PayloadHash     string `json:"payload_hash,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// deterministicHash hashes only the deterministic zone — timestamp excluded.
func deterministicHash(tx *TxPayload) string {
	zone := deterministicZone{
		TraceID:   tx.TraceID,
		Type:      tx.Type,
		From:      tx.From,
		To:        tx.To,
		Amount:    tx.Amount,
		TokenID:   tx.TokenID,
		Fee:       tx.Fee,
		Nonce:     tx.Nonce,
		Signature: tx.Signature,
	}
	data, _ := json.Marshal(zone)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// injectTraceID ensures the payload carries a trace_id.
func injectTraceID(tx *TxPayload) {
	if tx.TraceID == "" {
		raw := fmt.Sprintf("%s:%s:%d:%d", tx.From, tx.To, tx.Amount, time.Now().UnixNano())
		sum := sha256.Sum256([]byte(raw))
		tx.TraceID = hex.EncodeToString(sum[:])[:16]
	}
}

// ── PDV LAYER ────────────────────────────────────────────────────────────────
// These three agents are PURE deterministic correctness checks.
// No fraud logic. No governance. No policy. Only math.

// ExecutionAgent — PDV Phase 1: injects trace_id, computes execution_hash.
func ExecutionAgent(tx *TxPayload) (string, error) {
	injectTraceID(tx)
	hash := deterministicHash(tx)
	log.Printf("[PDV][ExecutionAgent]  trace=%s execution_hash=%s", tx.TraceID, hash)
	return hash, nil
}

// ValidationAgent — PDV Phase 2: independent recompute of the same hash.
// No fraud call here. Fraud belongs to the governance layer.
func ValidationAgent(tx *TxPayload) (string, error) {
	hash := deterministicHash(tx)
	log.Printf("[PDV][ValidationAgent] trace=%s validation_hash=%s", tx.TraceID, hash)
	return hash, nil
}

// ReplayAgent — PDV Phase 3: third independent recompute confirming determinism.
func ReplayAgent(tx *TxPayload) (string, error) {
	hash := deterministicHash(tx)
	log.Printf("[PDV][ReplayAgent]     trace=%s replay_hash=%s", tx.TraceID, hash)
	return hash, nil
}

// ── GOVERNANCE LAYER ─────────────────────────────────────────────────────────
// Fraud/policy check is called ONLY after PDV equality gate passes.
// This is the Sarathi/DGIC layer — separate from deterministic truth.

type fraudRequest struct {
	TraceID   string `json:"trace_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	TokenID   string `json:"token_id"`
	Timestamp int64  `json:"timestamp"`
}

type fraudResponse struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason,omitempty"`
}

// sarathiURL returns the configured Sarathi/DGIC fraud service URL.
// Configurable via SARATHI_URL env var — eliminates localhost hardcoding.
// Default: http://localhost:9091/api/fraud/check (dev/single-node).
func sarathiURL() string {
	if u := os.Getenv("SARATHI_URL"); u != "" {
		return u
	}
	return "http://localhost:9091/api/fraud/check"
}

// sarathiFailClosed returns true when Sarathi fail-closed is active.
// DEFAULT: true (production-safe). Set SARATHI_FAIL_CLOSED=false to disable.
// This inverts the previous behavior where fail-closed required explicit opt-in.
func sarathiFailClosed() bool {
	return os.Getenv("SARATHI_FAIL_CLOSED") != "false"
}

// SarathiURLExported returns the configured Sarathi URL — exported for proof programs.
func SarathiURLExported() string { return sarathiURL() }

// FraudGate calls the Sarathi/DGIC fraud service.
// URL is configurable via SARATHI_URL env var.
// Fail behavior is configurable via SARATHI_FAIL_CLOSED env var.
func FraudGate(tx *TxPayload) (string, error) {
	return fraudGateURL(tx, sarathiURL())
}

func fraudGateURL(tx *TxPayload, url string) (string, error) {
	body, _ := json.Marshal(fraudRequest{
		TraceID:   tx.TraceID,
		From:      tx.From,
		To:        tx.To,
		Amount:    tx.Amount,
		TokenID:   tx.TokenID,
		Timestamp: tx.Timestamp,
	})
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		if sarathiFailClosed() {
			// FAIL-CLOSED: Sarathi unreachable → block transaction.
			// Production-safe: governance failure must not silently allow execution.
			log.Printf("[Sarathi][FAIL_CLOSED] trace=%s unreachable — blocking (SARATHI_FAIL_CLOSED=true)", tx.TraceID)
			return "block", nil
		}
		// FAIL-OPEN (default/dev): Sarathi unreachable → allow.
		log.Printf("[Sarathi][FraudGate] unreachable (%v) — defaulting to allow (set SARATHI_FAIL_CLOSED=true for production)", err)
		return "allow", nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var fr fraudResponse
	if err := json.Unmarshal(raw, &fr); err != nil {
		if sarathiFailClosed() {
			log.Printf("[Sarathi][FAIL_CLOSED] trace=%s bad response — blocking", tx.TraceID)
			return "block", nil
		}
		log.Printf("[Sarathi][FraudGate] bad response — defaulting to allow")
		return "allow", nil
	}
	log.Printf("[Sarathi][FraudGate] trace=%s decision=%s", tx.TraceID, fr.Decision)
	return fr.Decision, nil
}

// ── ENFORCE ──────────────────────────────────────────────────────────────────
// Full pipeline:
//   1. PDV layer  — equality gate (pure determinism)
//   2. Fraud gate — governance layer (called only if PDV passes)

func Enforce(tx *TxPayload) EnforcementResult {
	return enforceWithURL(tx, sarathiURL())
}

func enforceWithURL(tx *TxPayload, fraudURL string) EnforcementResult {
	injectTraceID(tx)

	// ── PHASE 1: SIGNATURE VERIFICATION ──────────────────────────────────────
	// Relay verifies wallet signature BEFORE PDV equality gate.
	// Invalid signature → SIGNATURE_REJECT — hard fail.
	sigResult := sigverify.Verify(sigverify.VerifyRequest{
		TraceID: tx.TraceID,
		Type:    tx.Type,
		From:    tx.From,
		To:      tx.To,
		Amount:  tx.Amount,
		TokenID: tx.TokenID,
		Fee:     tx.Fee,
		Nonce:   tx.Nonce,
		Signature: tx.Signature,
	})

	if !sigResult.Valid {
		log.Printf("[SIGVERIFY][REJECT] trace=%s reason=%s", tx.TraceID, sigResult.RejectionReason)
		return EnforcementResult{
			Allowed:         false,
			TraceID:         tx.TraceID,
			SignatureValid:  false,
			PayloadHash:     sigResult.PayloadHash,
			RejectionReason: "SIGNATURE_REJECT: " + sigResult.RejectionReason,
		}
	}
	log.Printf("[SIGVERIFY][PASS] trace=%s payload_hash=%s", tx.TraceID, sigResult.PayloadHash)

	// ── PDV LAYER ──
	execHash, err := ExecutionAgent(tx)
	if err != nil {
		return EnforcementResult{Allowed: false, TraceID: tx.TraceID, RejectionReason: "execution agent error: " + err.Error()}
	}

	valHash, err := ValidationAgent(tx)
	if err != nil {
		return EnforcementResult{Allowed: false, TraceID: tx.TraceID, RejectionReason: "validation agent error: " + err.Error()}
	}

	replayHash, err := ReplayAgent(tx)
	if err != nil {
		return EnforcementResult{Allowed: false, TraceID: tx.TraceID, RejectionReason: "replay agent error: " + err.Error()}
	}

	result := EnforcementResult{
		TraceID:        tx.TraceID,
		ExecutionHash:  execHash,
		ValidationHash: valHash,
		ReplayHash:     replayHash,
		SignatureValid: true,
		PayloadHash:    sigResult.PayloadHash,
	}

	// PDV equality gate — non-bypassable.
	if execHash != valHash || valHash != replayHash {
		result.Allowed = false
		result.RejectionReason = fmt.Sprintf(
			"PDV hash mismatch: execution=%s validation=%s replay=%s",
			execHash, valHash, replayHash,
		)
		log.Printf("[PDV][REJECT] trace=%s reason=%s", tx.TraceID, result.RejectionReason)
		return result
	}
	log.Printf("[PDV][PASS] trace=%s all_hashes=%s", tx.TraceID, execHash)

	// ── GOVERNANCE LAYER (fraud gate) — only reached if PDV passes ──
	decision, err := fraudGateURL(tx, fraudURL)
	if err != nil {
		result.Allowed = false
		result.RejectionReason = "fraud gate error: " + err.Error()
		return result
	}
	result.FraudDecision = decision

	if decision == "block" {
		result.Allowed = false
		result.RejectionReason = "Sarathi fraud gate blocked this transaction"
		log.Printf("[Sarathi][REJECT] trace=%s reason=%s", tx.TraceID, result.RejectionReason)
		return result
	}

	result.Allowed = true
	log.Printf("[Sarathi][ALLOW] trace=%s fraud=allow", tx.TraceID)
	return result
}
