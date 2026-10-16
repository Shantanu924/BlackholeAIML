// Package pdv implements distributed multi-node PDV (Proof of Deterministic Validity).
//
// In single-node mode all three agents share process locality — this is the
// architectural gap that prevents true distributed sovereign execution.
//
// This package solves it by:
//   1. Running each agent as an independent HTTP endpoint (separate goroutines,
//      separate ports, or separate processes via env config).
//   2. Sending the same canonical payload to all three agents independently.
//   3. Comparing the returned hashes over the network.
//   4. Any disagreement → DISTRIBUTED_PDV_REJECT — hard fail.
//
// Agent endpoints are configured via environment variables:
//   PDV_EXECUTION_AGENT_URL   (default: http://localhost:9101/pdv/execute)
//   PDV_VALIDATION_AGENT_URL  (default: http://localhost:9102/pdv/validate)
//   PDV_REPLAY_AGENT_URL      (default: http://localhost:9103/pdv/replay)
//
// When all three URLs point to the same process (default), behaviour is
// identical to the existing single-node PDV — no regression.
// When they point to separate processes/machines, true distributed PDV is achieved.
package pdv

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// AgentRequest is the payload sent to each PDV agent.
type AgentRequest struct {
	TraceID   string `json:"trace_id"`
	Type      string `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"`
	Amount    uint64 `json:"amount"`
	TokenID   string `json:"token_id"`
	Fee       uint64 `json:"fee"`
	Nonce     uint64 `json:"nonce"`
	Signature string `json:"signature"`
}

// AgentResponse is returned by each PDV agent.
type AgentResponse struct {
	Hash    string `json:"hash"`
	TraceID string `json:"trace_id"`
	Agent   string `json:"agent"`
	Error   string `json:"error,omitempty"`
}

// DistributedResult is the result of a distributed PDV check.
type DistributedResult struct {
	Agreed          bool   `json:"agreed"`
	ExecutionHash   string `json:"execution_hash"`
	ValidationHash  string `json:"validation_hash"`
	ReplayHash      string `json:"replay_hash"`
	TraceID         string `json:"trace_id"`
	RejectionReason string `json:"rejection_reason,omitempty"`
}

// agentURLs returns the configured agent endpoints.
func agentURLs() (exec, val, replay string) {
	exec = os.Getenv("PDV_EXECUTION_AGENT_URL")
	if exec == "" {
		exec = "http://localhost:9101/pdv/execute"
	}
	val = os.Getenv("PDV_VALIDATION_AGENT_URL")
	if val == "" {
		val = "http://localhost:9102/pdv/validate"
	}
	replay = os.Getenv("PDV_REPLAY_AGENT_URL")
	if replay == "" {
		replay = "http://localhost:9103/pdv/replay"
	}
	return
}

// DeterministicHash computes SHA-256 over the canonical agent request.
func DeterministicHash(req AgentRequest) string {
	return deterministicHash(req)
}

func deterministicHash(req AgentRequest) string {
	type zone struct {
		TraceID   string `json:"trace_id"`
		Type      string `json:"type"`
		From      string `json:"from"`
		To        string `json:"to"`
		Amount    uint64 `json:"amount"`
		TokenID   string `json:"token_id"`
		Fee       uint64 `json:"fee"`
		Nonce     uint64 `json:"nonce"`
		Signature string `json:"signature"`
	}
	z := zone{
		TraceID:   req.TraceID,
		Type:      req.Type,
		From:      req.From,
		To:        req.To,
		Amount:    req.Amount,
		TokenID:   req.TokenID,
		Fee:       req.Fee,
		Nonce:     req.Nonce,
		Signature: req.Signature,
	}
	data, _ := json.Marshal(z)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// strictMode returns true when PDV strict mode is active.
// DEFAULT: true (production-safe). Set PDV_STRICT_MODE=false to disable.
// This inverts the previous behavior where strict mode required explicit opt-in.
func strictMode() bool {
	return os.Getenv("PDV_STRICT_MODE") != "false"
}

// callAgent sends the request to a remote PDV agent and returns its hash.
// Strict mode: unreachable agent → hard error (no fallback).
// Non-strict mode: unreachable agent → local computation fallback (dev only).
func callAgent(url, agentName string, req AgentRequest) (string, error) {
	body, _ := json.Marshal(req)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		if strictMode() {
			// STRICT MODE: agent unreachable = hard fail.
			// This is the production-safe path — no silent local fallback.
			log.Printf("[PDV][%s][STRICT] unreachable — HARD FAIL (PDV_STRICT_MODE=true)", agentName)
			return "", fmt.Errorf("PDV_STRICT: agent %s unreachable at %s: %v", agentName, url, err)
		}
		// NON-STRICT: fall back to local computation (dev/single-node only).
		log.Printf("[PDV][%s] unreachable (%v) — computing locally (set PDV_STRICT_MODE=true for production)", agentName, err)
		return deterministicHash(req), nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var ar AgentResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		if strictMode() {
			return "", fmt.Errorf("PDV_STRICT: agent %s bad response: %v", agentName, err)
		}
		log.Printf("[PDV][%s] bad response — computing locally", agentName)
		return deterministicHash(req), nil
	}
	if ar.Error != "" {
		return "", fmt.Errorf("agent %s error: %s", agentName, ar.Error)
	}
	return ar.Hash, nil
}

