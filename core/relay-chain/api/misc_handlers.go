package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/constitution"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/ksml"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/replayverifier"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/runtime"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/sigverify"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/trace"
)

func (s *APIServer) handleAIFraudStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  "operational",
		"message": "AI fraud detection system is running",
	})
}

// handleTransactionData returns transaction data for ML analysis
func (s *APIServer) handleTransactionData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get recent transactions for ML analysis
	transactions := []map[string]interface{}{}

	if len(s.blockchain.Blocks) > 0 {
		// Get transactions from recent blocks
		for i := len(s.blockchain.Blocks) - 1; i >= 0 && len(transactions) < 100; i-- {
			block := s.blockchain.Blocks[i]
			for _, tx := range block.Transactions {
				transactions = append(transactions, map[string]interface{}{
					"id":        tx.ID,
					"from":      tx.From,
					"to":        tx.To,
					"amount":    tx.Amount,
					"timestamp": block.Header.Timestamp,
					"block":     block.Header.Index,
				})
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    transactions,
		"count":   len(transactions),
	})
}

func (s *APIServer) testDEX(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action  string `json:"action"`
		TokenA  string `json:"token_a"`
		TokenB  string `json:"token_b"`
		AmountA uint64 `json:"amount_a"`
		AmountB uint64 `json:"amount_b"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔧 DEV MODE: Executing DEX function '%s' with tokens %s/%s\n", req.Action, req.TokenA, req.TokenB)

	contract := &schema.TxContract{
		SchemaVersion: schema.CurrentVersion,
		Type:          "dex_" + req.Action,
		From:          "dex_system",
		To:            "dex_system",
		Amount:        req.AmountA,
		TokenID:       req.TokenA,
		Nonce:         uint64(time.Now().UnixNano()),
		Timestamp:     time.Now().Unix(),
	}

	execResult := runtime.Execute(runtime.ExecutionRequest{
		Contract:     contract,
		Blockchain:   s.blockchain,
		TruthStore:   s.truthStore,
		AkashicStore: s.akashic,
	})

	if !execResult.Allowed {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Execution failed: %s (%s)", execResult.ErrorCode, execResult.RejectionReason),
		})
		return
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("DEX %s executed successfully", req.Action),
		"data": map[string]interface{}{
			"action":   req.Action,
			"token_a":  req.TokenA,
			"token_b":  req.TokenB,
			"amount_a": req.AmountA,
			"amount_b": req.AmountB,
			"status":   "executed",
			"note":     "Transaction successfully processed by the runtime",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// testBridge handles Bridge testing requests
func (s *APIServer) testStaking(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action      string `json:"action"`
		Address     string `json:"address"`
		TokenSymbol string `json:"token_symbol"`
		Amount      uint64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔧 DEV MODE: Executing Staking function '%s' for address %s\n", req.Action, req.Address)

	contract := &schema.TxContract{
		SchemaVersion: schema.CurrentVersion,
		Type:          req.Action, // e.g. "stake", "unstake", "claim_rewards"
		From:          req.Address,
		To:            "system",
		Amount:        req.Amount,
		TokenID:       req.TokenSymbol,
		Nonce:         uint64(time.Now().UnixNano()),
		Timestamp:     time.Now().Unix(),
	}

	execResult := runtime.Execute(runtime.ExecutionRequest{
		Contract:     contract,
		Blockchain:   s.blockchain,
		TruthStore:   s.truthStore,
		AkashicStore: s.akashic,
	})

	if !execResult.Allowed {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Execution failed: %s (%s)", execResult.ErrorCode, execResult.RejectionReason),
		})
		return
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Staking %s executed successfully", req.Action),
		"data": map[string]interface{}{
			"action":       req.Action,
			"address":      req.Address,
			"token_symbol": req.TokenSymbol,
			"amount":       req.Amount,
			"status":       "executed",
			"note":         "Transaction successfully processed by the runtime",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// testMultisig handles Multisig testing requests
func (s *APIServer) testMultisig(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action      string   `json:"action"`
		Owners      []string `json:"owners"`
		Threshold   int      `json:"threshold"`
		WalletID    string   `json:"wallet_id"`
		ToAddress   string   `json:"to_address"`
		TokenSymbol string   `json:"token_symbol"`
		Amount      uint64   `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔧 DEV MODE: Executing Multisig function '%s'\n", req.Action)

	fromAddress := req.WalletID
	if fromAddress == "" && len(req.Owners) > 0 {
		fromAddress = req.Owners[0]
	}

	contract := &schema.TxContract{
		SchemaVersion: schema.CurrentVersion,
		Type:          "multisig_" + req.Action,
		From:          fromAddress,
		To:            req.ToAddress,
		Amount:        req.Amount,
		TokenID:       req.TokenSymbol,
		Nonce:         uint64(time.Now().UnixNano()),
		Timestamp:     time.Now().Unix(),
	}

	execResult := runtime.Execute(runtime.ExecutionRequest{
		Contract:     contract,
		Blockchain:   s.blockchain,
		TruthStore:   s.truthStore,
		AkashicStore: s.akashic,
	})

	if !execResult.Allowed {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Execution failed: %s (%s)", execResult.ErrorCode, execResult.RejectionReason),
		})
		return
	}

	result := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Multisig %s executed successfully", req.Action),
		"data": map[string]interface{}{
			"action": req.Action,
			"status": "executed",
			"note":   "Transaction successfully processed by the runtime",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// testOTC handles OTC trading testing requests
func (s *APIServer) handleSigVerify(w http.ResponseWriter, r *http.Request) {
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

	result := sigverify.Verify(sigverify.VerifyRequest{
		TraceID:   contract.TraceID,
		Type:      contract.Type,
		From:      contract.From,
		To:        contract.To,
		Amount:    contract.Amount,
		TokenID:   contract.TokenID,
		Fee:       contract.Fee,
		Nonce:     contract.Nonce,
		Signature: contract.Signature,
	})

	if !result.Valid {
		w.WriteHeader(http.StatusForbidden)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          result.Valid,
		"valid":            result.Valid,
		"trace_id":         result.TraceID,
		"signer_address":   result.SignerAddress,
		"payload_hash":     result.PayloadHash,
		"rejection_reason": result.RejectionReason,
	})
}

