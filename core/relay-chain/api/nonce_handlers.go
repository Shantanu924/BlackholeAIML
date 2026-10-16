package api

import (
	"encoding/json"
	"net/http"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/noncestore"
)

func (s *APIServer) handleNonceLookup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	address := r.URL.Query().Get("address")
	if address == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "address query param required"})
		return
	}
	if s.nonceStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "nonce store not initialized"})
		return
	}
	latest := s.nonceStore.Latest(address)
	records, _ := s.nonceStore.AddressRecords(address)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"address":      address,
		"latest_nonce": latest,
		"next_nonce":   latest + 1,
		"record_count": len(records),
		"records":      records,
	})
}

// handleNonceRecords returns all nonce records (full lineage).
// GET /api/nonce/records
func (s *APIServer) handleNonceRecords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.nonceStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "nonce store not initialized"})
		return
	}
	records, err := s.nonceStore.AllRecords()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	if records == nil {
		records = []noncestore.NonceRecord{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(records),
		"records": records,
	})
}

// handleSigVerify verifies a wallet signature against the canonical payload hash.
// POST /api/sig/verify
// Body: same schema v1 contract — verifies the signature field against the from address.
