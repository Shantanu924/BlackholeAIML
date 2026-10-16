package api

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/diagnostics"
)

// handleDiagnostics serves deep node and subsystem diagnostics.
// Endpoint: GET /api/diagnostics, GET /api/v1/diagnostics
func (s *APIServer) handleDiagnostics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	nodeID := os.Getenv("NODE_ID")
	diag := diagnostics.RunDiagnostics(s.blockchain, nodeID)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(diag)
}