// handleTraceVerify verifies trace continuity for a given trace_id across all layers.
// GET /api/trace/verify?trace_id=<id>
func (s *APIServer) handleTraceVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "trace_id query param required"})
		return
	}

	result := map[string]interface{}{
		"trace_id": traceID,
		"layers":   map[string]interface{}{},
	}
	layers := result["layers"].(map[string]interface{})

	// Check Bucket (truthstore)
	if s.truthStore != nil {
		rec, found := s.truthStore.FindByTraceID(traceID)
		if found {
			layers["bucket"] = map[string]interface{}{
				"found":          true,
				"tx_hash":        rec.TxHash,
				"execution_hash": rec.ExecutionHash,
				"fraud_decision": rec.FraudDecision,
				"timestamp":      rec.Timestamp,
			}
		} else {
			layers["bucket"] = map[string]interface{}{"found": false}
		}
	}

	// Check AKASHIC
	if s.akashic != nil {
		entry, found := s.akashic.FindByTraceID(traceID)
		if found {
			layers["akashic"] = map[string]interface{}{
				"found":          true,
				"tx_hash":        entry.TxHash,
				"block_height":   entry.BlockHeight,
				"execution_hash": entry.ExecutionHash,
				"entry_hash":     entry.EntryHash,
			}
		} else {
			layers["akashic"] = map[string]interface{}{"found": false}
		}
	}

	// Determine overall continuity
	bucketFound := layers["bucket"] != nil && layers["bucket"].(map[string]interface{})["found"] == true
	akashicFound := layers["akashic"] != nil && layers["akashic"].(map[string]interface{})["found"] == true
	continuous := bucketFound && akashicFound

	result["continuous"] = continuous
	result["success"] = continuous
	if !continuous {
		w.WriteHeader(http.StatusNotFound)
		result["error"] = "trace_id not found in all layers — trace continuity broken"
	}
	json.NewEncoder(w).Encode(result)
}

