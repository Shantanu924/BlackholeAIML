package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/chain"
)

func (s *APIServer) handleSlashingEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	events := s.blockchain.SlashingManager.GetSlashingEvents()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    events,
	})
}

func (s *APIServer) handleSlashingReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	var req struct {
		Validator   string `json:"validator"`
		Condition   int    `json:"condition"`
		Evidence    string `json:"evidence"`
		BlockHeight uint64 `json:"block_height"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("🚨 Slashing violation reported for validator %s\n", req.Validator)

	event, err := s.blockchain.SlashingManager.ReportViolation(
		req.Validator,
		chain.SlashingCondition(req.Condition),
		req.Evidence,
		req.BlockHeight,
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
		"message": "Slashing violation reported successfully",
		"data":    event,
	})
}

func (s *APIServer) handleSlashingExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	var req struct {
		EventID string `json:"event_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	fmt.Printf("⚡ Executing slashing event %s\n", req.EventID)

	err := s.blockchain.SlashingManager.ExecuteSlashing(req.EventID)
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
		"message": "Slashing executed successfully",
	})
}

func (s *APIServer) handleValidatorStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Method not allowed",
		})
		return
	}

	validator := r.URL.Query().Get("validator")
	if validator == "" {
		// Return all validator statuses
		validators := s.blockchain.StakeLedger.GetAllStakes()
		validatorStatuses := make(map[string]interface{})

		for validatorAddr := range validators {
			validatorStatuses[validatorAddr] = map[string]interface{}{
				"stake":   s.blockchain.StakeLedger.GetStake(validatorAddr),
				"strikes": s.blockchain.SlashingManager.GetValidatorStrikes(validatorAddr),
				"jailed":  s.blockchain.SlashingManager.IsValidatorJailed(validatorAddr),
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"data":    validatorStatuses,
		})
		return
	}

	// Return specific validator status
	status := map[string]interface{}{
		"validator": validator,
		"stake":     s.blockchain.StakeLedger.GetStake(validator),
		"strikes":   s.blockchain.SlashingManager.GetValidatorStrikes(validator),
		"jailed":    s.blockchain.SlashingManager.IsValidatorJailed(validator),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    status,
	})
}

// Cross-Chain DEX API Handlers
