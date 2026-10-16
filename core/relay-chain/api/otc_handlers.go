package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/runtime"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
)

func (s *APIServer) testOTC(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action          string `json:"action"`
		Creator         string `json:"creator"`
		TokenOffered    string `json:"token_offered"`
		AmountOffered   uint64 `json:"amount_offered"`
		TokenRequested  string `json:"token_requested"`
		AmountRequested uint64 `json:"amount_requested"`
		OrderID         string `json:"order_id"`
		Counterparty    string `json:"counterparty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔧 DEV MODE: Executing OTC function '%s'\n", req.Action)

	contract := &schema.TxContract{
		SchemaVersion: schema.CurrentVersion,
		Type:          "otc_" + req.Action,
		From:          req.Creator,
		To:            req.Counterparty,
		Amount:        req.AmountOffered,
		TokenID:       req.TokenOffered,
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
		"message": fmt.Sprintf("OTC %s executed successfully", req.Action),
		"data": map[string]interface{}{
			"action": req.Action,
			"status": "executed",
			"note":   "Transaction successfully processed by the runtime",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// testEscrow handles Escrow testing requests
func (s *APIServer) handleOTCCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	var req struct {
		Creator         string   `json:"creator"`
		TokenOffered    string   `json:"token_offered"`
		AmountOffered   uint64   `json:"amount_offered"`
		TokenRequested  string   `json:"token_requested"`
		AmountRequested uint64   `json:"amount_requested"`
		ExpirationHours int      `json:"expiration_hours"`
		IsMultiSig      bool     `json:"is_multisig"`
		RequiredSigs    []string `json:"required_sigs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🤝 Creating OTC order: %+v\n", req)

	// Create OTC order with real token locking
	// Generate order ID
	orderID := fmt.Sprintf("otc_%d_%s", time.Now().UnixNano(), req.Creator[:8])

	// Validate tokens exist
	offeredToken, exists := s.blockchain.TokenRegistry[req.TokenOffered]
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Token %s not found", req.TokenOffered),
		})
		return
	}

	_, exists = s.blockchain.TokenRegistry[req.TokenRequested]
	if !exists {
		fmt.Printf("❌ Token %s not found in registry. Available tokens: %v\n", req.TokenRequested, getTokenNames(s.blockchain.TokenRegistry))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Token %s not found", req.TokenRequested),
		})
		return
	}

	// Check creator's balance
	balance, err := offeredToken.BalanceOf(req.Creator)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to check balance: " + err.Error(),
		})
		return
	}

	if balance < req.AmountOffered {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Insufficient balance: has %d, needs %d", balance, req.AmountOffered),
		})
		return
	}

	// Lock offered tokens in OTC contract
	err = offeredToken.Transfer(req.Creator, "otc_contract", req.AmountOffered)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to lock tokens: " + err.Error(),
		})
		return
	}

	fmt.Printf("✅ OTC order created: %s\n", orderID)

	// Store the order for future operations
	orderData := map[string]interface{}{
		"order_id":         orderID,
		"creator":          req.Creator,
		"token_offered":    req.TokenOffered,
		"amount_offered":   req.AmountOffered,
		"token_requested":  req.TokenRequested,
		"amount_requested": req.AmountRequested,
		"expiration_hours": req.ExpirationHours,
		"is_multi_sig":     req.IsMultiSig,
		"required_sigs":    req.RequiredSigs,
		"status":           "open",
		"created_at":       time.Now().Unix(),
		"expires_at":       time.Now().Add(time.Duration(req.ExpirationHours) * time.Hour).Unix(),
	}

	// Store the order for future operations
	s.storeOTCOrder(orderID, orderData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "OTC order created successfully",
		"data":    orderData,
	})
}

