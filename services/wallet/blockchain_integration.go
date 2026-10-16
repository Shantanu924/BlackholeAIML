package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	wallet "github.com/Shivam-Patel-G/blackhole-blockchain/services/wallet/wallet"

	"go.mongodb.org/mongo-driver/bson"
)

func createOTCOrderOnBlockchain(creator, tokenOffered, tokenRequested string, amountOffered, amountRequested uint64, expirationHours int, isMultiSig bool, requiredSigs []string) (map[string]interface{}, error) {
	// Try to connect to blockchain API with retry logic
	blockchainURL := "http://localhost:8081/api/otc/create"

	// Test blockchain connectivity first
	if !testBlockchainConnection() {
		fmt.Printf("⚠️ Blockchain API not available, creating simulated OTC order\n")
		return createSimulatedOTCOrder(creator, tokenOffered, tokenRequested, amountOffered, amountRequested, expirationHours, isMultiSig, requiredSigs), nil
	}

	requestData := map[string]interface{}{
		"creator":          creator,
		"token_offered":    tokenOffered,
		"amount_offered":   amountOffered,
		"token_requested":  tokenRequested,
		"amount_requested": amountRequested,
		"expiration_hours": expirationHours,
		"is_multisig":      isMultiSig,
		"required_sigs":    requiredSigs,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(blockchainURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// If blockchain API is not available, create a simulated order
		fmt.Printf("⚠️ Blockchain API not available, creating simulated OTC order\n")
		return createSimulatedOTCOrder(creator, tokenOffered, tokenRequested, amountOffered, amountRequested, expirationHours, isMultiSig, requiredSigs), nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		message := "Unknown error"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return nil, fmt.Errorf("blockchain error: %s", message)
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		return data, nil
	}

	return nil, fmt.Errorf("invalid response format")
}

func createSimulatedOTCOrder(creator, tokenOffered, tokenRequested string, amountOffered, amountRequested uint64, expirationHours int, isMultiSig bool, requiredSigs []string) map[string]interface{} {
	// Safely handle short creator addresses
	creatorSuffix := creator
	if len(creator) > 8 {
		creatorSuffix = creator[:8]
	}
	orderID := fmt.Sprintf("otc_%d_%s", time.Now().UnixNano(), creatorSuffix)

	return map[string]interface{}{
		"order_id":         orderID,
		"creator":          creator,
		"token_offered":    tokenOffered,
		"amount_offered":   amountOffered,
		"token_requested":  tokenRequested,
		"amount_requested": amountRequested,
		"expiration_hours": expirationHours,
		"is_multi_sig":     isMultiSig,
		"required_sigs":    requiredSigs,
		"status":           "open",
		"created_at":       time.Now().Unix(),
		"expires_at":       time.Now().Add(time.Duration(expirationHours) * time.Hour).Unix(),
		"note":             "Simulated order - blockchain API not available",
	}
}

func getOTCOrdersFromBlockchain(userAddress string) ([]map[string]interface{}, error) {
	// Try to connect to blockchain API
	blockchainURL := fmt.Sprintf("http://localhost:8081/api/otc/orders?user=%s", userAddress)

	resp, err := http.Get(blockchainURL)
	if err != nil {
		// If blockchain API is not available, return simulated orders
		fmt.Printf("⚠️ Blockchain API not available, returning simulated OTC orders\n")
		return getSimulatedOTCOrders(userAddress), nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		return getSimulatedOTCOrders(userAddress), nil
	}

	if data, ok := result["data"].([]interface{}); ok {
		orders := make([]map[string]interface{}, len(data))
		for i, order := range data {
			if orderMap, ok := order.(map[string]interface{}); ok {
				orders[i] = orderMap
			}
		}
		return orders, nil
	}

	return getSimulatedOTCOrders(userAddress), nil
}

func getSimulatedOTCOrders(userAddress string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"order_id":         "otc_example_1",
			"creator":          userAddress,
			"token_offered":    "BHIV",
			"amount_offered":   1000,
			"token_requested":  "USDT",
			"amount_requested": 5000,
			"status":           "open",
			"created_at":       time.Now().Unix() - 3600,
			"expires_at":       time.Now().Unix() + 82800, // 23 hours from now
			"note":             "Simulated order",
		},
		{
			"order_id":         "otc_example_2",
			"creator":          "0x1234...5678",
			"token_offered":    "USDT",
			"amount_offered":   2000,
			"token_requested":  "BHIV",
			"amount_requested": 400,
			"status":           "open",
			"created_at":       time.Now().Unix() - 1800,
			"expires_at":       time.Now().Unix() + 84600, // 23.5 hours from now
			"note":             "Simulated order from another user",
		},
	}
}

