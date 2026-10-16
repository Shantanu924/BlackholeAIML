package replayverifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestReplayVerifier_VerifyEquality_Agreement(t *testing.T) {
	// Create 2 mock nodes returning identical hashes and state roots
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/akashic/trace" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"entry": map[string]interface{}{
					"trace_id":        "trace-agree",
					"execution_hash":  "exec-hash-consensus",
					"validation_hash": "val-hash-consensus",
					"replay_hash":     "replay-hash-consensus",
				},
			})
		} else if r.URL.Path == "/api/akashic/reconstruct" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"result": map[string]interface{}{
					"final_state_root": "state-root-123",
				},
			})
		}
	})

	n1 := httptest.NewServer(handler)
	defer n1.Close()
	n2 := httptest.NewServer(handler)
	defer n2.Close()

	os.Setenv("KARMACHAIN_NODES", n1.URL+","+n2.URL)
	defer os.Unsetenv("KARMACHAIN_NODES")

	res := VerifyEquality("trace-agree")
	if !res.Equal {
		t.Fatalf("Expected all nodes to agree on state root and execution hash, got: %s", res.Message)
	}
	if res.ConsensusHash != "exec-hash-consensus" {
		t.Errorf("Expected consensus hash 'exec-hash-consensus', got '%s'", res.ConsensusHash)
	}
	if res.StateRoot != "state-root-123" {
		t.Errorf("Expected state root 'state-root-123', got '%s'", res.StateRoot)
	}
}

func TestReplayVerifier_VerifyEquality_DivergenceDetected(t *testing.T) {
	// Node 1 returns hash A
	n1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/akashic/trace" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"entry": map[string]interface{}{
					"trace_id":       "trace-diverge",
					"execution_hash": "hash-A",
				},
			})
		}
	}))
	defer n1.Close()

	// Node 2 returns divergent hash B
	n2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/akashic/trace" {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"entry": map[string]interface{}{
					"trace_id":       "trace-diverge",
					"execution_hash": "hash-B",
				},
			})
		}
	}))
	defer n2.Close()

	os.Setenv("KARMACHAIN_NODES", n1.URL+","+n2.URL)
	defer os.Unsetenv("KARMACHAIN_NODES")

	res := VerifyEquality("trace-diverge")
	if res.Equal {
		t.Fatalf("Expected divergence to be detected between nodes, but Equal was true")
	}
}
