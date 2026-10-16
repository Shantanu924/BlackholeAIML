package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	wallet "github.com/Shivam-Patel-G/blackhole-blockchain/services/wallet/wallet"
)

func handleWallets(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	ctx := context.Background()
	wallets, err := wallet.ListUserWallets(ctx, user)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var walletData []map[string]interface{}
	for _, w := range wallets {
		walletData = append(walletData, map[string]interface{}{
			"name":       w.WalletName,
			"address":    w.Address,
			"public_key": w.PublicKey,
			"created_at": w.CreatedAt,
		})
	}

	sendJSONResponse(w, APIResponse{Success: true, Data: walletData}, http.StatusOK)
}

// handleCreateWallet creates a new wallet
func handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	logInfo("CREATE_WALLET", "Processing wallet creation request")

	if r.Method != "POST" {
		logError("CREATE_WALLET_METHOD", fmt.Errorf("invalid method: %s", r.Method))
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		logError("CREATE_WALLET_AUTH", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	var req struct {
		WalletName string `json:"wallet_name"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logError("CREATE_WALLET_DECODE", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	logInfo("CREATE_WALLET_USER", fmt.Sprintf("Creating wallet '%s' for user %s", req.WalletName, user.Username))

	ctx := context.Background()
	newWallet, err := wallet.GenerateWalletFromMnemonic(ctx, user, req.Password, req.WalletName)
	if err != nil {
		logError("CREATE_WALLET_GENERATE", fmt.Errorf("failed to create wallet '%s' for user %s: %v", req.WalletName, user.Username, err))
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	logSuccess("CREATE_WALLET_SUCCESS", fmt.Sprintf("Wallet '%s' created successfully for user %s with address %s", req.WalletName, user.Username, newWallet.Address))

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "Wallet created successfully",
		Data: map[string]interface{}{
			"name":    newWallet.WalletName,
			"address": newWallet.Address,
		},
	}, http.StatusOK)
}

// handleCheckBalance checks wallet balance
func handleCheckBalance(w http.ResponseWriter, r *http.Request) {
	logInfo("CHECK_BALANCE", "Processing balance check request")

	if r.Method != "POST" {
		logError("CHECK_BALANCE_METHOD", fmt.Errorf("invalid method: %s", r.Method))
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		logError("CHECK_BALANCE_AUTH", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	var req struct {
		WalletName  string `json:"wallet_name"`
		Password    string `json:"password"`
		TokenSymbol string `json:"token_symbol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logError("CHECK_BALANCE_DECODE", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	logInfo("CHECK_BALANCE_REQUEST", fmt.Sprintf("Checking balance for wallet '%s', token '%s', user '%s'", req.WalletName, req.TokenSymbol, user.Username))

	ctx := context.Background()
	balance, err := wallet.CheckTokenBalance(ctx, user, req.WalletName, req.Password, req.TokenSymbol)
	if err != nil {
		logError("CHECK_BALANCE_QUERY", fmt.Errorf("failed to check balance for wallet '%s', token '%s', user '%s': %v", req.WalletName, req.TokenSymbol, user.Username, err))
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	logSuccess("CHECK_BALANCE_SUCCESS", fmt.Sprintf("Balance retrieved: %d %s for wallet '%s', user '%s'", balance, req.TokenSymbol, req.WalletName, user.Username))

	sendJSONResponse(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"balance":      balance,
			"token_symbol": req.TokenSymbol,
			"wallet_name":  req.WalletName,
		},
	}, http.StatusOK)
}

// handleCheckBalanceCached checks wallet balance using production cache system
func handleCheckBalanceCached(w http.ResponseWriter, r *http.Request) {
	logInfo("CHECK_BALANCE_CACHED", "Processing cached balance check request")

	if r.Method != "POST" {
		logError("CHECK_BALANCE_CACHED_METHOD", fmt.Errorf("invalid method: %s", r.Method))
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		logError("CHECK_BALANCE_CACHED_AUTH", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	var req struct {
		WalletName    string `json:"wallet_name"`
		Password      string `json:"password"`
		TokenSymbol   string `json:"token_symbol"`
		ForValidation bool   `json:"for_validation"` // true for transaction validation, false for UI display
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logError("CHECK_BALANCE_CACHED_DECODE", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	// Get wallet address without decryption
	ctx := context.Background()
	wlt, err := wallet.GetWalletByName(ctx, user.ID, req.WalletName)
	if err != nil {
		logError("CHECK_BALANCE_CACHED_WALLET", fmt.Errorf("failed to get wallet '%s' for user '%s': %v", req.WalletName, user.Username, err))
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to access wallet"}, http.StatusInternalServerError)
		return
	}
	walletAddress := wlt.Address

	// Get cached balance from blockchain
	balance, err := getTokenBalanceFromBlockchainCached(user.ID.Hex(), walletAddress, req.TokenSymbol, req.ForValidation)
	if err != nil {
		logError("CHECK_BALANCE_CACHED_QUERY", fmt.Errorf("failed to get cached balance for wallet '%s', token '%s', user '%s': %v", req.WalletName, req.TokenSymbol, user.Username, err))
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	logSuccess("CHECK_BALANCE_CACHED_SUCCESS", fmt.Sprintf("Cached balance retrieved: %d %s for wallet '%s', user '%s'", balance, req.TokenSymbol, req.WalletName, user.Username))

	sendJSONResponse(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"balance":      balance,
			"token_symbol": req.TokenSymbol,
			"wallet_name":  req.WalletName,
			"cached":       true,
		},
	}, http.StatusOK)
}

// handleGetAllBalances gets all token balances for user's wallets
func handleGetAllBalances(w http.ResponseWriter, r *http.Request) {
	logInfo("GET_ALL_BALANCES", "Processing get all balances request")

	if r.Method != "GET" {
		logError("GET_ALL_BALANCES_METHOD", fmt.Errorf("invalid method: %s", r.Method))
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		logError("GET_ALL_BALANCES_AUTH", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	// Get all user wallets (just metadata, no decryption needed)
	ctx := context.Background()
	wallets, err := getUserWalletsBasicInfo(ctx, user)
	if err != nil {
		logError("GET_ALL_BALANCES_WALLETS", fmt.Errorf("failed to get wallets for user '%s': %v", user.Username, err))

		// Return empty balances instead of failing completely
		sendJSONResponse(w, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"balances": make(map[string]map[string]uint64),
				"cached":   true,
				"message":  "No wallets found or wallet access error",
			},
		}, http.StatusOK)
		return
	}

	// Get all balances for all wallets
	allBalances := make(map[string]map[string]uint64)
	for _, w := range wallets {
		balances, err := getAllTokenBalancesFromBlockchainCached(user.ID.Hex(), w.Address)
		if err != nil {
			logError("GET_ALL_BALANCES_QUERY", fmt.Errorf("failed to get balances for wallet '%s': %v", w.WalletName, err))
			// Continue with other wallets, don't fail completely
			balances = make(map[string]uint64)
		}
		allBalances[w.WalletName] = balances
	}

	logSuccess("GET_ALL_BALANCES_SUCCESS", fmt.Sprintf("Retrieved balances for %d wallets for user '%s'", len(allBalances), user.Username))

	sendJSONResponse(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"balances": allBalances,
			"cached":   true,
		},
	}, http.StatusOK)
}

// handlePreloadBalances preloads balances for user's wallets into cache
func handlePreloadBalances(w http.ResponseWriter, r *http.Request) {
	logInfo("PRELOAD_BALANCES", "Processing preload balances request")

	if r.Method != "POST" {
		logError("PRELOAD_BALANCES_METHOD", fmt.Errorf("invalid method: %s", r.Method))
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		logError("PRELOAD_BALANCES_AUTH", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	// Get all user wallets
	ctx := context.Background()
	wallets, err := getUserWalletsBasicInfo(ctx, user)
	if err != nil {
		logError("PRELOAD_BALANCES_WALLETS", fmt.Errorf("failed to get wallets for user '%s': %v", user.Username, err))

		// Return success with 0 wallets instead of failing
		sendJSONResponse(w, APIResponse{
			Success: true,
			Data: map[string]interface{}{
				"preloaded_wallets": 0,
				"addresses":         []string{},
				"message":           "No wallets found or wallet access error",
			},
		}, http.StatusOK)
		return
	}

	// Extract wallet addresses
	addresses := make([]string, len(wallets))
	for i, w := range wallets {
		addresses[i] = w.Address
	}

	// Preload balances into blockchain cache
	err = preloadUserBalancesInBlockchain(user.ID.Hex(), addresses)
	if err != nil {
		logError("PRELOAD_BALANCES_BLOCKCHAIN", fmt.Errorf("failed to preload balances for user '%s': %v", user.Username, err))
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to preload balances"}, http.StatusInternalServerError)
		return
	}

	logSuccess("PRELOAD_BALANCES_SUCCESS", fmt.Sprintf("Preloaded balances for %d wallets for user '%s'", len(addresses), user.Username))

	sendJSONResponse(w, APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"preloaded_wallets": len(addresses),
			"addresses":         addresses,
		},
	}, http.StatusOK)
}

// handleTransfer transfers tokens
func handleTransfer(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	var req struct {
		WalletName  string `json:"wallet_name"`
		Password    string `json:"password"`
		ToAddress   string `json:"to_address"`
		TokenSymbol string `json:"token_symbol"`
		Amount      uint64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err = wallet.TransferTokensWithHistory(ctx, user, req.WalletName, req.Password, req.ToAddress, req.TokenSymbol, req.Amount)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully transferred %d %s tokens to %s", req.Amount, req.TokenSymbol, req.ToAddress),
	}, http.StatusOK)
}

// handleImportWallet imports a wallet from private key
func handleImportWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	var req struct {
		WalletName string `json:"wallet_name"`
		Password   string `json:"password"`
		PrivateKey string `json:"private_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	importedWallet, err := wallet.ImportWalletFromPrivateKey(ctx, user, req.Password, req.WalletName, req.PrivateKey)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "Wallet imported successfully",
		Data: map[string]interface{}{
			"name":    importedWallet.WalletName,
			"address": importedWallet.Address,
		},
	}, http.StatusOK)
}

// handleExportWallet exports wallet private key
func handleExportWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	var req struct {
		WalletName string `json:"wallet_name"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	privateKeyHex, err := wallet.ExportWalletPrivateKey(ctx, user, req.WalletName, req.Password)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "Private key exported successfully",
		Data: map[string]interface{}{
			"private_key": privateKeyHex,
		},
	}, http.StatusOK)
}

