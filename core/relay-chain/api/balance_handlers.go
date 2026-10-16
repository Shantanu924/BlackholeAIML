package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *APIServer) handleBalanceQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Address     string `json:"address"`
		TokenSymbol string `json:"token_symbol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate inputs
	if req.Address == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Address is required",
		})
		return
	}

	if req.TokenSymbol == "" {
		req.TokenSymbol = "BHIV" // Default to BHIV
	}

	fmt.Printf("🔍 Balance query: address=%s, token=%s\n", req.Address, req.TokenSymbol)

	// Get token from blockchain
	token, exists := s.blockchain.TokenRegistry[req.TokenSymbol]

	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Token %s not found", req.TokenSymbol),
		})
		return
	}

	// Get balance
	balance, err := token.BalanceOf(req.Address)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get balance: %v", err),
		})
		return
	}

	fmt.Printf("✅ Balance found: %d %s for address %s\n", balance, req.TokenSymbol, req.Address)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"address":      req.Address,
			"token_symbol": req.TokenSymbol,
			"balance":      balance,
		},
	})
}

// Production Cache Balance API Handlers

// handleBalanceCached handles cached balance requests with user isolation
func (s *APIServer) handleBalanceCached(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID        string `json:"user_id"`
		Address       string `json:"address"`
		TokenSymbol   string `json:"token_symbol"`
		ForValidation bool   `json:"for_validation"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate inputs
	if req.UserID == "" || req.Address == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "UserID and Address are required",
		})
		return
	}

	if req.TokenSymbol == "" {
		req.TokenSymbol = "BHIV" // Default to BHIV
	}

	fmt.Printf("🚀 Cached balance query: user=%s, address=%s, token=%s, validation=%v\n",
		req.UserID, req.Address, req.TokenSymbol, req.ForValidation)

	// Register wallet address in account registry
	if s.blockchain.AccountRegistry != nil {
		s.blockchain.AccountRegistry.RegisterAccount(req.Address, "wallet_api", false, req.UserID, "")
	}

	// Get cached balance
	balance, err := s.blockchain.GetTokenBalanceWithCache(req.UserID, req.Address, req.TokenSymbol, req.ForValidation)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get cached balance: %v", err),
		})
		return
	}

	fmt.Printf("✅ Cached balance found: %d %s for address %s (user: %s)\n",
		balance, req.TokenSymbol, req.Address, req.UserID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"address":      req.Address,
			"token_symbol": req.TokenSymbol,
			"balance":      balance,
			"user_id":      req.UserID,
			"cached":       true,
			"validation":   req.ForValidation,
		},
	})
}

// handleBalanceAll handles requests for all token balances for an address
func (s *APIServer) handleBalanceAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID  string `json:"user_id"`
		Address string `json:"address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate inputs
	if req.UserID == "" || req.Address == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "UserID and Address are required",
		})
		return
	}

	fmt.Printf("🚀 All balances query: user=%s, address=%s\n", req.UserID, req.Address)

	// Register wallet address in account registry
	if s.blockchain.AccountRegistry != nil {
		s.blockchain.AccountRegistry.RegisterAccount(req.Address, "wallet_api", false, req.UserID, "")
	}

	// Get all cached balances
	balances, err := s.blockchain.GetAllTokenBalancesWithCache(req.UserID, req.Address)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get all balances: %v", err),
		})
		return
	}

	fmt.Printf("✅ All balances found for address %s (user: %s): %v\n", req.Address, req.UserID, balances)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"address":  req.Address,
			"balances": balances,
			"user_id":  req.UserID,
			"cached":   true,
		},
	})
}

// handleBalancePreload handles preloading balances for multiple addresses into cache
func (s *APIServer) handleBalancePreload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID    string   `json:"user_id"`
		Addresses []string `json:"addresses"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate inputs
	if req.UserID == "" || len(req.Addresses) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "UserID and Addresses are required",
		})
		return
	}

	fmt.Printf("🚀 Preload balances: user=%s, addresses=%v\n", req.UserID, req.Addresses)

	// Register all addresses in account registry
	if s.blockchain.AccountRegistry != nil {
		for _, address := range req.Addresses {
			s.blockchain.AccountRegistry.RegisterAccount(address, "wallet_api", false, req.UserID, "")
		}
	}

	// Preload balances into cache
	err := s.blockchain.PreloadUserBalances(req.UserID, req.Addresses)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to preload balances: %v", err),
		})
		return
	}

	fmt.Printf("✅ Preloaded balances for %d addresses (user: %s)\n", len(req.Addresses), req.UserID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"user_id":         req.UserID,
			"preloaded_count": len(req.Addresses),
			"addresses":       req.Addresses,
			"preloaded":       true,
		},
	})
}

// handleBalanceSimple handles simple balance requests (for backward compatibility)
func (s *APIServer) handleBalanceSimple(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse query parameters
	address := r.URL.Query().Get("address")
	tokenSymbol := r.URL.Query().Get("token")

	// Validate inputs
	if address == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Address parameter is required",
		})
		return
	}

	if tokenSymbol == "" {
		tokenSymbol = "BHIV" // Default to BHIV
	}

	fmt.Printf("🔍 Simple balance query: address=%s, token=%s\n", address, tokenSymbol)

	// Get token from blockchain
	token, exists := s.blockchain.TokenRegistry[tokenSymbol]
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Token %s not found", tokenSymbol),
		})
		return
	}

	// Get balance directly (no cache)
	balance, err := token.BalanceOf(address)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Failed to get balance: %v", err),
		})
		return
	}

	fmt.Printf("✅ Simple balance found: %d %s for address %s\n", balance, tokenSymbol, address)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"balance": balance,
		"address": address,
		"token":   tokenSymbol,
	})
}

// OTC Trading API Handlers