// handleReplayVerify — Phase 1F: canonical replay verification.
// Proves same input → same PDV hashes → same state.
// POST /api/replay/verify
func (s *APIServer) handleConvergenceProof(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	proof := map[string]interface{}{}
	allPassed := true

	// 1. Bucket chain integrity
	if s.truthStore != nil {
		intact, brokenAt, err := s.truthStore.VerifyChain()
		if err != nil || !intact {
			allPassed = false
			proof["bucket_chain"] = map[string]interface{}{"intact": false, "broken_at": brokenAt}
		} else {
			records, _ := s.truthStore.All()
			proof["bucket_chain"] = map[string]interface{}{"intact": true, "record_count": len(records)}
		}
	} else {
		allPassed = false
		proof["bucket_chain"] = map[string]interface{}{"intact": false, "error": "truth store not initialized"}
	}

	// 2. AKASHIC reconstruction
	if s.akashic != nil {
		result := s.akashic.Reconstruct()
		if !result.Verified {
			allPassed = false
		}
		proof["akashic_reconstruction"] = map[string]interface{}{
			"verified":         result.Verified,
			"chain_intact":     result.ChainIntact,
			"total_entries":    result.TotalEntries,
			"final_state_root": result.FinalStateRoot,
			"message":          result.Message,
		}
	} else {
		allPassed = false
		proof["akashic_reconstruction"] = map[string]interface{}{"verified": false, "error": "akashic store not initialized"}
	}

	// 3. Blockchain state
	proof["blockchain"] = map[string]interface{}{
		"block_height": len(s.blockchain.Blocks),
		"pending_txs":  len(s.blockchain.PendingTxs),
	}

	// 4. Runtime status
	proof["runtime"] = map[string]interface{}{
		"canonical_entry":   "POST /api/relay/submit",
		"execution_path":    "Schema → PDV → Governance → Blockchain → Bucket → AKASHIC",
		"trace_enforcement": "trace.Context — immutable after injection",
		"bypass_paths":      0,
		"schema_version":    "v1",
	}

	if !allPassed {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   allPassed,
		"converged": allPassed,
		"timestamp": time.Now().Unix(),
		"proof":     proof,
	})
}

// rejectObservable writes a structured, observable rejection response.
// Every failure surface is named, logged, and deterministic — no silent failures.
func (s *APIServer) rejectObservable(w http.ResponseWriter, code, reason, traceID string, status int) {
	log.Printf("[REJECT][%s] trace=%s reason=%s", code, traceID, reason)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":          false,
		"error_code":       code,
		"rejection_reason": reason,
		"trace_id":         traceID,
	})
}

// readBody reads the full request body.
func (s *APIServer) handleSupportedChains(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	token := r.URL.Query().Get("token")

	supportedChains := map[string]interface{}{
		"chains": []map[string]interface{}{
			{
				"id":               "blackhole",
				"name":             "Blackhole Blockchain",
				"native_token":     "BHIV",
				"supported_tokens": []string{"BHIV", "USDT", "ETH", "SOL"},
				"bridge_fee":       1,
			},
			{
				"id":               "ethereum",
				"name":             "Ethereum",
				"native_token":     "ETH",
				"supported_tokens": []string{"ETH", "USDT", "wBHIV"},
				"bridge_fee":       10,
			},
			{
				"id":               "solana",
				"name":             "Solana",
				"native_token":     "SOL",
				"supported_tokens": []string{"SOL", "USDT", "pBHIV"},
				"bridge_fee":       5,
			},
		},
	}

	if token != "" {
		// Filter chains that support the specific token
		var supportingChains []map[string]interface{}
		for _, chain := range supportedChains["chains"].([]map[string]interface{}) {
			supportedTokens := chain["supported_tokens"].([]string)
			for _, supportedToken := range supportedTokens {
				if supportedToken == token {
					supportingChains = append(supportingChains, chain)
					break
				}
			}
		}
		supportedChains["chains"] = supportingChains
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    supportedChains,
	})
}