func (s *APIServer) handleOTCOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	// Get user parameter from query string
	userAddress := r.URL.Query().Get("user")

	fmt.Printf("🔍 Getting OTC orders for user: %s\n", userAddress)

	// Get orders from storage
	var orderData []map[string]interface{}

	for _, order := range otcOrderStore {
		// Filter by user if specified
		if userAddress != "" {
			creator, ok := order["creator"].(string)
			if !ok || creator != userAddress {
				continue
			}
		}

		// Check if order is still valid (not expired)
		expiresAt, ok := order["expires_at"].(int64)
		if ok && time.Now().Unix() > expiresAt {
			// Mark as expired
			order["status"] = "expired"
		}

		orderData = append(orderData, order)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    orderData,
	})
}

func (s *APIServer) handleOTCMatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	var req struct {
		OrderID      string `json:"order_id"`
		Counterparty string `json:"counterparty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🤝 Matching OTC order %s with counterparty %s\n", req.OrderID, req.Counterparty)

	// Get the order from storage
	order, exists := s.getOTCOrder(req.OrderID)
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Order not found",
		})
		return
	}

	// Check if order is still open
	status, ok := order["status"].(string)
	if !ok || status != "open" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Order is not available for matching",
		})
		return
	}

	// Check if order has expired
	expiresAt, ok := order["expires_at"].(int64)
	if ok && time.Now().Unix() > expiresAt {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Order has expired",
		})
		return
	}

	// Perform the token swap
	err := s.executeRealOTCTokenSwap(order, req.Counterparty)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to execute token swap: " + err.Error(),
		})
		return
	}

	// Update order status
	order["status"] = "completed"
	order["matched_with"] = req.Counterparty
	order["matched_at"] = time.Now().Unix()
	order["completed_at"] = time.Now().Unix()
	s.storeOTCOrder(req.OrderID, order)

	fmt.Printf("✅ OTC order %s matched with %s\n", req.OrderID, req.Counterparty)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "OTC order matched successfully",
		"data": map[string]interface{}{
			"order_id":     req.OrderID,
			"counterparty": req.Counterparty,
			"matched_at":   time.Now().Unix(),
		},
	})
}

func (s *APIServer) handleOTCCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	var req struct {
		OrderID   string `json:"order_id"`
		Canceller string `json:"canceller"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("❌ Cancelling OTC order %s by %s\n", req.OrderID, req.Canceller)

	// Get the order from storage
	order, exists := s.getOTCOrder(req.OrderID)
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Order not found",
		})
		return
	}

	// Check if canceller is the creator
	creator, ok := order["creator"].(string)
	if !ok || creator != req.Canceller {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Only order creator can cancel",
		})
		return
	}

	// Check if order can be cancelled
	status, ok := order["status"].(string)
	if !ok || status != "open" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Order cannot be cancelled in current status",
		})
		return
	}

	// Release locked tokens back to creator
	err := s.releaseOTCTokens(order)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Failed to release tokens: " + err.Error(),
		})
		return
	}

	// Update order status
	order["status"] = "cancelled"
	order["cancelled_at"] = time.Now().Unix()
	s.storeOTCOrder(req.OrderID, order)

	fmt.Printf("✅ OTC order %s cancelled by %s\n", req.OrderID, req.Canceller)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "OTC order cancelled successfully",
		"data": map[string]interface{}{
			"order_id":     req.OrderID,
			"status":       "cancelled",
			"cancelled_at": time.Now().Unix(),
		},
	})
}

