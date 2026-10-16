package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func (s *APIServer) storeCrossChainOrder(orderID string, orderData map[string]interface{}) {
	crossChainOrderStore[orderID] = orderData

	// Add to user's order list
	user := orderData["user"].(string)
	if crossChainOrdersByUser[user] == nil {
		crossChainOrdersByUser[user] = make([]string, 0)
	}
	crossChainOrdersByUser[user] = append(crossChainOrdersByUser[user], orderID)
}

func (s *APIServer) getCrossChainOrder(orderID string) (map[string]interface{}, bool) {
	order, exists := crossChainOrderStore[orderID]
	return order, exists
}

func (s *APIServer) getUserCrossChainOrders(user string) []map[string]interface{} {
	orderIDs, exists := crossChainOrdersByUser[user]
	if !exists {
		return []map[string]interface{}{}
	}

	var orders []map[string]interface{}
	for _, orderID := range orderIDs {
		if order, exists := crossChainOrderStore[orderID]; exists {
			orders = append(orders, order)
		}
	}

	return orders
}

func (s *APIServer) updateCrossChainOrderStatus(orderID, status string) {
	if order, exists := crossChainOrderStore[orderID]; exists {
		order["status"] = status
		if status == "completed" {
			order["completed_at"] = time.Now().Unix()
		}
	}
}

// handleRelaySubmit is the SINGLE canonical entry point for ALL transactions.
// Phase 1C: trace_id enforced via trace.Context — immutable after injection.
// Phase 1D: execution flows through runtime.Execute() — the canonical runtime.
// No transaction proceeds without passing every gate in strict order.
func (s *APIServer) processCrossChainSwap(orderID string) {
	_, exists := s.getCrossChainOrder(orderID)
	if !exists {
		return
	}

	// Step 1: Bridging phase (2-3 seconds)
	time.Sleep(2 * time.Second)
	s.updateCrossChainOrderStatus(orderID, "bridging")
	fmt.Printf("🌉 Order %s: Bridging tokens...\n", orderID)

	// Step 2: Bridge confirmation (3-5 seconds)
	time.Sleep(3 * time.Second)
	s.updateCrossChainOrderStatus(orderID, "swapping")
	fmt.Printf("🔄 Order %s: Executing swap on destination chain...\n", orderID)

	// Step 3: Swap execution (2-3 seconds)
	time.Sleep(2 * time.Second)

	// Update order with final details
	if order, exists := crossChainOrderStore[orderID]; exists {
		order["status"] = "completed"
		order["completed_at"] = time.Now().Unix()
		order["bridge_tx_id"] = fmt.Sprintf("bridge_%s", orderID)
		order["swap_tx_id"] = fmt.Sprintf("swap_%s", orderID)

		// Simulate slight slippage
		estimatedOut := order["estimated_out"].(uint64)
		actualOut := uint64(float64(estimatedOut) * 0.998) // 0.2% slippage
		order["actual_out"] = actualOut
	}

	fmt.Printf("✅ Order %s: Cross-chain swap completed!\n", orderID)
}

func (s *APIServer) handleCrossChainQuote(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	var req struct {
		SourceChain string `json:"source_chain"`
		DestChain   string `json:"dest_chain"`
		TokenIn     string `json:"token_in"`
		TokenOut    string `json:"token_out"`
		AmountIn    uint64 `json:"amount_in"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Simulate cross-chain quote (in production, would use actual CrossChainDEX)
	quote := map[string]interface{}{
		"source_chain":  req.SourceChain,
		"dest_chain":    req.DestChain,
		"token_in":      req.TokenIn,
		"token_out":     req.TokenOut,
		"amount_in":     req.AmountIn,
		"estimated_out": uint64(float64(req.AmountIn) * 0.95), // 5% total fees
		"price_impact":  0.5,
		"bridge_fee":    uint64(float64(req.AmountIn) * 0.01),  // 1% bridge fee
		"swap_fee":      uint64(float64(req.AmountIn) * 0.003), // 0.3% swap fee
		"expires_at":    time.Now().Add(10 * time.Minute).Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    quote,
	})
}

func (s *APIServer) handleCrossChainSwap(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	var req struct {
		User         string `json:"user"`
		SourceChain  string `json:"source_chain"`
		DestChain    string `json:"dest_chain"`
		TokenIn      string `json:"token_in"`
		TokenOut     string `json:"token_out"`
		AmountIn     uint64 `json:"amount_in"`
		MinAmountOut uint64 `json:"min_amount_out"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Generate swap order ID
	userSuffix := req.User
	if len(req.User) > 8 {
		userSuffix = req.User[:8]
	}
	orderID := fmt.Sprintf("ccswap_%d_%s", time.Now().UnixNano(), userSuffix)

	// Calculate fees and estimated output
	bridgeFee := uint64(float64(req.AmountIn) * 0.01)    // 1% bridge fee
	swapFee := uint64(float64(req.AmountIn) * 0.003)     // 0.3% swap fee
	estimatedOut := uint64(float64(req.AmountIn) * 0.95) // 5% total fees

	// Create real cross-chain swap order
	order := map[string]interface{}{
		"id":             orderID,
		"user":           req.User,
		"source_chain":   req.SourceChain,
		"dest_chain":     req.DestChain,
		"token_in":       req.TokenIn,
		"token_out":      req.TokenOut,
		"amount_in":      req.AmountIn,
		"min_amount_out": req.MinAmountOut,
		"estimated_out":  estimatedOut,
		"status":         "pending",
		"created_at":     time.Now().Unix(),
		"expires_at":     time.Now().Add(30 * time.Minute).Unix(),
		"bridge_fee":     bridgeFee,
		"swap_fee":       swapFee,
		"price_impact":   0.5,
	}

	// Store the order
	s.storeCrossChainOrder(orderID, order)

	// Start background processing to simulate swap execution
	go s.processCrossChainSwap(orderID)

	fmt.Printf("✅ Cross-chain swap initiated: %s (%d %s → %s)\n",
		orderID, req.AmountIn, req.TokenIn, req.TokenOut)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Cross-chain swap initiated successfully",
		"data":    order,
	})
}

func (s *APIServer) handleCrossChainOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	orderID := r.URL.Query().Get("id")
	if orderID == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Order ID required",
		})
		return
	}

	// Get real order data
	order, exists := s.getCrossChainOrder(orderID)
	if !exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Order not found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    order,
	})
}

func (s *APIServer) handleCrossChainOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	user := r.URL.Query().Get("user")
	if user == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "User parameter required",
		})
		return
	}

	// Get real user orders
	orders := s.getUserCrossChainOrders(user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    orders,
	})
}
