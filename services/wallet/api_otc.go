package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	wallet "github.com/Shivam-Patel-G/blackhole-blockchain/services/wallet/wallet"
)

func handleCreateOTCOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletName      string   `json:"wallet_name"`
		Password        string   `json:"password"`
		TokenOffered    string   `json:"token_offered"`
		AmountOffered   uint64   `json:"amount_offered"`
		TokenRequested  string   `json:"token_requested"`
		AmountRequested uint64   `json:"amount_requested"`
		ExpirationHours int      `json:"expiration_hours"`
		IsMultiSig      bool     `json:"is_multi_sig"`
		RequiredSigs    []string `json:"required_sigs"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request format"}, http.StatusBadRequest)
		return
	}

	// Get user from session
	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication required"}, http.StatusUnauthorized)
		return
	}

	// Get wallet details to get the address
	ctx := context.Background()
	walletDoc, _, _, err := wallet.GetWalletDetails(ctx, user, req.WalletName, req.Password)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to access wallet: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	// Connect to blockchain and create OTC order
	orderData, err := createOTCOrderOnBlockchain(walletDoc.Address, req.TokenOffered, req.TokenRequested,
		req.AmountOffered, req.AmountRequested, req.ExpirationHours, req.IsMultiSig, req.RequiredSigs)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to create OTC order: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	logSuccess("OTC_ORDER_CREATE", fmt.Sprintf("Order %s created by %s", orderData["order_id"], user.Username))

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "OTC order created successfully",
		Data:    orderData,
	}, http.StatusOK)
}

func handleGetOTCOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	// Get user from session to filter orders
	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication required"}, http.StatusUnauthorized)
		return
	}

	// Get user's wallet address (use first wallet for now)
	userAddress := fmt.Sprintf("user_%s", user.Username) // Use username as placeholder address

	// Get orders from blockchain
	orders, err := getOTCOrdersFromBlockchain(userAddress)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to get OTC orders: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, APIResponse{Success: true, Data: orders}, http.StatusOK)
}

func handleGetMarketOTCOrders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	// Get all market orders (from all users) from blockchain
	orders, err := getAllOTCOrdersFromBlockchain()
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to get market orders: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, APIResponse{Success: true, Data: orders}, http.StatusOK)
}

func handleCancelOTCOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID    string `json:"order_id"`
		WalletName string `json:"wallet_name"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request format"}, http.StatusBadRequest)
		return
	}

	// Get user from session
	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication required"}, http.StatusUnauthorized)
		return
	}

	// Get wallet details to get the address
	ctx := context.Background()
	walletDoc, _, _, err := wallet.GetWalletDetails(ctx, user, req.WalletName, req.Password)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to access wallet: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	// Connect to blockchain and cancel OTC order
	success, err := cancelOTCOrderOnBlockchain(req.OrderID, walletDoc.Address)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to cancel OTC order: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	if !success {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Order cancellation failed"}, http.StatusInternalServerError)
		return
	}

	logSuccess("OTC_ORDER_CANCEL", fmt.Sprintf("Order %s cancelled by %s", req.OrderID, user.Username))

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "OTC order cancelled successfully",
		Data: map[string]interface{}{
			"order_id":     req.OrderID,
			"cancelled_by": walletDoc.Address,
			"cancelled_at": time.Now().Unix(),
		},
	}, http.StatusOK)
}

func handleMatchOTCOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID    string `json:"order_id"`
		WalletName string `json:"wallet_name"`
		Password   string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request format"}, http.StatusBadRequest)
		return
	}

	// Get user from session
	user, err := getUserFromSession(r)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Authentication required"}, http.StatusUnauthorized)
		return
	}

	// Get wallet details to get the address
	ctx := context.Background()
	walletDoc, _, _, err := wallet.GetWalletDetails(ctx, user, req.WalletName, req.Password)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to access wallet: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	// Connect to blockchain and match OTC order
	success, err := matchOTCOrderOnBlockchain(req.OrderID, walletDoc.Address)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to match OTC order: " + err.Error()}, http.StatusInternalServerError)
		return
	}

	if !success {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Order matching failed"}, http.StatusInternalServerError)
		return
	}

	logSuccess("OTC_ORDER_MATCH", fmt.Sprintf("Order %s matched by %s", req.OrderID, user.Username))

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "OTC order matched successfully",
		Data: map[string]interface{}{
			"order_id":   req.OrderID,
			"matched_by": walletDoc.Address,
			"matched_at": time.Now().Unix(),
		},
	}, http.StatusOK)
}

// Blockchain OTC Integration Functions