// OTC Order Management Functions
func (s *APIServer) executeOTCOrderMatch(orderID, counterparty string) (bool, error) {
	fmt.Printf("🔄 Executing OTC order match: %s with %s\n", orderID, counterparty)

	// In a real implementation, this would:
	// 1. Find the order in the OTC manager
	// 2. Validate counterparty has required tokens
	// 3. Execute the token swap
	// 4. Update order status

	// For now, simulate a successful match with actual token transfers
	// This demonstrates the complete flow

	// Simulate order data (in real implementation, this would come from OTC manager)
	orderData := map[string]interface{}{
		"creator":          "test_creator",
		"token_offered":    "BHIV",
		"amount_offered":   uint64(1000),
		"token_requested":  "USDT",
		"amount_requested": uint64(5000),
	}

	// Check if counterparty has required tokens
	if requestedToken, exists := s.blockchain.TokenRegistry[orderData["token_requested"].(string)]; exists {
		balance, err := requestedToken.BalanceOf(counterparty)
		if err != nil {
			return false, fmt.Errorf("failed to check counterparty balance: %v", err)
		}

		if balance < orderData["amount_requested"].(uint64) {
			return false, fmt.Errorf("counterparty has insufficient balance: has %d, needs %d",
				balance, orderData["amount_requested"].(uint64))
		}

		// Execute the token swap
		// 1. Transfer offered tokens from OTC contract to counterparty
		if offeredToken, exists := s.blockchain.TokenRegistry[orderData["token_offered"].(string)]; exists {
			err = offeredToken.Transfer("otc_contract", counterparty, orderData["amount_offered"].(uint64))
			if err != nil {
				return false, fmt.Errorf("failed to transfer offered tokens: %v", err)
			}
		}

		// 2. Transfer requested tokens from counterparty to creator
		err = requestedToken.Transfer(counterparty, orderData["creator"].(string), orderData["amount_requested"].(uint64))
		if err != nil {
			return false, fmt.Errorf("failed to transfer requested tokens: %v", err)
		}

		fmt.Printf("✅ OTC trade completed: %d %s ↔ %d %s\n",
			orderData["amount_offered"], orderData["token_offered"],
			orderData["amount_requested"], orderData["token_requested"])

		return true, nil
	}

	return false, fmt.Errorf("requested token not found")
}

// Store for OTC orders (in real implementation, this would be in the blockchain)
var otcOrderStore = make(map[string]map[string]interface{})

// Store for Cross-Chain DEX orders
var crossChainOrderStore = make(map[string]map[string]interface{})
var crossChainOrdersByUser = make(map[string][]string) // user -> order IDs

func (s *APIServer) storeOTCOrder(orderID string, orderData map[string]interface{}) {
	otcOrderStore[orderID] = orderData
}

func (s *APIServer) getOTCOrder(orderID string) (map[string]interface{}, bool) {
	order, exists := otcOrderStore[orderID]
	return order, exists
}

// executeRealOTCTokenSwap performs the actual token swap for OTC orders
func (s *APIServer) executeRealOTCTokenSwap(order map[string]interface{}, counterparty string) error {
	// Extract order details
	creator, ok := order["creator"].(string)
	if !ok {
		return fmt.Errorf("invalid creator")
	}

	tokenOffered, ok := order["token_offered"].(string)
	if !ok {
		return fmt.Errorf("invalid token offered")
	}

	tokenRequested, ok := order["token_requested"].(string)
	if !ok {
		return fmt.Errorf("invalid token requested")
	}

	amountOffered, ok := order["amount_offered"].(uint64)
	if !ok {
		// Try to convert from float64 (JSON number)
		if amountFloat, ok := order["amount_offered"].(float64); ok {
			amountOffered = uint64(amountFloat)
		} else {
			return fmt.Errorf("invalid amount offered")
		}
	}

	amountRequested, ok := order["amount_requested"].(uint64)
	if !ok {
		// Try to convert from float64 (JSON number)
		if amountFloat, ok := order["amount_requested"].(float64); ok {
			amountRequested = uint64(amountFloat)
		} else {
			return fmt.Errorf("invalid amount requested")
		}
	}

	// Get tokens from registry
	offeredToken, exists := s.blockchain.TokenRegistry[tokenOffered]
	if !exists {
		return fmt.Errorf("offered token %s not found", tokenOffered)
	}

	requestedToken, exists := s.blockchain.TokenRegistry[tokenRequested]
	if !exists {
		return fmt.Errorf("requested token %s not found", tokenRequested)
	}

	// Check counterparty has enough of the requested token
	counterpartyBalance, err := requestedToken.BalanceOf(counterparty)
	if err != nil {
		return fmt.Errorf("failed to check counterparty balance: %v", err)
	}

	if counterpartyBalance < amountRequested {
		return fmt.Errorf("counterparty has insufficient balance: has %d, needs %d", counterpartyBalance, amountRequested)
	}

	// Execute the swap:
	// 1. Transfer offered tokens from OTC contract to counterparty
	err = offeredToken.Transfer("otc_contract", counterparty, amountOffered)
	if err != nil {
		return fmt.Errorf("failed to transfer offered tokens: %v", err)
	}

	// 2. Transfer requested tokens from counterparty to creator
	err = requestedToken.Transfer(counterparty, creator, amountRequested)
	if err != nil {
		// Rollback: transfer offered tokens back to OTC contract
		offeredToken.Transfer(counterparty, "otc_contract", amountOffered)
		return fmt.Errorf("failed to transfer requested tokens: %v", err)
	}

	fmt.Printf("✅ OTC token swap completed: %s gave %d %s, %s gave %d %s\n",
		creator, amountOffered, tokenOffered, counterparty, amountRequested, tokenRequested)

	return nil
}

