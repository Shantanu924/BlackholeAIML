// Package fraud implements the real fraud detection service (Phase 2).
// Runs on port 9090. Receives transactions from ValidationAgent,
// applies rule-based fraud detection, returns allow/block decision.
package fraud

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type FraudRequest struct {
	TraceID   string `json:"trace_id"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	TokenID   string `json:"token_id"`
	Timestamp int64  `json:"timestamp"`
}

type FraudResponse struct {
	Decision  string `json:"decision"`
	Reason    string `json:"reason,omitempty"`
	TraceID   string `json:"trace_id"`
	RiskScore int    `json:"risk_score"`
}

type txRecord struct {
	count    int
	totalAmt uint64
	lastSeen time.Time
}

type FraudService struct {
	mu      sync.Mutex
	history map[string]*txRecord
	blocked map[string]string
}

func NewFraudService() *FraudService {
	fs := &FraudService{
		history: make(map[string]*txRecord),
		blocked: make(map[string]string),
	}
	fs.blocked["bad-actor"] = "known fraudulent address"
	fs.blocked["scammer"] = "flagged by manual review"
	return fs
}

func (fs *FraudService) evaluate(req FraudRequest) (string, int, string) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Rule 1 — permanently blocked address
	if reason, blocked := fs.blocked[req.From]; blocked {
		return "block", 100, "sender is permanently blocked: " + reason
	}
	if reason, blocked := fs.blocked[req.To]; blocked {
		return "block", 100, "recipient is permanently blocked: " + reason
	}
	// Rule 2 — empty addresses
	if req.From == "" || req.To == "" {
		return "block", 90, "missing sender or recipient address"
	}
	// Rule 3 — self transfer
	if req.From == req.To {
		return "block", 80, "self-transfer detected"
	}
	// Rule 4 — zero amount
	if req.Amount == 0 {
		return "block", 70, "zero amount transaction"
	}
	// Rule 5 — whale alert
	if req.Amount > 500000 {
		return "block", 85, fmt.Sprintf("transaction amount %d exceeds single-tx limit of 500000", req.Amount)
	}
	// Rule 6 — velocity check
	rec, exists := fs.history[req.From]
	now := time.Now()
	if exists {
		if now.Sub(rec.lastSeen) < 60*time.Second {
			rec.count++
			rec.totalAmt += req.Amount
			if rec.count > 10 {
				return "block", 90, fmt.Sprintf("velocity limit exceeded: %d txs in 60s from %s", rec.count, req.From)
			}
			// Rule 7 — cumulative amount
			if rec.totalAmt > 1000000 {
				return "block", 88, fmt.Sprintf("cumulative amount %d in 60s exceeds limit from %s", rec.totalAmt, req.From)
			}
		} else {
			rec.count = 1
			rec.totalAmt = req.Amount
			rec.lastSeen = now
		}
	} else {
		fs.history[req.From] = &txRecord{count: 1, totalAmt: req.Amount, lastSeen: now}
	}
	// Rule 8 — stale timestamp
	if req.Timestamp > 0 {
		age := time.Now().Unix() - req.Timestamp
		if age > 600 {
			return "block", 75, fmt.Sprintf("stale transaction: %d seconds old", age)
		}
	}

	riskScore := 0
	if req.Amount > 100000 {
		riskScore = 20
	}
	return "allow", riskScore, "all fraud checks passed"
}

func (fs *FraudService) handleCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req FraudRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(FraudResponse{Decision: "block", Reason: "invalid request: " + err.Error(), TraceID: req.TraceID})
		return
	}
	decision, riskScore, reason := fs.evaluate(req)
	resp := FraudResponse{Decision: decision, Reason: reason, TraceID: req.TraceID, RiskScore: riskScore}
	log.Printf("[FraudService] trace=%s from=%s to=%s amount=%d decision=%s risk=%d",
		req.TraceID, req.From, req.To, req.Amount, decision, riskScore)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (fs *FraudService) handleStatus(w http.ResponseWriter, r *http.Request) {
	fs.mu.Lock()
	tracked := len(fs.history)
	blocked := len(fs.blocked)
	fs.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":            "operational",
		"port":              9090,
		"tracked_addresses": tracked,
		"blocked_addresses": blocked,
		"rules": []string{
			"permanently_blocked_address", "empty_address", "self_transfer",
			"zero_amount", "single_tx_limit_500000", "velocity_10_per_60s",
			"cumulative_1000000_per_60s", "stale_timestamp_600s",
		},
	})
}

func (fs *FraudService) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Address string `json:"address"`
		Reason  string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Address == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	fs.mu.Lock()
	fs.blocked[req.Address] = req.Reason
	fs.mu.Unlock()
	log.Printf("[FraudService] manually blocked address=%s reason=%s", req.Address, req.Reason)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": fmt.Sprintf("address %s blocked", req.Address)})
}

func (fs *FraudService) handleIdentity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service":       "vajra-fraud-detection",
		"trust_layer":   "Cyber Defence",
		"port":          9090,
		"authority":     []string{"FRAUD_DETECTION", "VELOCITY_CHECK", "ADDRESS_BLOCKING", "GOVERNANCE_DECISION"},
		"non_authority": []string{"PDV_EQUALITY", "BLOCKCHAIN_WRITE", "NONCE_GOVERNANCE"},
		"endpoints": []string{
			"POST /api/fraud/check",
			"GET  /api/fraud/status",
			"POST /api/fraud/block",
			"GET  /health",
			"GET  /identity",
		},
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

func (fs *FraudService) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/fraud/check", fs.handleCheck)
	mux.HandleFunc("/api/fraud/status", fs.handleStatus)
	mux.HandleFunc("/api/fraud/block", fs.handleBlock)
	mux.HandleFunc("/identity", fs.handleIdentity)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	fmt.Println("🛡️  Fraud Detection Service starting on port 9090")
	fmt.Println("   POST /api/fraud/check  — evaluate transaction")
	fmt.Println("   GET  /api/fraud/status — service health + stats")
	fmt.Println("   POST /api/fraud/block  — manually block an address")
	log.Fatal(http.ListenAndServe(":9090", mux))
}
