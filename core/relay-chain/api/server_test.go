package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/chain"
)

func TestEnableCORS_NormalExecution(t *testing.T) {
	s := &APIServer{}

	called := false
	handler := s.enableCORS(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Test GET request
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if !called {
		t.Fatalf("expected handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if origin := rec.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Fatalf("expected Access-Control-Allow-Origin '*', got %q", origin)
	}

	// Test OPTIONS preflight
	reqOptions := httptest.NewRequest(http.MethodOptions, "/api/test", nil)
	recOptions := httptest.NewRecorder()

	called = false
	handler(recOptions, reqOptions)

	if called {
		t.Fatalf("did not expect handler to be called for OPTIONS request")
	}
	if recOptions.Code != http.StatusOK {
		t.Fatalf("expected status 200 for OPTIONS, got %d", recOptions.Code)
	}
}

func TestEnableCORS_PanicRecovery_ErrorBoundary(t *testing.T) {
	s := &APIServer{}

	panickingHandler := s.enableCORS(func(w http.ResponseWriter, r *http.Request) {
		panic("simulated fatal payload crash")
	})

	req := httptest.NewRequest(http.MethodPost, "/api/risky/endpoint", nil)
	rec := httptest.NewRecorder()

	// Should not crash the process; panic must be caught by recovery middleware
	panickingHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	if resp["success"] != false {
		t.Fatalf("expected success: false, got %v", resp["success"])
	}
	if resp["error_code"] != "INTERNAL_SERVER_PANIC_RECOVERED" {
		t.Fatalf("expected error_code 'INTERNAL_SERVER_PANIC_RECOVERED', got %v", resp["error_code"])
	}
	if msg, ok := resp["message"].(string); !ok || len(msg) == 0 {
		t.Fatalf("expected informative message in error response, got %v", resp["message"])
	}
}

func TestHandleHealth(t *testing.T) {
	// Create mock blockchain
	bc := &chain.Blockchain{
		Blocks: make([]*chain.Block, 0),
	}
	s := &APIServer{
		blockchain: bc,
	}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to parse JSON health response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Fatalf("expected status 'healthy', got %v", resp["status"])
	}
	if resp["success"] != true {
		t.Fatalf("expected success true, got %v", resp["success"])
	}
}

func TestHandleCEXBlocksLatest(t *testing.T) {
	bc := &chain.Blockchain{
		Blocks: make([]*chain.Block, 0),
	}
	s := &APIServer{
		blockchain: bc,
	}

	// 1. When no blocks exist, returns 404
	req := httptest.NewRequest(http.MethodGet, "/api/blocks/latest", nil)
	rec := httptest.NewRecorder()

	s.handleCEXBlocksLatest(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 when no blocks exist, got %d", rec.Code)
	}

	// 2. Add a block
	bc.Blocks = append(bc.Blocks, &chain.Block{
		Header: chain.BlockHeader{
			Index: 1,
		},
		Hash: "0xabc123mockblockhash",
		Transactions: []*chain.Transaction{
			{
				ID:      "tx-001",
				Type:    chain.RegularTransfer,
				From:    "0xAlice",
				To:      "0xBob",
				TokenID: "BHIV",
				Amount:  1000,
			},
		},
	})

	rec2 := httptest.NewRecorder()
	s.handleCEXBlocksLatest(rec2, req)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 with blocks, got %d", rec2.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec2.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["success"] != true {
		t.Fatalf("expected success true")
	}
	data, ok := resp["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected data object in response")
	}
	if data["hash"] != "0xabc123mockblockhash" {
		t.Fatalf("expected hash '0xabc123mockblockhash', got %v", data["hash"])
	}
}