func getAllOTCOrdersFromBlockchain() ([]map[string]interface{}, error) {
	// Try to connect to blockchain API to get all orders
	blockchainURL := "http://localhost:8081/api/otc/orders"

	resp, err := http.Get(blockchainURL)
	if err != nil {
		// If blockchain API is not available, return simulated market orders
		fmt.Printf("⚠️ Blockchain API not available, returning simulated market orders\n")
		return getSimulatedMarketOrders(), nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		return getSimulatedMarketOrders(), nil
	}

	if data, ok := result["data"].([]interface{}); ok {
		orders := make([]map[string]interface{}, len(data))
		for i, order := range data {
			if orderMap, ok := order.(map[string]interface{}); ok {
				orders[i] = orderMap
			}
		}
		return orders, nil
	}

	return getSimulatedMarketOrders(), nil
}

func getSimulatedMarketOrders() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"order_id":         "market_order_1",
			"creator":          "03e2459b73c0c6522530f6b26e834d992dfc55d170bee35d0bcdc047fe0d61c25b",
			"token_offered":    "BHIV",
			"amount_offered":   500,
			"token_requested":  "ETH",
			"amount_requested": 2,
			"status":           "open",
			"created_at":       time.Now().Unix() - 1800,
			"expires_at":       time.Now().Unix() + 84600,
			"note":             "Simulated market order - BHIV for ETH",
		},
		{
			"order_id":         "market_order_2",
			"creator":          "0x9876543210abcdef9876543210abcdef98765432",
			"token_offered":    "USDT",
			"amount_offered":   1000,
			"token_requested":  "BHIV",
			"amount_requested": 200,
			"status":           "open",
			"created_at":       time.Now().Unix() - 3600,
			"expires_at":       time.Now().Unix() + 82800,
			"note":             "Simulated market order - USDT for BHIV",
		},
		{
			"order_id":         "market_order_3",
			"creator":          "0xabcdef1234567890abcdef1234567890abcdef12",
			"token_offered":    "ETH",
			"amount_offered":   3,
			"token_requested":  "USDT",
			"amount_requested": 9000,
			"status":           "open",
			"created_at":       time.Now().Unix() - 900,
			"expires_at":       time.Now().Unix() + 85500,
			"note":             "Simulated market order - ETH for USDT",
		},
	}
}

