package api

import (
	"encoding/json"
	"net/http"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/constitution"
)

func (s *APIServer) handleConstitutionDeclaration(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	decl := constitution.Declare()
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "declaration": decl})
}

// handleVerifyBoundary — Phase 5: verify a specific constitutional boundary.
// GET /api/constitution/verify-boundary?name=<boundary_name>
