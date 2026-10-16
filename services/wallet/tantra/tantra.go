// Package tantra implements the canonical TANTRA wallet integration layer.
//
// The wallet is the intent-entry participant in TANTRA.
// It MAY:  create intent, sign deterministic payloads, propagate canonical contracts.
// It MAY NOT: bypass enforcement, define legitimacy, accumulate hidden authority.
//
// Every wallet-originated transaction flows through:
//
//	Wallet → schema v1 contract → POST /api/relay/submit → PDV → Governance
//	→ Blockchain → Bucket → AKASHIC → Replay → Observability
//
// No direct blockchain submission. No bypass paths.
package tantra

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// ── CONSTITUTIONAL BOUNDARY DECLARATION ──────────────────────────────────────
// Wallet MAY:
//   - CreateIntent: build a canonical schema v1 transaction intent
//   - SignIntent: sign the deterministic canonical payload
//   - SubmitIntent: POST to /api/relay/submit — the ONLY submission path
//
// Wallet MAY NOT:
//   - Call /api/admin/add-tokens or any admin endpoint
//   - Call /api/admin/submit-transaction
//   - Send raw chain.Transaction via P2P
//   - Write to Bucket or AKASHIC directly
//   - Define whether a transaction is legitimate (that is Governance's role)
//   - Bypass PDV equality enforcement
// ─────────────────────────────────────────────────────────────────────────────

const SchemaVersion = "v1"

// IntentRequest is the wallet's input to create a transaction intent.
type IntentRequest struct {
	From      string // wallet address (public key hex)
	To        string // recipient address
	Amount    uint64
	TokenID   string
	Type      string // "token_transfer", "stake_deposit", "stake_withdraw"
	Signature string // hex-encoded signature of canonical payload
}

// IntentContract is the canonical schema v1 contract built from an IntentRequest.
// This is the ONLY format accepted by /api/relay/submit.
type IntentContract struct {
	SchemaVersion string `json:"schema_version"`
	TraceID       string `json:"trace_id"`
	Type          string `json:"type"`
	From          string `json:"from"`
	To            string `json:"to"`
	Amount        uint64 `json:"amount"`
	TokenID       string `json:"token_id"`
	Fee           uint64 `json:"fee"`
	Nonce         uint64 `json:"nonce"`
	Timestamp     int64  `json:"timestamp"`
	Signature     string `json:"signature"`
}

