package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// handleExplorerBlocks returns the latest blocks from the chain
func (s *APIServer) handleExplorerBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limit := 10
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Fetch actual blocks from the blockchain
	totalBlocks := len(s.blockchain.Blocks)
	startIdx := totalBlocks - limit
	if startIdx < 0 {
		startIdx = 0
	}

	blocks := []map[string]interface{}{}
	for i := totalBlocks - 1; i >= startIdx; i-- {
		block := s.blockchain.Blocks[i]
		blocks = append(blocks, map[string]interface{}{
			"index":     block.Header.Index,
			"timestamp": block.Header.Timestamp.Unix(),
			"hash":      block.Hash,
			"txCount":   len(block.Transactions),
			"miner":     block.Header.Validator,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data":   blocks,
	})
}

// handleExplorerTransaction returns transaction details by hash
func (s *APIServer) handleExplorerTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "Transaction hash required", http.StatusBadRequest)
		return
	}

	var foundTx interface{}
	var txBlock uint64
	var txStatus string

	// Search in blocks
	for _, block := range s.blockchain.Blocks {
		for _, tx := range block.Transactions {
			if tx.ID == hash {
				foundTx = tx
				txBlock = block.Header.Index
				txStatus = "confirmed"
				break
			}
		}
		if foundTx != nil {
			break
		}
	}

	// Search in pending pool if not found
	if foundTx == nil {
		for _, tx := range s.blockchain.PendingTxs {
			if tx.ID == hash {
				foundTx = tx
				txStatus = "pending"
				break
			}
		}
	}

	if foundTx == nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"transaction": foundTx,
			"status":      txStatus,
			"block":       txBlock,
		},
	})
}

// handleExplorerAddress returns address details (balance, tx count)
func (s *APIServer) handleExplorerAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Address required", http.StatusBadRequest)
		return
	}

	// Calculate transaction count and find latest txs
	txCount := 0
	for _, block := range s.blockchain.Blocks {
		for _, tx := range block.Transactions {
			if tx.From == address || tx.To == address {
				txCount++
			}
		}
	}

	// For pending txs
	for _, tx := range s.blockchain.PendingTxs {
		if tx.From == address || tx.To == address {
			txCount++
		}
	}

	// Get balance safely using blockchain method
	balance := s.blockchain.GetTokenBalance(address, "BHIV")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"address": address,
			"balance": strconv.FormatUint(balance, 10) + " BHIV",
			"txCount": txCount,
		},
	})
}

// handleExplorerStats returns network statistics
func (s *APIServer) handleExplorerStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	totalTxs := 0
	for _, block := range s.blockchain.Blocks {
		totalTxs += len(block.Transactions)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"network":          "BlackHole Infiverse",
			"nativeCoin":       "BHIV",
			"currentBlock":     len(s.blockchain.Blocks) - 1,
			"totalTxs":         totalTxs,
			"activeValidators": len(s.blockchain.StakeLedger.GetAllStakes()), // Count of stakers
			"tps":              0, // calculate if needed, or return basic stats
		},
	})
}

// handleTransactionHistory returns the transaction history for a specific address.
func (s *APIServer) handleTransactionHistory(w http.ResponseWriter, r *http.Request) {
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "missing address parameter", http.StatusBadRequest)
		return
	}

	var txHistory []map[string]interface{}

	// Scan blocks backward for latest transactions (very rudimentary for testnet phase)
	for i := len(s.blockchain.Blocks) - 1; i >= 0; i-- {
		block := s.blockchain.Blocks[i]
		for _, tx := range block.Transactions {
			if tx.From == address || tx.To == address {
				txHistory = append(txHistory, map[string]interface{}{
					"id":        tx.ID,
					"type":      tx.Type,
					"from":      tx.From,
					"to":        tx.To,
					"amount":    tx.Amount,
					"token":     tx.TokenID,
					"timestamp": tx.Timestamp,
					"block":     block.Header.Index,
					"status":    "confirmed",
				})
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    txHistory,
	})
}
