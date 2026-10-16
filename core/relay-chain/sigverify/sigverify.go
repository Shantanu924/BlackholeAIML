// Package sigverify implements Phase 1 — Relay-Side Cryptographic Enforcement.
//
// The relay verifies wallet signatures BEFORE PDV PASS.
// Execution path:
//   Wallet Signature → Relay Verification → PDV Equality → Governance → Blockchain
//
// Design:
//   - Signature is btcec/ecdsa over the canonical payload hash (SHA-256)
//   - Canonical payload = deterministicZone JSON (timestamp excluded)
//   - Public key is the wallet's From address (compressed hex)
//   - Invalid signature → SIGNATURE_REJECT — hard fail before PDV
//   - Missing signature → SIGNATURE_MISSING — hard fail
//   - Signature is included in the PDV hash zone — replay-safe
//
// Replay safety:
//   The signature covers: trace_id + type + from + to + amount + token_id + fee + nonce
//   Timestamp is excluded — same as PDV deterministicZone.
//   This means the same transaction replayed with a different timestamp
//   produces the same signature verification result — deterministic.
package sigverify

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

// VerifyRequest is the input to signature verification.
type VerifyRequest struct {
	TraceID   string
	Type      string
	From      string // compressed public key hex — used as wallet address
	To        string
	Amount    uint64
	TokenID   string
	Fee       uint64
	Nonce     uint64
	Signature string // hex-encoded DER signature
}

// VerifyResult is returned by Verify.
type VerifyResult struct {
	Valid           bool   `json:"valid"`
	TraceID         string `json:"trace_id"`
	SignerAddress   string `json:"signer_address"`
	PayloadHash     string `json:"payload_hash"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// canonicalPayloadHash computes SHA-256 over the deterministic zone.
// Identical to the PDV deterministicZone — replay-safe.
func canonicalPayloadHash(req VerifyRequest) ([]byte, string) {
	type zone struct {
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
	// Signature field is empty when computing the hash to sign —
	// the wallet signs the payload WITHOUT the signature field,
	// then includes the signature in the contract.
	z := zone{
		TraceID: req.TraceID,
		Type:    req.Type,
		From:    req.From,
		To:      req.To,
		Amount:  req.Amount,
		TokenID: req.TokenID,
		Fee:     req.Fee,
		Nonce:   req.Nonce,
		// Signature intentionally omitted from hash input
	}
	data, _ := json.Marshal(z)
	sum := sha256.Sum256(data)
	return sum[:], hex.EncodeToString(sum[:])
}

// Verify verifies a btcec/ecdsa signature against the canonical payload hash.
// Returns VerifyResult with Valid=true if the signature is correct.
// Returns VerifyResult with Valid=false and RejectionReason if invalid.
func Verify(req VerifyRequest) VerifyResult {
	result := VerifyResult{
		TraceID:       req.TraceID,
		SignerAddress: req.From,
	}

	// Step 1 — signature must be present (unless dev mode allows unsigned).
	if req.Signature == "" {
		if os.Getenv("SIGVERIFY_STRICT_MODE") != "false" {
			result.Valid = false
			result.RejectionReason = "signature missing — wallet must sign canonical payload"
			log.Printf("[SIGVERIFY][REJECT] trace=%s reason=%s", req.TraceID, result.RejectionReason)
			return result
		}
		// Dev mode: allow unsigned transactions with a warning.
		log.Printf("[SIGVERIFY][WARN] trace=%s no signature — allowed (SIGVERIFY_STRICT_MODE=false, DEV ONLY)", req.TraceID)
		result.Valid = true
		result.PayloadHash = "unsigned-dev-mode"
		return result
	}

	// Step 2 — decode public key from From address (compressed hex).
	// Both hex-decode failure AND btcec parse failure mean the address is not a
	// valid compressed public key (e.g. named addresses like "alice", "bob").
	// Route both through the strict-mode gate — do NOT hard-reject before that.
	pubKeyBytes, hexErr := hex.DecodeString(req.From)
	var pubKey *btcec.PublicKey
	var parseErr error
	if hexErr == nil {
		pubKey, parseErr = btcec.ParsePubKey(pubKeyBytes)
	}

	if hexErr != nil || parseErr != nil {
		// From address is not a valid compressed public key.
		// DEFAULT: strict reject (production-safe).
		// Set SIGVERIFY_STRICT_MODE=false to allow named addresses (dev only).
		if os.Getenv("SIGVERIFY_STRICT_MODE") != "false" {
			result.Valid = false
			result.RejectionReason = fmt.Sprintf("SIGVERIFY_STRICT: from address is not a valid compressed public key (from=%s)", req.From)
			log.Printf("[SIGVERIFY][STRICT_REJECT] trace=%s from=%s reason=%s",
				req.TraceID, req.From, result.RejectionReason)
			return result
		}
		// Non-strict (dev only): named addresses allowed with warning.
		// Set SIGVERIFY_STRICT_MODE=false to enable this path.
		log.Printf("[SIGVERIFY][WARN] trace=%s from=%s not a valid pubkey — skipping sig check (SIGVERIFY_STRICT_MODE=false, DEV ONLY)",
			req.TraceID, req.From)
		result.Valid = true
		result.PayloadHash = "skipped-non-pubkey-address"
		return result
	}

	// Step 3 — compute canonical payload hash.
	hashBytes, hashHex := canonicalPayloadHash(req)
	result.PayloadHash = hashHex

	// Step 4 — decode DER signature.
	sigBytes, err := hex.DecodeString(req.Signature)
	if err != nil {
		result.Valid = false
		result.RejectionReason = fmt.Sprintf("invalid signature encoding (not hex): %v", err)
		log.Printf("[SIGVERIFY][REJECT] trace=%s reason=%s", req.TraceID, result.RejectionReason)
		return result
	}

	sig, err := ecdsa.ParseDERSignature(sigBytes)
	if err != nil {
		result.Valid = false
		result.RejectionReason = fmt.Sprintf("invalid DER signature: %v", err)
		log.Printf("[SIGVERIFY][REJECT] trace=%s reason=%s", req.TraceID, result.RejectionReason)
		return result
	}

	// Step 5 — verify signature against canonical hash.
	if !sig.Verify(hashBytes, pubKey) {
		result.Valid = false
		result.RejectionReason = "signature verification failed — payload hash mismatch"
		log.Printf("[SIGVERIFY][REJECT] trace=%s reason=%s payload_hash=%s",
			req.TraceID, result.RejectionReason, hashHex)
		return result
	}

	result.Valid = true
	log.Printf("[SIGVERIFY][PASS] trace=%s signer=%s payload_hash=%s",
		req.TraceID, req.From[:min(8, len(req.From))], hashHex[:8])
	return result
}

// Sign produces a btcec/ecdsa DER signature over the canonical payload hash.
// Called by the wallet before submitting — returns hex-encoded DER signature.
func Sign(req VerifyRequest, privKeyHex string) (string, error) {
	privKeyBytes, err := hex.DecodeString(privKeyHex)
	if err != nil {
		return "", fmt.Errorf("invalid private key hex: %v", err)
	}

	privKey, _ := btcec.PrivKeyFromBytes(privKeyBytes)
	hashBytes, _ := canonicalPayloadHash(req)

	sig := ecdsa.Sign(privKey, hashBytes)
	return hex.EncodeToString(sig.Serialize()), nil
}

// PayloadHash returns the canonical payload hash for a given request.
// Used by the wallet to know exactly what it is signing.
func PayloadHash(req VerifyRequest) string {
	_, hashHex := canonicalPayloadHash(req)
	return hashHex
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
