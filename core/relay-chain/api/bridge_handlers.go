package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/runtime"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/bridge"
)

func (s *APIServer) testBridge(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action        string `json:"action"`
		SourceChain   string `json:"source_chain"`
		DestChain     string `json:"dest_chain"`
		SourceAddress string `json:"source_address"`
		DestAddress   string `json:"dest_address"`
		TokenSymbol   string `json:"token_symbol"`
		Amount        uint64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔧 DEV MODE: Executing Bridge function '%s' from %s to %s\n", req.Action, req.SourceChain, req.DestChain)

	contract := &schema.TxContract{
		SchemaVersion: schema.CurrentVersion,
		Type:          "bridge_" + req.Action,
		From:          req.SourceAddress,
		To:            req.DestAddress,
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
		"message": fmt.Sprintf("Bridge %s executed successfully", req.Action),
		"data": map[string]interface{}{
			"action":         req.Action,
			"source_chain":   req.SourceChain,
			"dest_chain":     req.DestChain,
			"source_address": req.SourceAddress,
			"dest_address":   req.DestAddress,
			"token_symbol":   req.TokenSymbol,
			"amount":         req.Amount,
			"status":         "executed",
			"note":           "Transaction successfully processed by the runtime",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// testStaking handles Staking testing requests
func (s *APIServer) handleBridgeEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	walletAddress := r.URL.Query().Get("wallet")
	if walletAddress == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "wallet parameter required",
		})
		return
	}

	// Get bridge events for the wallet (simplified implementation)
	events := []map[string]interface{}{
		{
			"id":           "bridge_event_1",
			"type":         "transfer",
			"source_chain": "ethereum",
			"dest_chain":   "blackhole",
			"token_symbol": "USDT",
			"amount":       1000000,
			"from_address": walletAddress,
			"to_address":   "0x8ba1f109551bD432803012645",
			"status":       "confirmed",
			"tx_hash":      "0xabcdef1234567890",
			"timestamp":    time.Now().Unix() - 3600,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    events,
	})
}

// handleBridgeSubscribe handles bridge event subscriptions
func (s *APIServer) handleBridgeSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletAddress string `json:"wallet_address"`
		Endpoint      string `json:"endpoint"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Subscribe wallet to bridge events (simplified implementation)
	fmt.Printf("📡 Wallet %s subscribed to bridge events at %s\n", req.WalletAddress, req.Endpoint)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Successfully subscribed to bridge events",
	})
}

// handleBridgeApprovalSimulation handles bridge approval simulation
func (s *APIServer) handleBridgeApprovalSimulation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TokenSymbol string `json:"token_symbol"`
		Owner       string `json:"owner"`
		Spender     string `json:"spender"`
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

	// Simulate bridge approval using the bridge
	if s.bridge != nil {
		simulation, err := s.bridge.SimulateApproval(
			bridge.ChainTypeBlackhole,
			req.TokenSymbol,
			req.Owner,
			req.Spender,
			req.Amount,
		)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    simulation,
		})
	} else {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Bridge not available",
		})
	}
}

// handleHealth handles health check requests
