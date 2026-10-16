package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/truthstore"
)

func (s *APIServer) handleTantraStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	check := func(url string) bool {
		client := &http.Client{Timeout: 500 * time.Millisecond}
		resp, err := client.Get(url)
		return err == nil && resp.StatusCode == 200
	}

	pdvExecURL := os.Getenv("PDV_EXECUTION_AGENT_URL")
	pdvValURL := os.Getenv("PDV_VALIDATION_AGENT_URL")
	pdvReplayURL := os.Getenv("PDV_REPLAY_AGENT_URL")
	sarathiURL := os.Getenv("SARATHI_URL")
	nonceCoordURL := os.Getenv("NONCE_COORDINATOR_URL")

	// Derive health URLs from agent URLs
	pdvExecHealth := ""
	if pdvExecURL != "" {
		parts := strings.SplitN(pdvExecURL, "/pdv/", 2)
		if len(parts) == 2 {
			pdvExecHealth = parts[0] + "/health"
		}
	}
	pdvValHealth := ""
	if pdvValURL != "" {
		parts := strings.SplitN(pdvValURL, "/pdv/", 2)
		if len(parts) == 2 {
			pdvValHealth = parts[0] + "/health"
		}
	}
	pdvReplayHealth := ""
	if pdvReplayURL != "" {
		parts := strings.SplitN(pdvReplayURL, "/pdv/", 2)
		if len(parts) == 2 {
			pdvReplayHealth = parts[0] + "/health"
		}
	}

	sarathiHealth := ""
	if sarathiURL != "" {
		parts := strings.SplitN(sarathiURL, "/api/", 2)
		if len(parts) == 2 {
			sarathiHealth = parts[0] + "/health"
		}
	}

	nonceCoordHealth := ""
	if nonceCoordURL != "" {
		nonceCoordHealth = nonceCoordURL + "/health"
	}

	// Bucket + AKASHIC local state
	bucketIntact := false
	akashicVerified := false
	if s.truthStore != nil {
		intact, _, err := s.truthStore.VerifyChain()
		bucketIntact = err == nil && intact
	}
	if s.akashic != nil {
		res := s.akashic.Reconstruct()
		akashicVerified = res.Verified
	}

	components := map[string]interface{}{
		"wallet": map[string]interface{}{
			"description": "Wallet service (services/wallet/)",
			"endpoint":    "POST /api/relay/submit via tantra.go",
			"status":      "INTEGRATED",
		},
		"pdv_execution_agent": map[string]interface{}{
			"url":    pdvExecURL,
			"online": pdvExecHealth != "" && check(pdvExecHealth),
		},
		"pdv_validation_agent": map[string]interface{}{
			"url":    pdvValURL,
			"online": pdvValHealth != "" && check(pdvValHealth),
		},
		"pdv_replay_agent": map[string]interface{}{
			"url":    pdvReplayURL,
			"online": pdvReplayHealth != "" && check(pdvReplayHealth),
		},
		"governance_sarathi": map[string]interface{}{
			"url":    sarathiURL,
			"online": sarathiHealth != "" && check(sarathiHealth),
		},
		"nonce_coordinator": map[string]interface{}{
			"url":    nonceCoordURL,
			"online": nonceCoordHealth != "" && check(nonceCoordHealth),
		},
		"blockchain": map[string]interface{}{
			"block_height": len(s.blockchain.Blocks),
			"online":       true,
		},
		"bucket_truthstore": map[string]interface{}{
			"chain_intact": bucketIntact,
			"online":       s.truthStore != nil,
		},
		"akashic_lineage": map[string]interface{}{
			"verified": akashicVerified,
			"online":   s.akashic != nil,
		},
		"replay_verifier": map[string]interface{}{
			"endpoint": "GET /api/replay/state-root",
			"online":   true,
		},
		"explorer_observability": map[string]interface{}{
			"endpoints": []string{
				"GET /api/tantra/records",
				"GET /api/akashic/lineage",
				"GET /api/nonce/records",
				"GET /api/constitution/declaration",
				"GET /api/convergence/proof",
			},
			"online": true,
		},
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":    true,
		"timestamp":  time.Now().Unix(),
		"node_id":    os.Getenv("NODE_ID"),
		"components": components,
		"flow":       "Wallet→PDV→Governance→Blockchain→Bucket→AKASHIC→Replay→Explorer→MultiNode",
	})
}

// handleWalletKeygen generates a new btcec secp256k1 keypair.
// POST /api/wallet/keygen
// Returns: { private_key, public_key, address } — address = compressed pubkey hex = from field in txs.
// WARNING: private_key is returned once — store it securely, never send it again.
func (s *APIServer) handleTantraVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.truthStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "truth store not initialized"})
		return
	}

	txHash := r.URL.Query().Get("tx_hash")
	traceID := r.URL.Query().Get("trace_id")

	if txHash == "" && traceID != "" {
		// Lookup by trace_id first, then verify by tx_hash
		rec, found := s.truthStore.FindByTraceID(traceID)
		if !found {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "trace_id not found in truth store"})
			return
		}
		txHash = rec.TxHash
	}

	if txHash == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "tx_hash or trace_id query param required"})
		return
	}

	result := s.truthStore.Verify(txHash, s.blockchain)
	if !result.Found || !result.OnChain || !result.HashesMatch {
		w.WriteHeader(http.StatusConflict)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": result.Found && result.OnChain && result.HashesMatch,
		"result":  result,
	})
}

// handleTantraRecords returns all records in the truth store (audit view).
// GET /api/tantra/records
func (s *APIServer) handleTantraRecords(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.truthStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "truth store not initialized"})
		return
	}
	records, err := s.truthStore.All()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	if records == nil {
		records = []truthstore.Record{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"count":   len(records),
		"records": records,
	})
}

// Unified Monitoring Handlers

// handleUnifiedMonitoring provides a comprehensive view of all system components
func (s *APIServer) handleTantraChainIntegrity(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if s.truthStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "truth store not initialized"})
		return
	}
	intact, brokenAt, err := s.truthStore.VerifyChain()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	if !intact {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":   false,
			"intact":    false,
			"broken_at": brokenAt,
			"error":     "truth store chain integrity violation detected — possible tampering",
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"intact":  true,
		"message": "truth store chain is intact — no tampering detected",
	})
}

// handleKSMLSubmit accepts a KSML/CET contract, converts it to schema v1, and routes through runtime.
// POST /api/ksml/submit
