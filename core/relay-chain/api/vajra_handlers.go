package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func (s *APIServer) handleVAJRAContracts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fraudURL := os.Getenv("FRAUD_SERVICE_URL")
	if fraudURL == "" {
		fraudURL = "http://localhost:9090"
	}
	walletURL := os.Getenv("WALLET_SERVICE_URL")
	if walletURL == "" {
		walletURL = "http://localhost:9000"
	}
	bridgeURL := os.Getenv("BRIDGE_SERVICE_URL")
	if bridgeURL == "" {
		bridgeURL = "http://localhost:8084"
	}

	type participantResult struct {
		Service   string                 `json:"service"`
		URL       string                 `json:"url"`
		Reachable bool                   `json:"reachable"`
		Identity  map[string]interface{} `json:"identity,omitempty"`
		Error     string                 `json:"error,omitempty"`
	}

	probe := func(name, base string) participantResult {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Get(base + "/identity")
		if err != nil {
			return participantResult{Service: name, URL: base, Reachable: false, Error: err.Error()}
		}
		defer resp.Body.Close()
		var identity map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&identity); err != nil {
			return participantResult{Service: name, URL: base, Reachable: true, Error: "decode error: " + err.Error()}
		}
		return participantResult{Service: name, URL: base, Reachable: true, Identity: identity}
	}

	// Self identity
	selfIdentity := map[string]interface{}{
		"service":     "vajra-blockchain-node",
		"trust_layer": "VAJRA Blockchain",
		"port":        s.port,
	}

	participants := []participantResult{
		{Service: "blockchain-node", URL: fmt.Sprintf("http://localhost:%d", s.port), Reachable: true, Identity: selfIdentity},
		probe("fraud-detection", fraudURL),
		probe("wallet", walletURL),
		probe("bridge-sdk", bridgeURL),
	}

	allReachable := true
	for _, p := range participants[1:] {
		if !p.Reachable {
			allReachable = false
			break
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"vajra_contracts": "phase2",
		"all_reachable":   allReachable,
		"participants":    participants,
		"timestamp":       time.Now().UTC().Format(time.RFC3339),
	})
}