// Check runs distributed PDV — sends the same payload to all three agents
// independently (in parallel), then compares the returned hashes.
// All three must agree. Any disagreement = DISTRIBUTED_PDV_REJECT.
func Check(req AgentRequest) DistributedResult {
	execURL, valURL, replayURL := agentURLs()

	type result struct {
		name string
		hash string
		err  error
	}

	results := make([]result, 3)
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		h, err := callAgent(execURL, "ExecutionAgent", req)
		results[0] = result{"ExecutionAgent", h, err}
	}()
	go func() {
		defer wg.Done()
		h, err := callAgent(valURL, "ValidationAgent", req)
		results[1] = result{"ValidationAgent", h, err}
	}()
	go func() {
		defer wg.Done()
		h, err := callAgent(replayURL, "ReplayAgent", req)
		results[2] = result{"ReplayAgent", h, err}
	}()
	wg.Wait()

	// Check for agent errors.
	for _, r := range results {
		if r.err != nil {
			log.Printf("[PDV][DISTRIBUTED_REJECT] agent=%s trace=%s err=%v",
				r.name, req.TraceID, r.err)
			return DistributedResult{
				Agreed:          false,
				TraceID:         req.TraceID,
				RejectionReason: fmt.Sprintf("agent %s returned error: %v", r.name, r.err),
			}
		}
	}

	execHash := results[0].hash
	valHash := results[1].hash
	replayHash := results[2].hash

	log.Printf("[PDV][DISTRIBUTED] trace=%s exec=%s val=%s replay=%s",
		req.TraceID, execHash[:8], valHash[:8], replayHash[:8])

	if execHash != valHash || valHash != replayHash {
		reason := fmt.Sprintf("distributed PDV disagreement: exec=%s val=%s replay=%s",
			execHash, valHash, replayHash)
		log.Printf("[PDV][DISTRIBUTED_REJECT] trace=%s reason=%s", req.TraceID, reason)
		return DistributedResult{
			Agreed:          false,
			ExecutionHash:   execHash,
			ValidationHash:  valHash,
			ReplayHash:      replayHash,
			TraceID:         req.TraceID,
			RejectionReason: reason,
		}
	}

	log.Printf("[PDV][DISTRIBUTED_PASS] trace=%s all_hashes=%s", req.TraceID, execHash)
	return DistributedResult{
		Agreed:         true,
		ExecutionHash:  execHash,
		ValidationHash: valHash,
		ReplayHash:     replayHash,
		TraceID:        req.TraceID,
	}
}

// ── AGENT HTTP SERVER ─────────────────────────────────────────────────────────
// StartAgentServer starts a standalone PDV agent HTTP server on the given port.
// Run three instances on different ports for true distributed PDV:
//   go pdv.StartAgentServer(9101, "ExecutionAgent")
//   go pdv.StartAgentServer(9102, "ValidationAgent")
//   go pdv.StartAgentServer(9103, "ReplayAgent")

// StartAgentServer starts a PDV agent HTTP server (fire-and-forget, logs errors).
func StartAgentServer(port int, agentName string) {
	if err := StartAgentServerErr(port, agentName); err != nil {
		log.Printf("[PDV][%s] server error: %v", agentName, err)
	}
}

// StartAgentServerErr starts a PDV agent HTTP server and returns any error.
// Used by the standalone pdv-agent binary so it can call log.Fatal on the error.
func StartAgentServerErr(port int, agentName string) error {
	mux := http.NewServeMux()

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var req AgentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(AgentResponse{Error: err.Error()})
			return
		}
		hash := deterministicHash(req)
		log.Printf("[PDV][%s] trace=%s hash=%s", agentName, req.TraceID, hash[:8])
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AgentResponse{
			Hash:    hash,
			TraceID: req.TraceID,
			Agent:   agentName,
		})
	}

	mux.HandleFunc("/pdv/execute", handler)
	mux.HandleFunc("/pdv/validate", handler)
	mux.HandleFunc("/pdv/replay", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "agent": agentName, "strict_mode": fmt.Sprintf("%v", strictMode())})
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("[PDV][%s] starting on %s (strict_mode=%v)", agentName, addr, strictMode())
	return http.ListenAndServe(addr, mux)
}