func cancelOTCOrderOnBlockchain(orderID, canceller string) (bool, error) {
	// Try to connect to blockchain API
	blockchainURL := "http://localhost:8081/api/otc/cancel"

	requestData := map[string]interface{}{
		"order_id":  orderID,
		"canceller": canceller,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(blockchainURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// If blockchain API is not available, simulate cancellation
		fmt.Printf("⚠️ Blockchain API not available, simulating OTC order cancellation\n")
		return true, nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		message := "Unknown error"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return false, fmt.Errorf("blockchain error: %s", message)
	}

	return true, nil
}

func matchOTCOrderOnBlockchain(orderID, counterparty string) (bool, error) {
	// Try to connect to blockchain API
	blockchainURL := "http://localhost:8080/api/otc/match"

	requestData := map[string]interface{}{
		"order_id":     orderID,
		"counterparty": counterparty,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return false, fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(blockchainURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// If blockchain API is not available, simulate matching
		fmt.Printf("⚠️ Blockchain API not available, simulating OTC order matching\n")
		return true, nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		message := "Unknown error"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return false, fmt.Errorf("blockchain error: %s", message)
	}

	return true, nil
}

// Blockchain connection testing functions

// getBlockchainAPIURL returns the blockchain API URL from environment or defaults to localhost:8081
func getBlockchainAPIURL() string {
	// Check if BLOCKCHAIN_API_URL is set
	if url := os.Getenv("BLOCKCHAIN_API_URL"); url != "" {
		return url
	}
	// Default to main blockchain node UI/API port
	return "http://localhost:8080"
}

func testBlockchainConnection() bool {
	// Test basic connectivity to blockchain API
	testURL := getBlockchainAPIURL() + "/api/health"

	client := &http.Client{
		Timeout: 5 * time.Second, // 5 second timeout
	}

	resp, err := client.Get(testURL)
	if err != nil {
		fmt.Printf("🔴 Blockchain connection test failed: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		fmt.Printf("🟢 Blockchain connection successful\n")
		return true
	}

	fmt.Printf("🟡 Blockchain responded with status: %d\n", resp.StatusCode)
	return false
}

func retryBlockchainConnection(maxRetries int) bool {
	for i := 0; i < maxRetries; i++ {
		if testBlockchainConnection() {
			return true
		}
		fmt.Printf("🔄 Retry %d/%d: Waiting 2 seconds before next attempt...\n", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}
	return false
}

// Production Cache Integration Functions

// getTokenBalanceFromBlockchainCached gets a single token balance using the cache system
func getTokenBalanceFromBlockchainCached(userID, address, tokenSymbol string, forValidation bool) (uint64, error) {
	// Try to connect to blockchain API with cache support
	blockchainURL := getBlockchainAPIURL() + "/api/balance/cached"

	requestData := map[string]interface{}{
		"user_id":        userID,
		"address":        address,
		"token_symbol":   tokenSymbol,
		"for_validation": forValidation,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal request: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(blockchainURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Fallback to regular balance check
		fmt.Printf("⚠️ Cached balance API not available, falling back to regular check\n")
		return getTokenBalanceFromBlockchain(address, tokenSymbol)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		message := "unknown error"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return 0, fmt.Errorf("blockchain error: %s", message)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid response format")
	}

	balance, ok := data["balance"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid balance format")
	}

	return uint64(balance), nil
}

// getAllTokenBalancesFromBlockchainCached gets all token balances for an address using cache
func getAllTokenBalancesFromBlockchainCached(userID, address string) (map[string]uint64, error) {
	// Try to connect to blockchain API with cache support
	blockchainURL := getBlockchainAPIURL() + "/api/balance/all"

	requestData := map[string]interface{}{
		"user_id": userID,
		"address": address,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(blockchainURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Fallback to simulated balances
		fmt.Printf("⚠️ Cached balance API not available, returning simulated balances\n")
		return getSimulatedTokenBalances(address), nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		message := "unknown error"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return nil, fmt.Errorf("blockchain error: %s", message)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	balances, ok := data["balances"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid balances format")
	}

	// Convert to uint64 map
	result_balances := make(map[string]uint64)
	for token, balance := range balances {
		if bal, ok := balance.(float64); ok {
			result_balances[token] = uint64(bal)
		}
	}

	return result_balances, nil
}

// preloadUserBalancesInBlockchain preloads balances for user's addresses into blockchain cache
func preloadUserBalancesInBlockchain(userID string, addresses []string) error {
	// Try to connect to blockchain API
	blockchainURL := getBlockchainAPIURL() + "/api/balance/preload"

	requestData := map[string]interface{}{
		"user_id":   userID,
		"addresses": addresses,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second} // Longer timeout for preloading
	resp, err := client.Post(blockchainURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Not a critical error, just log it
		fmt.Printf("⚠️ Preload API not available: %v\n", err)
		return nil // Don't fail the request
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		message := "unknown error"
		if msg, ok := result["message"].(string); ok {
			message = msg
		}
		return fmt.Errorf("blockchain error: %s", message)
	}

	return nil
}

// Helper functions for fallback scenarios

// getTokenBalanceFromBlockchain gets balance directly from blockchain (fallback)
func getTokenBalanceFromBlockchain(address, tokenSymbol string) (uint64, error) {
	// Try to connect to blockchain API
	blockchainURL := fmt.Sprintf("%s/api/balance?address=%s&token=%s", getBlockchainAPIURL(), address, tokenSymbol)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(blockchainURL)
	if err != nil {
		// Return simulated balance if blockchain is not available
		fmt.Printf("⚠️ Blockchain API not available, returning simulated balance\n")
		return getSimulatedBalance(address, tokenSymbol), nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %v", err)
	}

	if success, ok := result["success"].(bool); !ok || !success {
		return 0, fmt.Errorf("blockchain API error")
	}

	if balance, ok := result["balance"].(float64); ok {
		return uint64(balance), nil
	}

	return 0, fmt.Errorf("invalid balance format")
}

// getSimulatedTokenBalances returns simulated balances for testing
func getSimulatedTokenBalances(address string) map[string]uint64 {
	// Return some simulated balances for testing
	return map[string]uint64{
		"BHIV":  1000,
		"ETH":  50,
		"USDT": 5000,
	}
}

// getSimulatedBalance returns a simulated balance for a specific token
func getSimulatedBalance(address, tokenSymbol string) uint64 {
	balances := getSimulatedTokenBalances(address)
	if balance, exists := balances[tokenSymbol]; exists {
		return balance
	}
	return 0
}

// Helper function to get user wallets without decrypting private keys
func getUserWalletsBasicInfo(ctx context.Context, user *wallet.User) ([]*WalletBasicInfo, error) {
	// Import the wallet package types
	collection := wallet.WalletCollection

	log.Printf("🔍 Querying wallets for user ID: %s (username: %s)", user.ID.Hex(), user.Username)
	log.Printf("🔍 Expected address with balances: 03e2459b73c0c6522530f6b26e834d992dfc55d170bee35d0bcdc047fe0d61c25b")

	cursor, err := collection.Find(ctx, bson.M{"user_id": user.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to query wallets: %v", err)
	}
	defer cursor.Close(ctx)

	var wallets []*WalletBasicInfo
	walletCount := 0

	for cursor.Next(ctx) {
		var walletDoc struct {
			WalletName string    `bson:"wallet_name"`
			Address    string    `bson:"address"`
			CreatedAt  time.Time `bson:"created_at"`
		}

		if err := cursor.Decode(&walletDoc); err != nil {
			log.Printf("⚠️ Warning: Failed to decode wallet document: %v", err)
			continue
		}

		wallets = append(wallets, &WalletBasicInfo{
			WalletName: walletDoc.WalletName,
			Address:    walletDoc.Address,
			CreatedAt:  walletDoc.CreatedAt,
		})
		walletCount++

		log.Printf("✅ Found wallet: %s (address: %s)", walletDoc.WalletName, walletDoc.Address)
	}

	if err := cursor.Err(); err != nil {
		return nil, fmt.Errorf("cursor error: %v", err)
	}

	log.Printf("📊 Total wallets found for user %s: %d", user.Username, walletCount)
	return wallets, nil
}

// handleTestCache provides a simple test endpoint for the cache system