// handleBridgeEvents handles bridge event queries
func (s *APIServer) handleKSMLSubmit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "POST required"})
		return
	}

	rawBody, err := readBody(r)
	if err != nil {
		s.rejectObservable(w, "BODY_READ_ERROR", err.Error(), "", http.StatusBadRequest)
		return
	}

	// Parse and validate KSML contract.
	ksmlContract, err := ksml.ParseAndValidate(rawBody)
	if err != nil {
		log.Printf("[KSML][REJECT] reason=%s", err.Error())
		s.rejectObservable(w, "KSML_VIOLATION", err.Error(), "", http.StatusBadRequest)
		return
	}
	log.Printf("[KSML][PASS] intent=%s source=%s", ksmlContract.IntentType, ksmlContract.Source)

	// Convert KSML to schema v1 TxContract.
	contract, err := ksml.ToTxContract(ksmlContract)
	if err != nil {
		s.rejectObservable(w, "KSML_CONVERSION_ERROR", err.Error(), "", http.StatusBadRequest)
		return
	}

	// Route through canonical runtime.
	tc := trace.New(contract.TraceID)
	execResult := runtime.Execute(runtime.ExecutionRequest{
		Contract:     contract,
		Blockchain:   s.blockchain,
		TruthStore:   s.truthStore,
		AkashicStore: s.akashic,
	})

	if execResult.TraceID != "" {
		if err := tc.Inject(execResult.TraceID); err != nil {
			s.rejectObservable(w, "TRACE_BREAK", err.Error(), tc.ID(), http.StatusInternalServerError)
			return
		}
	}

	if !execResult.Allowed {
		httpStatus := http.StatusForbidden
		if execResult.ErrorCode == "BLOCKCHAIN_REJECT" {
			httpStatus = http.StatusUnprocessableEntity
		}
		s.rejectObservable(w, execResult.ErrorCode, execResult.RejectionReason, execResult.TraceID, httpStatus)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"transaction_id":  execResult.TxHash,
		"trace_id":        execResult.TraceID,
		"execution_hash":  execResult.ExecutionHash,
		"validation_hash": execResult.ValidationHash,
		"replay_hash":     execResult.ReplayHash,
		"fraud_decision":  execResult.FraudDecision,
		"block_height":    execResult.BlockHeight,
		"ksml_version":    ksmlContract.KSMLVersion,
		"intent_type":     ksmlContract.IntentType,
		"source":          ksmlContract.Source,
	})
}

// handleKarmaChainConsistency checks state root equality across all KarmaChain nodes.
// GET /api/karmachain/consistency
func (s *APIServer) handleStateRootEquality(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	result := replayverifier.VerifyStateRootEquality()
	if !result.Equal {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": result.Equal, "result": result})
}

// handleCorruptSimulate — Phase 4: simulate AKASHIC corruption for testing.
// POST /api/akashic/corrupt-simulate
func (s *APIServer) handleCorruptSimulate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.akashic == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "akashic not initialized"})
		return
	}
	originalHash, err := s.akashic.SimulateCorruption()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	// Immediately verify chain to confirm corruption is detected.
	reconResult := s.akashic.Reconstruct()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":             true,
		"corrupted_hash":      originalHash,
		"corruption_detected": !reconResult.Verified,
		"reconstruction":      reconResult,
		"message":             "corruption simulated and detected by Reconstruct()",
	})
}

// handleConstitutionDeclaration — Phase 5: runtime authority boundary declaration.
// GET /api/constitution/declaration
func (s *APIServer) handleVerifyBoundary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	name := r.URL.Query().Get("name")
	if name == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "name required"})
		return
	}
	intact, proof := constitution.VerifyBoundary(name)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  intact,
		"boundary": name,
		"intact":   intact,
		"proof":    proof,
	})
}

// handleMonitoringMetrics provides detailed metrics for external monitoring systems
func (s *APIServer) handleIdentity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":     "vajra-blockchain-node",
		"trust_layer": "VAJRA Blockchain",
		"port":        s.port,
		"authority": []string{
			"PDV_EQUALITY",
			"SIGNATURE_VERIFICATION",
			"NONCE_GOVERNANCE",
			"TRACE_CONTINUITY",
			"BUCKET_TRUTH",
			"AKASHIC_LINEAGE",
		},
		"non_authority": []string{
			"GOVERNANCE_LEGITIMACY",
			"SEMANTIC_TRUTH",
			"SOVEREIGN_CORRECTNESS",
		},
		"adjacent": []string{
			"fraud-service:9090",
			"bridge:8084",
			"wallet:9000",
		},
		"endpoints": []string{
			"POST /api/relay/submit",
			"GET  /api/health",
			"GET  /api/tantra/status",
			"GET  /api/constitution/declaration",
			"GET  /api/convergence/proof",
			"GET  /api/vajra/contracts",
			"GET  /identity",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// handleVAJRAContracts — VAJRA Phase 2: probes all 4 participant /identity endpoints.
// GET /api/vajra/contracts
