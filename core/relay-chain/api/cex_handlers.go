package api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/chain"
)

// handleCEXBlocksLatest returns the latest finalized block for CEX polling
func (s *APIServer) handleCEXBlocksLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	blocksCount := len(s.blockchain.Blocks)
	var latestBlock *chain.Block
	if blocksCount > 0 {
		latestBlock = s.blockchain.Blocks[blocksCount-1]
	}

	if latestBlock == nil {
		http.Error(w, "No blocks available", http.StatusNotFound)
		return
	}

	// Format transactions for CEX
	var formattedTxs []map[string]interface{}
	for _, tx := range latestBlock.Transactions {
		formattedTxs = append(formattedTxs, map[string]interface{}{
			"tx_id":        tx.ID,
			"type":         tx.Type,
			"from":         tx.From,
			"to":           tx.To,
			"token_symbol": tx.TokenID,
			"amount":       fmt.Sprintf("%d", tx.Amount),
			"status":       "CONFIRMED",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"height":       latestBlock.Header.Index,
			"hash":         latestBlock.Hash,
			"transactions": formattedTxs,
		},
	})
}

// handleCEXBlocksHeight returns a specific historical block by height
func (s *APIServer) handleCEXBlocksHeight(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract height from URL path (e.g., /api/blocks/123)
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, "Invalid block height parameter", http.StatusBadRequest)
		return
	}
	
	heightStr := parts[3]
	height, err := strconv.ParseUint(heightStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid block height format", http.StatusBadRequest)
		return
	}

	var targetBlock *chain.Block
	if int(height) < len(s.blockchain.Blocks) {
		targetBlock = s.blockchain.Blocks[height]
	}

	if targetBlock == nil {
		http.Error(w, "Block not found", http.StatusNotFound)
		return
	}

	// Format transactions for CEX
	var formattedTxs []map[string]interface{}
	for _, tx := range targetBlock.Transactions {
		formattedTxs = append(formattedTxs, map[string]interface{}{
			"tx_id":        tx.ID,
			"type":         tx.Type,
			"from":         tx.From,
			"to":           tx.To,
			"token_symbol": tx.TokenID,
			"amount":       fmt.Sprintf("%d", tx.Amount),
			"status":       "CONFIRMED",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"height":       targetBlock.Header.Index,
			"hash":         targetBlock.Hash,
			"transactions": formattedTxs,
		},
	})
}

// handleCEXBroadcast accepts a raw signed transaction (hex) and adds it to the mempool
func (s *APIServer) handleCEXBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SignedTxHex string `json:"signed_tx_hex"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SignedTxHex == "" {
		http.Error(w, "signed_tx_hex is required", http.StatusBadRequest)
		return
	}

	// Strip "0x" prefix if present
	hexStr := strings.TrimPrefix(req.SignedTxHex, "0x")
	
	txBytes, err := hex.DecodeString(hexStr)
	if err != nil {
		http.Error(w, "Invalid hex encoding", http.StatusBadRequest)
		return
	}

	var tx chain.Transaction
	if err := json.Unmarshal(txBytes, &tx); err != nil {
		// Try gob decoding if JSON fails, since we often serialize with gob in the node
		// But let's assume JSON for now as it's standard for external APIs.
		http.Error(w, "Failed to decode transaction payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Submit to mempool
	if err := s.blockchain.ProcessTransaction(&tx); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to add transaction: " + err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"tx_id":   tx.ID,
		"status":  "PENDING",
	})
}
