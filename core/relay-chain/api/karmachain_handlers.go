package api

import (
	"encoding/json"
	"net/http"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/karmachain"
)

func (s *APIServer) handleKarmaChainConsistency(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	result := karmachain.VerifyConsistency()
	if !result.Consistent {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": result.Consistent,
		"result":  result,
	})
}

// handleKarmaChainReconstruct pulls lineage from the first reachable KarmaChain node.
// GET /api/karmachain/reconstruct
func (s *APIServer) handleKarmaChainReconstruct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	entries, stateRoot, err := karmachain.Reconstruct()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"entry_count": len(entries),
		"state_root":  stateRoot,
		"entries":     entries,
	})
}

// handleAkashicReplicate receives a replicated lineage entry from another node.
// POST /api/akashic/replicate
