package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/escrow"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/runtime"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
)

func (s *APIServer) testEscrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Action      string `json:"action"`
		Sender      string `json:"sender"`
		Receiver    string `json:"receiver"`
		Arbitrator  string `json:"arbitrator"`
		TokenSymbol string `json:"token_symbol"`
		Amount      uint64 `json:"amount"`
		EscrowID    string `json:"escrow_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🔧 DEV MODE: Executing Escrow function '%s'\n", req.Action)

	contract := &schema.TxContract{
		SchemaVersion: schema.CurrentVersion,
		Type:          "escrow_" + req.Action,
		From:          req.Sender,
		To:            req.Receiver,
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
		"message": fmt.Sprintf("Escrow %s executed successfully", req.Action),
		"data": map[string]interface{}{
			"action": req.Action,
			"status": "executed",
			"note":   "Transaction successfully processed by the runtime",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleEscrowRequest handles real escrow operations from the blockchain client
func (s *APIServer) handleEscrowRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	action, ok := req["action"].(string)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Missing or invalid action",
		})
		return
	}

	// Log the escrow request
	fmt.Printf("🔒 ESCROW REQUEST: %s\n", action)

	// Check if escrow manager is initialized
	if s.escrowManager == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Escrow manager not initialized",
		})
		return
	}

	var result map[string]interface{}
	var err error

	switch action {
	case "create_escrow":
		result, err = s.handleCreateEscrow(req)
	case "confirm_escrow":
		result, err = s.handleConfirmEscrow(req)
	case "release_escrow":
		result, err = s.handleReleaseEscrow(req)
	case "cancel_escrow":
		result, err = s.handleCancelEscrow(req)
	case "get_escrow":
		result, err = s.handleGetEscrow(req)
	case "get_user_escrows":
		result, err = s.handleGetUserEscrows(req)
	default:
		err = fmt.Errorf("unknown action: %s", action)
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleCreateEscrow handles escrow creation requests
func (s *APIServer) handleCreateEscrow(req map[string]interface{}) (map[string]interface{}, error) {
	sender, ok := req["sender"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid sender")
	}

	receiver, ok := req["receiver"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid receiver")
	}

	tokenSymbol, ok := req["token_symbol"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid token_symbol")
	}

	amount, ok := req["amount"].(float64)
	if !ok {
		return nil, fmt.Errorf("missing or invalid amount")
	}

	expirationHours, ok := req["expiration_hours"].(float64)
	if !ok {
		expirationHours = 24 // Default to 24 hours
	}

	arbitrator, _ := req["arbitrator"].(string)   // Optional
	description, _ := req["description"].(string) // Optional

	// Create escrow using the real escrow manager
	escrowManager := s.escrowManager.(*escrow.EscrowManager)

	contract, err := escrowManager.CreateEscrow(
		sender,
		receiver,
		arbitrator,
		tokenSymbol,
		uint64(amount),
		int(expirationHours),
		description,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success":   true,
		"escrow_id": contract.ID,
		"message":   fmt.Sprintf("Escrow created successfully: %s", contract.ID),
		"data": map[string]interface{}{
			"id":            contract.ID,
			"sender":        contract.Sender,
			"receiver":      contract.Receiver,
			"arbitrator":    contract.Arbitrator,
			"token_symbol":  contract.TokenSymbol,
			"amount":        contract.Amount,
			"status":        contract.Status.String(),
			"created_at":    contract.CreatedAt,
			"expires_at":    contract.ExpiresAt,
			"required_sigs": contract.RequiredSigs,
			"description":   contract.Description,
		},
	}, nil
}

// handleConfirmEscrow handles escrow confirmation requests
func (s *APIServer) handleConfirmEscrow(req map[string]interface{}) (map[string]interface{}, error) {
	escrowID, ok := req["escrow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid escrow_id")
	}

	confirmer, ok := req["confirmer"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid confirmer")
	}

	// Use the real escrow manager
	escrowManager := s.escrowManager.(*escrow.EscrowManager)

	err := escrowManager.ConfirmEscrow(escrowID, confirmer)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Escrow %s confirmed successfully", escrowID),
		"data": map[string]interface{}{
			"escrow_id": escrowID,
			"confirmer": confirmer,
			"status":    "confirmed",
		},
	}, nil
}

// handleReleaseEscrow handles escrow release requests
func (s *APIServer) handleReleaseEscrow(req map[string]interface{}) (map[string]interface{}, error) {
	escrowID, ok := req["escrow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid escrow_id")
	}

	releaser, ok := req["releaser"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid releaser")
	}

	// Use the real escrow manager
	escrowManager := s.escrowManager.(*escrow.EscrowManager)

	err := escrowManager.ReleaseEscrow(escrowID, releaser)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Escrow %s released successfully", escrowID),
		"data": map[string]interface{}{
			"escrow_id": escrowID,
			"releaser":  releaser,
			"status":    "released",
		},
	}, nil
}

// handleCancelEscrow handles escrow cancellation requests
func (s *APIServer) handleCancelEscrow(req map[string]interface{}) (map[string]interface{}, error) {
	escrowID, ok := req["escrow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid escrow_id")
	}

	canceller, ok := req["canceller"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid canceller")
	}

	// Use the real escrow manager
	escrowManager := s.escrowManager.(*escrow.EscrowManager)

	err := escrowManager.CancelEscrow(escrowID, canceller)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Escrow %s cancelled successfully", escrowID),
		"data": map[string]interface{}{
			"escrow_id": escrowID,
			"canceller": canceller,
			"status":    "cancelled",
		},
	}, nil
}

// handleGetEscrow handles getting escrow details
func (s *APIServer) handleGetEscrow(req map[string]interface{}) (map[string]interface{}, error) {
	escrowID, ok := req["escrow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid escrow_id")
	}

	// Use the real escrow manager
	escrowManager := s.escrowManager.(*escrow.EscrowManager)

	contract, exists := escrowManager.Contracts[escrowID]
	if !exists {
		return nil, fmt.Errorf("escrow %s not found", escrowID)
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Escrow %s details retrieved", escrowID),
		"data": map[string]interface{}{
			"id":            contract.ID,
			"sender":        contract.Sender,
			"receiver":      contract.Receiver,
			"arbitrator":    contract.Arbitrator,
			"token_symbol":  contract.TokenSymbol,
			"amount":        contract.Amount,
			"status":        contract.Status.String(),
			"created_at":    contract.CreatedAt,
			"expires_at":    contract.ExpiresAt,
			"required_sigs": contract.RequiredSigs,
			"description":   contract.Description,
		},
	}, nil
}

// handleGetUserEscrows handles getting all escrows for a user
func (s *APIServer) handleGetUserEscrows(req map[string]interface{}) (map[string]interface{}, error) {
	userAddress, ok := req["user_address"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid user_address")
	}

	// Use the real escrow manager
	escrowManager := s.escrowManager.(*escrow.EscrowManager)

	var userEscrows []interface{}

	// Filter escrows where user is involved
	for _, contract := range escrowManager.Contracts {
		// Check if user is involved in this escrow
		if contract.Sender == userAddress || contract.Receiver == userAddress || contract.Arbitrator == userAddress {
			escrowData := map[string]interface{}{
				"id":            contract.ID,
				"sender":        contract.Sender,
				"receiver":      contract.Receiver,
				"arbitrator":    contract.Arbitrator,
				"token_symbol":  contract.TokenSymbol,
				"amount":        contract.Amount,
				"status":        contract.Status.String(),
				"created_at":    contract.CreatedAt,
				"expires_at":    contract.ExpiresAt,
				"required_sigs": contract.RequiredSigs,
				"description":   contract.Description,
			}
			userEscrows = append(userEscrows, escrowData)
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Found %d escrows for user %s", len(userEscrows), userAddress),
		"data": map[string]interface{}{
			"escrows": userEscrows,
			"count":   len(userEscrows),
		},
	}, nil
}

// handleBalanceQuery handles dedicated balance query requests
