package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/akashic"
)

func (s *APIServer) handleAkashicLineage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.akashic == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "akashic store not initialized"})
		return
	}
	entries, err := s.akashic.All()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	if entries == nil {
		entries = []akashic.LineageEntry{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(entries),
		"entries": entries,
	})
}

// handleAkashicTrace returns the lineage entry for a specific trace_id.
// GET /api/akashic/trace?trace_id=<id>
func (s *APIServer) handleAkashicTrace(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.akashic == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "akashic store not initialized"})
		return
	}
	traceID := r.URL.Query().Get("trace_id")
	if traceID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "trace_id query param required"})
		return
	}
	entry, found := s.akashic.FindByTraceID(traceID)
	if !found {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "trace_id not found in AKASHIC lineage"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"entry":   entry,
	})
}

// handleAkashicReconstruct runs Phase 6 reconstruction proof.
// GET /api/akashic/reconstruct
func (s *APIServer) handleAkashicReconstruct(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.akashic == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "akashic store not initialized"})
		return
	}
	result := s.akashic.Reconstruct()
	if !result.Verified {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": result.Verified,
		"result":  result,
	})
}

// handleTantraChainIntegrity verifies the truth store chain has not been tampered with.
// GET /api/tantra/chain-integrity
func (s *APIServer) handleAkashicReplicate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.akashic == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "akashic not initialized"})
		return
	}
	var entry akashic.LineageEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	if err := s.akashic.Append(entry); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	log.Printf("[KARMACHAIN][RECEIVED] trace=%s tx=%s", entry.TraceID, entry.TxHash)
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true})
}

// handleReplayEquality — Phase 3: distributed replay equality for a trace_id.
// GET /api/replay/equality?trace_id=<id>
