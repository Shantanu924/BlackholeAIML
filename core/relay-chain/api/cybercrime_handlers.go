package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Cybercrime data structures
type Report struct {
	ReporterID string    `json:"reporter_id"`
	Reason     string    `json:"reason"`
	Severity   int       `json:"severity"`
	TxHash     string    `json:"tx_hash"`
	Timestamp  time.Time `json:"timestamp"`
}

type WalletStatus struct {
	Address            string    `json:"address"`
	Status             string    `json:"status"` // "active", "frozen"
	ReportCount        int       `json:"report_count"`
	TotalSeverityScore int       `json:"total_severity_score"`
	LastReportedAt     time.Time `json:"last_reported_at,omitempty"`
	FrozenAt           time.Time `json:"frozen_at,omitempty"`
	FrozenBy           string    `json:"frozen_by,omitempty"`
	Reports            []Report  `json:"reports"`
}

// In-memory store for the Alpha POC
var (
	cybercrimeMu    sync.RWMutex
	walletReports   = make(map[string][]Report)
	frozenWallets   = make(map[string]struct {
		FrozenAt time.Time
		FrozenBy string
	})
	totalReports    = 0
	autoFreezeLevel = 15 // If severity sum >= 15, auto-freeze
)

// handleCybercrimeReport saves a new abuse report
func (s *APIServer) handleCybercrimeReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletAddress string `json:"wallet_address"`
		ReporterID    string `json:"reporter_id"`
		Reason        string `json:"reason"`
		Severity      int    `json:"severity"`
		TxHash        string `json:"tx_hash"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid payload"}`, http.StatusBadRequest)
		return
	}

	req.WalletAddress = strings.ToLower(req.WalletAddress)

	cybercrimeMu.Lock()
	defer cybercrimeMu.Unlock()

	report := Report{
		ReporterID: req.ReporterID,
		Reason:     req.Reason,
		Severity:   req.Severity,
		TxHash:     req.TxHash,
		Timestamp:  time.Now(),
	}

	walletReports[req.WalletAddress] = append(walletReports[req.WalletAddress], report)
	totalReports++

	// Check for auto-freeze
	totalSeverity := 0
	for _, r := range walletReports[req.WalletAddress] {
		totalSeverity += r.Severity
	}

	if totalSeverity >= autoFreezeLevel {
		frozenWallets[req.WalletAddress] = struct {
			FrozenAt time.Time
			FrozenBy string
		}{
			FrozenAt: time.Now(),
			FrozenBy: "SYSTEM_AUTO_FREEZE",
		}
		fmt.Printf("🧊 [CYBERCRIME] Auto-froze wallet %s due to high severity score (%d)\n", req.WalletAddress, totalSeverity)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleCybercrimeStatus retrieves status and reports for a wallet
func (s *APIServer) handleCybercrimeStatus(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 5 {
		http.Error(w, `{"error": "Address required"}`, http.StatusBadRequest)
		return
	}
	address := strings.ToLower(parts[4]) // /api/cybercrime/status/ADDRESS

	cybercrimeMu.RLock()
	defer cybercrimeMu.RUnlock()

	reports := walletReports[address]
	if reports == nil {
		reports = []Report{}
	}

	totalSeverity := 0
	var lastReport time.Time
	for _, r := range reports {
		totalSeverity += r.Severity
		if r.Timestamp.After(lastReport) {
			lastReport = r.Timestamp
		}
	}

	status := "active"
	var frozenAt time.Time
	var frozenBy string
	if freezeData, isFrozen := frozenWallets[address]; isFrozen {
		status = "frozen"
		frozenAt = freezeData.FrozenAt
		frozenBy = freezeData.FrozenBy
	}

	result := WalletStatus{
		Address:            address,
		Status:             status,
		ReportCount:        len(reports),
		TotalSeverityScore: totalSeverity,
		LastReportedAt:     lastReport,
		FrozenAt:           frozenAt,
		FrozenBy:           frozenBy,
		Reports:            reports,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleCybercrimeFreeze manually freezes a wallet
func (s *APIServer) handleCybercrimeFreeze(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletAddress string `json:"wallet_address"`
		AdminAddress  string `json:"admin_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid payload"}`, http.StatusBadRequest)
		return
	}

	req.WalletAddress = strings.ToLower(req.WalletAddress)

	cybercrimeMu.Lock()
	defer cybercrimeMu.Unlock()

	frozenWallets[req.WalletAddress] = struct {
		FrozenAt time.Time
		FrozenBy string
	}{
		FrozenAt: time.Now(),
		FrozenBy: req.AdminAddress,
	}

	fmt.Printf("🧊 [CYBERCRIME] Admin %s froze wallet %s\n", req.AdminAddress, req.WalletAddress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleCybercrimeUnfreeze manually unfreezes a wallet
func (s *APIServer) handleCybercrimeUnfreeze(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		WalletAddress string `json:"wallet_address"`
		AdminAddress  string `json:"admin_address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error": "Invalid payload"}`, http.StatusBadRequest)
		return
	}

	req.WalletAddress = strings.ToLower(req.WalletAddress)

	cybercrimeMu.Lock()
	defer cybercrimeMu.Unlock()

	delete(frozenWallets, req.WalletAddress)

	fmt.Printf("🔓 [CYBERCRIME] Admin %s unfroze wallet %s\n", req.AdminAddress, req.WalletAddress)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleCybercrimeStats provides system-wide metrics
func (s *APIServer) handleCybercrimeStats(w http.ResponseWriter, r *http.Request) {
	cybercrimeMu.RLock()
	defer cybercrimeMu.RUnlock()

	stats := map[string]interface{}{
		"total_reports":         totalReports,
		"total_frozen":          len(frozenWallets),
		"total_unfrozen":        0, // Simplified for POC
		"auto_freeze_threshold": autoFreezeLevel,
		"total_wallets":         len(walletReports),
		"admin_count":           1,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
