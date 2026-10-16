package pdv

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestPDV_DeterministicHash(t *testing.T) {
	req := AgentRequest{
		TraceID:   "trace-pdv-1",
		Type:      "token_transfer",
		From:      "0xAlice",
		To:        "0xBob",
		Amount:    500,
		TokenID:   "BHIV",
		Fee:       5,
		Nonce:     1,
		Signature: "sig123",
	}

	h1 := deterministicHash(req)
	h2 := deterministicHash(req)

	if h1 == "" || h1 != h2 {
		t.Errorf("Expected deterministic hash calculation to produce identical results, got %s and %s", h1, h2)
	}
}

func TestPDV_DistributedCheck_Agreed(t *testing.T) {
	// Create 3 test servers simulating the 3 PDV agents
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req AgentRequest
		json.NewDecoder(r.Body).Decode(&req)
		h := deterministicHash(req)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(AgentResponse{
			Hash:    h,
			TraceID: req.TraceID,
			Agent:   "MockAgent",
		})
	})

	s1 := httptest.NewServer(handler)
	defer s1.Close()
	s2 := httptest.NewServer(handler)
	defer s2.Close()
	s3 := httptest.NewServer(handler)
	defer s3.Close()

	os.Setenv("PDV_EXECUTION_AGENT_URL", s1.URL)
	os.Setenv("PDV_VALIDATION_AGENT_URL", s2.URL)
	os.Setenv("PDV_REPLAY_AGENT_URL", s3.URL)
	defer func() {
		os.Unsetenv("PDV_EXECUTION_AGENT_URL")
		os.Unsetenv("PDV_VALIDATION_AGENT_URL")
		os.Unsetenv("PDV_REPLAY_AGENT_URL")
	}()

	req := AgentRequest{
		TraceID: "trace-pdv-agree",
		Type:    "token_transfer",
		From:    "0xAlice",
		To:      "0xBob",
		Amount:  100,
		TokenID: "BHIV",
		Nonce:   1,
	}

	res := Check(req)
	if !res.Agreed {
		t.Fatalf("Expected PDV 3-agent agreement, got rejection: %s", res.RejectionReason)
	}
	if res.ExecutionHash == "" || res.ValidationHash == "" || res.ReplayHash == "" {
		t.Errorf("Expected non-empty hashes from all 3 agents")
	}
}

func TestPDV_DistributedCheck_PartitionRejection(t *testing.T) {
	os.Setenv("PDV_STRICT_MODE", "true")
	os.Setenv("PDV_EXECUTION_AGENT_URL", "http://127.0.0.1:59999/unreachable")
	os.Setenv("PDV_VALIDATION_AGENT_URL", "http://127.0.0.1:59998/unreachable")
	os.Setenv("PDV_REPLAY_AGENT_URL", "http://127.0.0.1:59997/unreachable")
	defer func() {
		os.Unsetenv("PDV_STRICT_MODE")
		os.Unsetenv("PDV_EXECUTION_AGENT_URL")
		os.Unsetenv("PDV_VALIDATION_AGENT_URL")
		os.Unsetenv("PDV_REPLAY_AGENT_URL")
	}()

	req := AgentRequest{
		TraceID: "trace-pdv-partition",
		Type:    "token_transfer",
		From:    "0xAlice",
		To:      "0xBob",
		Amount:  100,
	}

	res := Check(req)
	if res.Agreed {
		t.Fatalf("Expected PDV partition to reject in strict mode, but it passed")
	}
}