// handleStakeTokens handles token staking
func handleStakeTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	var req struct {
		WalletName  string `json:"wallet_name"`
		Password    string `json:"password"`
		TokenSymbol string `json:"token_symbol"`
		Amount      uint64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	err = wallet.StakeTokensWithHistory(ctx, user, req.WalletName, req.Password, req.TokenSymbol, req.Amount)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Successfully staked %d %s tokens", req.Amount, req.TokenSymbol),
	}, http.StatusOK)
}

// handleTransactions returns transaction history
func handleTransactions(w http.ResponseWriter, r *http.Request) {
	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication error"}, http.StatusUnauthorized)
		return
	}

	ctx := context.Background()
	transactions, err := wallet.GetAllUserTransactions(ctx, user.ID.Hex(), 50)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var txData []map[string]interface{}
	for _, tx := range transactions {
		txData = append(txData, map[string]interface{}{
			"type":         tx.Type,
			"from":         tx.From,
			"to":           tx.To,
			"amount":       tx.Amount,
			"token_symbol": tx.TokenSymbol,
			"status":       tx.Status,
			"timestamp":    tx.Timestamp,
			"block_height": tx.BlockHeight,
		})
	}

	sendJSONResponse(w, APIResponse{Success: true, Data: txData}, http.StatusOK)
}

