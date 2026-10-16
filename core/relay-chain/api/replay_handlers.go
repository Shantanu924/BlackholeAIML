package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/enforcement"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/replayverifier"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
)

func (s *APIServer) handleReplayVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "POST required"})
		return
	}

	rawBody, err := readBody(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	contract, err := schema.ParseAndValidate(rawBody)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}

	// Run PDV three times on the same payload — all hashes must be identical.
	payload := &enforcement.TxPayload{
		TraceID:   contract.TraceID,
		Type:      contract.Type,
		From:      contract.From,
		To:        contract.To,
		Amount:    contract.Amount,
		TokenID:   contract.TokenID,
		Fee:       contract.Fee,
		Nonce:     contract.Nonce,
		Timestamp: contract.Timestamp,
		Signature: contract.Signature,
	}

	h1, _ := enforcement.ExecutionAgent(payload)
	h2, _ := enforcement.ValidationAgent(payload)
	h3, _ := enforcement.ReplayAgent(payload)

	deterministic := h1 == h2 && h2 == h3
	log.Printf("[REPLAY][VERIFY] trace=%s h1=%s h2=%s h3=%s deterministic=%v",
		payload.TraceID, h1, h2, h3, deterministic)

	if !deterministic {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       deterministic,
		"deterministic": deterministic,
		"trace_id":      payload.TraceID,
		"run_1_hash":    h1,
		"run_2_hash":    h2,
		"run_3_hash":    h3,
		"message":       map[bool]string{true: "replay determinism confirmed", false: "REPLAY MISMATCH — non-deterministic execution detected"}[deterministic],
	})
}

// handleConvergenceProof — Phase 1G: full live convergence proof.
// GET /api/convergence/proof
func (s *APIServer) handleReplayEquality(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "trace_id required"})
		return
	}
	result := replayverifier.VerifyEquality(traceID)
	if !result.Equal {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": result.Equal, "result": result})
}

// handleStateRootEquality — Phase 3: state root equality across all nodes.
// GET /api/replay/state-root
