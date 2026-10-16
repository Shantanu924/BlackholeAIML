package explorer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// API provides a REST interface for the explorer indexer
type API struct {
	indexer *Indexer
	port    int
}

// NewAPI creates a new REST API for the explorer
func NewAPI(indexer *Indexer, port int) *API {
	return &API{
		indexer: indexer,
		port:    port,
	}
}

// Start runs the HTTP server for the explorer API
func (a *API) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/blocks/latest", a.handleLatestBlock)
	mux.HandleFunc("/api/v1/blocks/", a.handleGetBlock) // /api/v1/blocks/{hash_or_height}
	mux.HandleFunc("/api/v1/tx/", a.handleGetTx)        // /api/v1/tx/{hash}
	mux.HandleFunc("/api/v1/address/", a.handleGetAddress) // /api/v1/address/{address}/txs

	// Add CORS middleware
	handler := enableCORS(mux)

	fmt.Printf("🌐 Explorer API starting on port %d\n", a.port)
	return http.ListenAndServe(fmt.Sprintf(":%d", a.port), handler)
}

func (a *API) handleLatestBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	latest := a.indexer.blockchain.GetLatestBlock()
	sendJSON(w, latest)
}

func (a *API) handleGetBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Path[len("/api/v1/blocks/"):]
	if id == "" {
		http.Error(w, "Missing block hash or height", http.StatusBadRequest)
		return
	}

	// Try as height
	if height, err := strconv.ParseUint(id, 10, 64); err == nil {
		block, err := a.indexer.GetBlockByHeight(height)
		if err != nil {
			http.Error(w, "Block not found", http.StatusNotFound)
			return
		}
		sendJSON(w, block)
		return
	}

	// Try as hash
	block, err := a.indexer.GetBlockByHash(id)
	if err != nil {
		http.Error(w, "Block not found", http.StatusNotFound)
		return
	}
	sendJSON(w, block)
}

func (a *API) handleGetTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	hash := r.URL.Path[len("/api/v1/tx/"):]
	if hash == "" {
		http.Error(w, "Missing tx hash", http.StatusBadRequest)
		return
	}

	tx, err := a.indexer.GetTransaction(hash)
	if err != nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}
	sendJSON(w, tx)
}

func (a *API) handleGetAddress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// format: /api/v1/address/{address}/txs
	path := r.URL.Path[len("/api/v1/address/"):]
	if path == "" {
		http.Error(w, "Missing address", http.StatusBadRequest)
		return
	}

	// parse address from path up to first slash
	var addr string
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			addr = path[:i]
			break
		}
	}
	if addr == "" {
		addr = path // no slash, just the address
	}

	txs, err := a.indexer.GetAddressTransactions(addr)
	if err != nil {
		http.Error(w, "Failed to get transactions", http.StatusInternalServerError)
		return
	}
	sendJSON(w, txs)
}

func sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// enableCORS enables CORS for all requests
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