// OTC Trading Handlers
func handleTestCache(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Test blockchain connection
	blockchainConnected := testBlockchainConnection()

	// Test cache API
	cacheWorking := false
	var cacheError string

	testBalance, err := getTokenBalanceFromBlockchainCached("test_user", "test_address", "BHIV", false)
	if err != nil {
		cacheError = err.Error()
	} else {
		cacheWorking = true
	}

	response := map[string]interface{}{
		"cache_system_status":  "operational",
		"blockchain_connected": blockchainConnected,
		"cache_api_working":    cacheWorking,
		"test_balance":         testBalance,
		"timestamp":            time.Now().Format(time.RFC3339),
	}

	if cacheError != "" {
		response["cache_error"] = cacheError
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// handleTestBalance tests balance for the known address with balances
func handleTestBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Test the address that should have balances
	knownAddress := "03e2459b73c0c6522530f6b26e834d992dfc55d170bee35d0bcdc047fe0d61c25b"

	// Test all tokens
	tokens := []string{"BHIV", "ETH", "USDT"}
	balances := make(map[string]uint64)

	for _, token := range tokens {
		balance, err := getTokenBalanceFromBlockchainCached("test_user", knownAddress, token, false)
		if err != nil {
			balances[token] = 0
		} else {
			balances[token] = balance
		}
	}

	response := map[string]interface{}{
		"known_address": knownAddress,
		"balances":      balances,
		"timestamp":     time.Now().Format(time.RFC3339),
		"note":          "This should show the persistent balances from blockchain logs",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    response,
	})
}

// WalletBasicInfo represents basic wallet information without sensitive data
type WalletBasicInfo struct {
	WalletName string    `json:"wallet_name"`
	Address    string    `json:"address"`
	CreatedAt  time.Time `json:"created_at"`
}

// handleWalletIdentity — VAJRA Phase 1: Runtime Identity Card.
// GET /identity
func handleWalletIdentity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":     "vajra-wallet",
		"trust_layer": "Wallet",
		"port":        9000,
		"authority": []string{
			"WALLET_INTENT",
			"KEY_MANAGEMENT",
			"TRANSACTION_SIGNING",
			"USER_AUTHENTICATION",
		},
		"non_authority": []string{
			"PDV_EQUALITY",
			"GOVERNANCE_LEGITIMACY",
			"BLOCKCHAIN_WRITE",
			"FRAUD_DETECTION",
		},
		"adjacent": []string{
			"blockchain-node:8080",
			"bridge:8084",
		},
		"endpoints": []string{
			"POST /api/wallets/transfer",
			"GET  /api/wallets",
			"POST /api/wallets/create",
			"GET  /api/status",
			"GET  /identity",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