// releaseOTCTokens releases locked tokens back to the creator when an order is cancelled
func (s *APIServer) releaseOTCTokens(order map[string]interface{}) error {
	// Extract order details
	creator, ok := order["creator"].(string)
	if !ok {
		return fmt.Errorf("invalid creator")
	}

	tokenOffered, ok := order["token_offered"].(string)
	if !ok {
		return fmt.Errorf("invalid token offered")
	}

	amountOffered, ok := order["amount_offered"].(uint64)
	if !ok {
		// Try to convert from float64 (JSON number)
		if amountFloat, ok := order["amount_offered"].(float64); ok {
			amountOffered = uint64(amountFloat)
		} else {
			return fmt.Errorf("invalid amount offered")
		}
	}

	// Get token from registry
	offeredToken, exists := s.blockchain.TokenRegistry[tokenOffered]
	if !exists {
		return fmt.Errorf("offered token %s not found", tokenOffered)
	}

	// Transfer tokens back from OTC contract to creator
	err := offeredToken.Transfer("otc_contract", creator, amountOffered)
	if err != nil {
		return fmt.Errorf("failed to release tokens: %v", err)
	}

	fmt.Printf("✅ Released %d %s tokens back to %s\n", amountOffered, tokenOffered, creator)
	return nil
}

// getTokenNames returns a list of available token names
func (s *APIServer) updateOTCOrderStatus(orderID, status string) {
	if order, exists := otcOrderStore[orderID]; exists {
		order["status"] = status
		order["updated_at"] = time.Now().Unix()

		// Broadcast status update
		s.broadcastOTCEvent("order_updated", order)
	}
}

// Simple event broadcasting system (in production, use WebSockets)
func (s *APIServer) broadcastOTCEvent(eventType string, data map[string]interface{}) {
	fmt.Printf("📡 Broadcasting OTC event: %s\n", eventType)
	// In a real implementation, this would send WebSocket messages to connected clients
	// For now, just log the event
	eventData := map[string]interface{}{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	// Store recent events for polling-based updates
	s.storeRecentOTCEvent(eventData)
}

// Store for recent OTC events
var recentOTCEvents = make([]map[string]interface{}, 0, 100)

func (s *APIServer) storeRecentOTCEvent(event map[string]interface{}) {
	recentOTCEvents = append(recentOTCEvents, event)

	// Keep only last 100 events
	if len(recentOTCEvents) > 100 {
		recentOTCEvents = recentOTCEvents[1:]
	}
}

func (s *APIServer) getRecentOTCEvents() []map[string]interface{} {
	return recentOTCEvents
}

func (s *APIServer) handleOTCEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	events := s.getRecentOTCEvents()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    events,
	})
}

// Slashing API Handlers
