package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/noncecoord"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/runtime"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/trace"
)

func (s *APIServer) handleRelaySubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.rejectObservable(w, "METHOD_NOT_ALLOWED", "only POST allowed", "", http.StatusMethodNotAllowed)
		return
	}

	// ── PHASE 3 / 1D: SCHEMA CONTRACT VALIDATION ────────────────────────────
	rawBody, err := readBody(r)
	if err != nil {
		s.rejectObservable(w, "BODY_READ_ERROR", err.Error(), "", http.StatusBadRequest)
		return
	}

	contract, err := schema.ParseAndValidate(rawBody)
	if err != nil {
		log.Printf("[SCHEMA][REJECT] reason=%s", err.Error())
		s.rejectObservable(w, "SCHEMA_VIOLATION", err.Error(), "", http.StatusBadRequest)
		return
	}

	// ── PHASE 1C: TRACE CONTEXT INITIALISATION ──────────────────────────────
	// Create immutable trace context. trace_id from client preserved if present.
	// After this point, trace_id NEVER changes — any drift = HARD FAIL.
	tc := trace.New(contract.TraceID)
	log.Printf("[SCHEMA][PASS] type=%s from=%s to=%s amount=%d token=%s trace=%s",
		contract.Type, contract.From, contract.To, contract.Amount, contract.TokenID, tc.ID())

	// ── SCOPE 2: CROSS-NODE NONCE COORDINATION ─────────────────────────────
	// If NONCE_COORDINATOR_URL is set, check with the global coordinator FIRST.
	// This prevents the same nonce being accepted on two different relay nodes.
	// If coordinator is unreachable, fail-closed: reject the nonce.
	// If coordinator is not configured, skip (single-node mode).
	nodeID := fmt.Sprintf("relay-%d", s.port)
	if err := noncecoord.CheckWithCoordinator(contract.From, contract.Nonce, contract.TraceID, nodeID); err != nil {
		log.Printf("[NONCECOORD][REJECT] from=%s nonce=%d reason=%s", contract.From, contract.Nonce, err.Error())
		s.rejectObservable(w, "NONCE_REPLAY", err.Error(), contract.TraceID, http.StatusConflict)
		return
	}

	// ── PHASE 2: PERSISTENT NONCE GOVERNANCE ──────────────────────────────
	// Check nonce against persistent store — survives restart.
	// Duplicate nonce after restart → NONCE_REPLAY hard fail.
	if s.nonceStore != nil {
		if err := s.nonceStore.CheckAndAccept(contract.From, contract.Nonce, contract.TraceID); err != nil {
			log.Printf("[NONCESTORE][REJECT] from=%s nonce=%d reason=%s",
				contract.From, contract.Nonce, err.Error())
			s.rejectObservable(w, "NONCE_REPLAY", err.Error(), contract.TraceID, http.StatusConflict)
			return
		}
	}

	// ── PHASE 1D: CANONICAL RUNTIME EXECUTION ───────────────────────────────
	// ALL execution flows through runtime.Execute() — no direct enforcement calls.
	execResult := runtime.Execute(runtime.ExecutionRequest{
		Contract:     contract,
		Blockchain:   s.blockchain,
		TruthStore:   s.truthStore,
		AkashicStore: s.akashic,
	})

	// ── PHASE 1C: TRACE CONTINUITY ASSERTION ────────────────────────────────
	// After runtime returns, assert trace_id has not drifted.
	if execResult.TraceID != "" {
		if err := tc.Inject(execResult.TraceID); err != nil {
			s.rejectObservable(w, "TRACE_BREAK", err.Error(), tc.ID(), http.StatusInternalServerError)
			return
		}
	}
	tc.LogStage("RUNTIME_COMPLETE", fmt.Sprintf("allowed=%v tx=%s", execResult.Allowed, execResult.TxHash))

	if !execResult.Allowed {
		httpStatus := http.StatusForbidden
		if execResult.ErrorCode == "BLOCKCHAIN_REJECT" {
			httpStatus = http.StatusUnprocessableEntity
		}
		s.rejectObservable(w, execResult.ErrorCode, execResult.RejectionReason, execResult.TraceID, httpStatus)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"transaction_id":  execResult.TxHash,
		"status":          "pending",
		"submitted_at":    execResult.SubmittedAt,
		"trace_id":        execResult.TraceID,
		"execution_hash":  execResult.ExecutionHash,
		"validation_hash": execResult.ValidationHash,
		"replay_hash":     execResult.ReplayHash,
		"fraud_decision":  execResult.FraudDecision,
		"signature_valid": execResult.SignatureValid,
		"payload_hash":    execResult.PayloadHash,
		"block_height":    execResult.BlockHeight,
		"schema_version":  execResult.SchemaVersion,
	})
}

// handleNonceLookup returns the latest nonce for an address.
// GET /api/nonce/lookup?address=<addr>
func (s *APIServer) handleRelayStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	latestBlock := s.blockchain.GetLatestBlock()
	pendingTxs := s.blockchain.GetPendingTransactions()

	status := map[string]interface{}{
		"chain_id":             "blackhole-mainnet",
		"block_height":         latestBlock.Header.Index,
		"latest_block_hash":    latestBlock.Hash,
		"latest_block_time":    latestBlock.Header.Timestamp,
		"pending_transactions": len(pendingTxs),
		"relay_active":         true,
		"timestamp":            time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

// handleRelayEvents handles relay event streaming
func (s *APIServer) handleRelayEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Simple event list (in production, this would be a real-time stream)
	events := []map[string]interface{}{
		{
			"id":           "relay_event_1",
			"type":         "block_created",
			"block_height": s.blockchain.GetLatestBlock().Header.Index,
			"timestamp":    time.Now().Unix(),
			"data": map[string]interface{}{
				"validator":  "node1",
				"tx_count":   5,
				"block_size": 2048,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    events,
	})
}

// handleRelayValidate handles transaction validation
func (s *APIServer) handleRelayValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type    string `json:"type"`
		From    string `json:"from"`
		To      string `json:"to"`
		Amount  uint64 `json:"amount"`
		TokenID string `json:"token_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Basic validation
	warnings := []string{}
	valid := true

	if req.From == "" || req.To == "" {
		valid = false
		warnings = append(warnings, "from and to addresses are required")
	}

	if req.Amount == 0 {
		valid = false
		warnings = append(warnings, "amount must be greater than 0")
	}

	// Check token exists
	if req.TokenID != "" {
		if _, exists := s.blockchain.TokenRegistry[req.TokenID]; !exists {
			valid = false
			warnings = append(warnings, fmt.Sprintf("token %s not found", req.TokenID))
		}
	}

	validation := map[string]interface{}{
		"valid":               valid,
		"warnings":            warnings,
		"estimated_fee":       uint64(1000),
		"estimated_gas":       uint64(21000),
		"success_probability": 0.95,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    validation,
	})
}

// processCrossChainSwap simulates the cross-chain swap process