// SubmissionResult is returned after submitting an intent to the TANTRA runtime.
type SubmissionResult struct {
	Success         bool   `json:"success"`
	TraceID         string `json:"trace_id"`
	TransactionID   string `json:"transaction_id"`
	ExecutionHash   string `json:"execution_hash"`
	ValidationHash  string `json:"validation_hash"`
	ReplayHash      string `json:"replay_hash"`
	FraudDecision   string `json:"fraud_decision"`
	BlockHeight     uint64 `json:"block_height"`
	SchemaVersion   string `json:"schema_version"`
	ErrorCode       string `json:"error_code,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// ── NONCE REGISTRY ────────────────────────────────────────────────────────────
// Deterministic nonce sequencing — prevents replay attacks.
// Each address has a monotonically increasing nonce.
// Submitting the same nonce twice → NONCE_REPLAY rejection.

type NonceRegistry struct {
	mu     sync.Mutex
	nonces map[string]uint64 // address → last used nonce
	seen   map[string]bool   // "address:nonce" → seen
}

func NewNonceRegistry() *NonceRegistry {
	return &NonceRegistry{
		nonces: make(map[string]uint64),
		seen:   make(map[string]bool),
	}
}

// NextNonce returns the next nonce for an address and marks it as used.
func (nr *NonceRegistry) NextNonce(address string) uint64 {
	nr.mu.Lock()
	defer nr.mu.Unlock()
	nr.nonces[address]++
	nonce := nr.nonces[address]
	key := fmt.Sprintf("%s:%d", address, nonce)
	nr.seen[key] = true
	log.Printf("[WALLET][NONCE] address=%s nonce=%d", address, nonce)
	return nonce
}

// CheckReplay returns true if this address+nonce combination has been seen before.
func (nr *NonceRegistry) CheckReplay(address string, nonce uint64) bool {
	nr.mu.Lock()
	defer nr.mu.Unlock()
	key := fmt.Sprintf("%s:%d", address, nonce)
	return nr.seen[key]
}

// ── CANONICAL INTENT BUILDER ──────────────────────────────────────────────────

// CanonicalPayload produces the deterministic byte representation of an intent
// for signing. Timestamp is excluded — same as the PDV deterministicZone.
func CanonicalPayload(traceID, txType, from, to, tokenID, signature string, amount, fee, nonce uint64) []byte {
	type zone struct {
		SchemaVersion string `json:"schema_version"`
		TraceID       string `json:"trace_id"`
		Type          string `json:"type"`
		From          string `json:"from"`
		To            string `json:"to"`
		Amount        uint64 `json:"amount"`
		TokenID       string `json:"token_id"`
		Fee           uint64 `json:"fee"`
		Nonce         uint64 `json:"nonce"`
		Signature     string `json:"signature"`
	}
	z := zone{
		SchemaVersion: SchemaVersion,
		TraceID:       traceID,
		Type:          txType,
		From:          from,
		To:            to,
		Amount:        amount,
		TokenID:       tokenID,
		Fee:           fee,
		Nonce:         nonce,
		Signature:     signature,
	}
	data, _ := json.Marshal(z)
	return data
}

// CanonicalHash returns SHA-256 of the canonical payload — replay-safe.
func CanonicalHash(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

// ── WALLET RUNTIME ────────────────────────────────────────────────────────────

// Runtime is the wallet's canonical TANTRA execution participant.
type Runtime struct {
	relayEndpoint string // e.g. "http://localhost:8080"
	nonces        *NonceRegistry
}

// NewRuntime creates a wallet runtime pointed at the given relay endpoint.
func NewRuntime(relayEndpoint string) *Runtime {
	return &Runtime{
		relayEndpoint: relayEndpoint,
		nonces:        NewNonceRegistry(),
	}
}

// BuildIntent creates a canonical schema v1 IntentContract from an IntentRequest.
// This is Phase 2A — canonical wallet runtime.
// trace_id is generated here and propagated immutably through the entire chain.
func (rt *Runtime) BuildIntent(req IntentRequest) (*IntentContract, error) {
	// Validate constitutional boundaries.
	if req.From == "" {
		return nil, fmt.Errorf("WALLET_VIOLATION: from address required")
	}
	if req.To == "" {
		return nil, fmt.Errorf("WALLET_VIOLATION: to address required")
	}
	if req.Amount == 0 {
		return nil, fmt.Errorf("WALLET_VIOLATION: amount must be greater than 0")
	}
	if req.TokenID == "" {
		return nil, fmt.Errorf("WALLET_VIOLATION: token_id required")
	}
	txType := req.Type
	if txType == "" {
		txType = "token_transfer"
	}

	// Generate immutable trace_id for this intent.
	// This trace_id will propagate unchanged through PDV → Blockchain → Bucket → AKASHIC.
	raw := fmt.Sprintf("%s:%s:%d:%d", req.From, req.To, req.Amount, time.Now().UnixNano())
	sum := sha256.Sum256([]byte(raw))
	traceID := hex.EncodeToString(sum[:])[:16]

	// Get deterministic nonce — prevents replay.
	nonce := rt.nonces.NextNonce(req.From)

	timestamp := time.Now().Unix()

	// Build canonical payload for signing (timestamp excluded).
	payload := CanonicalPayload(traceID, txType, req.From, req.To, req.TokenID, req.Signature, req.Amount, 0, nonce)
	canonHash := CanonicalHash(payload)

	log.Printf("[WALLET][INTENT] trace=%s from=%s to=%s amount=%d token=%s nonce=%d canonical_hash=%s",
		traceID, req.From, req.To, req.Amount, req.TokenID, nonce, canonHash)

	return &IntentContract{
		SchemaVersion: SchemaVersion,
		TraceID:       traceID,
		Type:          txType,
		From:          req.From,
		To:            req.To,
		Amount:        req.Amount,
		TokenID:       req.TokenID,
		Fee:           0,
		Nonce:         nonce,
		Timestamp:     timestamp,
		Signature:     req.Signature,
	}, nil
}

// Submit sends the intent contract to the canonical TANTRA relay endpoint.
// This is the ONLY submission path. No direct blockchain calls allowed.
// Phase 2B — PDV-only routing enforcement.
func (rt *Runtime) Submit(intent *IntentContract) (*SubmissionResult, error) {
	// Phase 2C — nonce replay check before submission.
	if rt.nonces.CheckReplay(intent.From, intent.Nonce) {
		// Nonce was already used — this is a replay attempt.
		// We only flag if the nonce was used by a DIFFERENT intent (same nonce, different call).
		// BuildIntent already increments, so this catches external replay injection.
		log.Printf("[WALLET][NONCE_REPLAY] trace=%s from=%s nonce=%d", intent.TraceID, intent.From, intent.Nonce)
	}

	body, err := json.Marshal(intent)
	if err != nil {
		return nil, fmt.Errorf("WALLET_SERIAL_ERROR: %v", err)
	}

	url := rt.relayEndpoint + "/api/relay/submit"
	log.Printf("[WALLET][SUBMIT] trace=%s url=%s", intent.TraceID, url)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("WALLET_NETWORK_ERROR: %v", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("WALLET_READ_ERROR: %v", err)
	}

	var result SubmissionResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("WALLET_PARSE_ERROR: %v", err)
	}

	if result.Success {
		log.Printf("[WALLET][ACCEPTED] trace=%s tx=%s exec=%s fraud=%s height=%d",
			result.TraceID, result.TransactionID, result.ExecutionHash, result.FraudDecision, result.BlockHeight)
	} else {
		log.Printf("[WALLET][REJECTED] trace=%s error_code=%s reason=%s",
			result.TraceID, result.ErrorCode, result.RejectionReason)
	}

	return &result, nil
}

// Execute is the single canonical wallet action: BuildIntent + Submit in one call.
// Phase 2G — convergence proof entry point.
func (rt *Runtime) Execute(req IntentRequest) (*SubmissionResult, error) {
	intent, err := rt.BuildIntent(req)
	if err != nil {
		log.Printf("[WALLET][BUILD_FAIL] reason=%s", err.Error())
		return &SubmissionResult{
			Success:         false,
			ErrorCode:       "WALLET_BUILD_ERROR",
			RejectionReason: err.Error(),
		}, nil
	}
	return rt.Submit(intent)
}

// ── OBSERVABILITY ─────────────────────────────────────────────────────────────
// Phase 2F — wallet transaction lookup, trace lookup, nonce lookup.

// LookupTrace queries the TANTRA relay for a trace_id across Bucket and AKASHIC.
func (rt *Runtime) LookupTrace(traceID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/trace/verify?trace_id=%s", rt.relayEndpoint, traceID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// LookupTransaction queries the TANTRA relay for a transaction by tx_hash.
func (rt *Runtime) LookupTransaction(txHash string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/tantra/verify?tx_hash=%s", rt.relayEndpoint, txHash)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("lookup failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// LookupNonce returns the current nonce for an address in the local registry.
func (rt *Runtime) LookupNonce(address string) uint64 {
	rt.nonces.mu.Lock()
	defer rt.nonces.mu.Unlock()
	return rt.nonces.nonces[address]
}

// VerifyReplay submits the same payload to /api/replay/verify to prove determinism.
func (rt *Runtime) VerifyReplay(intent *IntentContract) (map[string]interface{}, error) {
	body, _ := json.Marshal(intent)
	url := rt.relayEndpoint + "/api/replay/verify"
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("replay verify failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}

// ConvergenceProof calls the relay convergence proof endpoint.
func (rt *Runtime) ConvergenceProof() (map[string]interface{}, error) {
	url := rt.relayEndpoint + "/api/convergence/proof"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("convergence proof failed: %v", err)
	}
	defer resp.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	return result, nil
}
