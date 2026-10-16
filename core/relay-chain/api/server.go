package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/akashic"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/bridge"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/chain"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/escrow"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/noncestore"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/runtime"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/token"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/trace"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/truthstore"
)

type APIServer struct {
	blockchain    *chain.Blockchain
	bridge        *bridge.Bridge
	port          int
	escrowManager interface{}
	truthStore    *truthstore.Store
	akashic       *akashic.Store
	nonceStore    *noncestore.Store
}

func NewAPIServer(blockchain *chain.Blockchain, bridgeInstance *bridge.Bridge, port int) *APIServer {
	// Initialize proper escrow manager using dependency injection
	escrowManager := NewEscrowManagerForBlockchain(blockchain)

	// Inject the escrow manager into the blockchain
	blockchain.EscrowManager = escrowManager

	// Initialize truth store (Phase 5)
	ts, err := truthstore.New("tantra_truth.jsonl")
	if err != nil {
		log.Printf("[TANTRA] Warning: failed to open truth store: %v", err)
	}

	// Initialize AKASHIC lineage store (Phase 4/6)
	ak, err := akashic.New("akashic_lineage.jsonl")
	if err != nil {
		log.Printf("[AKASHIC] Warning: failed to open lineage store: %v", err)
	}

	// Initialize persistent nonce store (Phase 2)
	ns, err := noncestore.New("nonce_ledger.jsonl")
	if err != nil {
		log.Printf("[NONCESTORE] Warning: failed to open nonce store: %v", err)
	}

	return &APIServer{
		blockchain:    blockchain,
		bridge:        bridgeInstance,
		port:          port,
		escrowManager: escrowManager,
		truthStore:    ts,
		akashic:       ak,
		nonceStore:    ns,
	}
}

// Note: Workflow manager functionality removed - bridge SDK runs separately

// NewEscrowManagerForBlockchain creates a new escrow manager for the blockchain
func NewEscrowManagerForBlockchain(blockchain *chain.Blockchain) interface{} {
	// Create a real escrow manager using dependency injection
	return escrow.NewEscrowManager(blockchain)
}

func (s *APIServer) Start() {
	// Enable CORS for all routes
	http.HandleFunc("/dev", s.enableCORS(s.serveDevMode))
	http.HandleFunc("/api/blockchain/info", s.enableCORS(s.getBlockchainInfo))
	http.HandleFunc("/api/admin/add-tokens", s.enableCORS(s.addTokens))
	http.HandleFunc("/api/admin/submit-transaction", s.enableCORS(s.submitTransaction))
	http.HandleFunc("/api/wallets", s.enableCORS(s.getWallets))
	http.HandleFunc("/api/node/info", s.enableCORS(s.getNodeInfo))
	// P2P info endpoint for persistent identity and multiaddrs
	http.HandleFunc("/api/p2p/info", s.enableCORS(s.getP2PInfo))
	http.HandleFunc("/api/dev/test-dex", s.enableCORS(s.testDEX))
	http.HandleFunc("/api/dev/test-bridge", s.enableCORS(s.testBridge))
	http.HandleFunc("/api/dev/test-staking", s.enableCORS(s.testStaking))
	http.HandleFunc("/api/dev/test-multisig", s.enableCORS(s.testMultisig))
	http.HandleFunc("/api/dev/test-otc", s.enableCORS(s.testOTC))
	http.HandleFunc("/api/dev/test-escrow", s.enableCORS(s.testEscrow))
	http.HandleFunc("/api/escrow/request", s.enableCORS(s.handleEscrowRequest))
	http.HandleFunc("/api/balance/query", s.enableCORS(s.handleBalanceQuery))

	// Production Cache Balance API endpoints
	http.HandleFunc("/api/balance/cached", s.enableCORS(s.handleBalanceCached))
	http.HandleFunc("/api/balance/all", s.enableCORS(s.handleBalanceAll))
	http.HandleFunc("/api/balance/preload", s.enableCORS(s.handleBalancePreload))
	http.HandleFunc("/api/balance", s.enableCORS(s.handleBalanceSimple))

	// Health and diagnostics endpoints
	http.HandleFunc("/api/health", s.enableCORS(s.handleHealth))
	http.HandleFunc("/api/diagnostics", s.enableCORS(s.handleDiagnostics))
	http.HandleFunc("/api/v1/diagnostics", s.enableCORS(s.handleDiagnostics))

	// OTC Trading API endpoints
	http.HandleFunc("/api/otc/create", s.enableCORS(s.handleOTCCreate))
	http.HandleFunc("/api/otc/orders", s.enableCORS(s.handleOTCOrders))
	http.HandleFunc("/api/otc/match", s.enableCORS(s.handleOTCMatch))
	http.HandleFunc("/api/otc/cancel", s.enableCORS(s.handleOTCCancel))
	http.HandleFunc("/api/otc/events", s.enableCORS(s.handleOTCEvents))

	// Slashing API endpoints
	http.HandleFunc("/api/slashing/events", s.enableCORS(s.handleSlashingEvents))
	http.HandleFunc("/api/slashing/report", s.enableCORS(s.handleSlashingReport))
	http.HandleFunc("/api/slashing/execute", s.enableCORS(s.handleSlashingExecute))
	http.HandleFunc("/api/slashing/validator-status", s.enableCORS(s.handleValidatorStatus))

	// Cross-Chain DEX API endpoints
	http.HandleFunc("/api/cross-chain/quote", s.enableCORS(s.handleCrossChainQuote))
	http.HandleFunc("/api/cross-chain/swap", s.enableCORS(s.handleCrossChainSwap))
	http.HandleFunc("/api/cross-chain/order", s.enableCORS(s.handleCrossChainOrder))
	http.HandleFunc("/api/cross-chain/orders", s.enableCORS(s.handleCrossChainOrders))
	http.HandleFunc("/api/cross-chain/supported-chains", s.enableCORS(s.handleSupportedChains))

	// Bridge event endpoints
	http.HandleFunc("/api/bridge/events", s.enableCORS(s.handleBridgeEvents))
	http.HandleFunc("/api/bridge/subscribe", s.enableCORS(s.handleBridgeSubscribe))
	http.HandleFunc("/api/bridge/approval/simulate", s.enableCORS(s.handleBridgeApprovalSimulation))

	// Note: Workflow management removed - bridge SDK runs separately

	// Unified monitoring endpoints
	http.HandleFunc("/api/monitoring/unified", s.enableCORS(s.handleUnifiedMonitoring))
	http.HandleFunc("/api/monitoring/dashboard", s.enableCORS(s.handleMonitoringDashboard))
	http.HandleFunc("/api/monitoring/metrics", s.enableCORS(s.handleMonitoringMetrics))

	// Relay endpoints for external chains
	http.HandleFunc("/api/relay/submit", s.enableCORS(s.handleRelaySubmit))
	http.HandleFunc("/api/relay/status", s.enableCORS(s.handleRelayStatus))

	// Transaction History API
	http.HandleFunc("/api/transactions/history", s.enableCORS(s.handleTransactionHistory))
	
	// CEX Integration API
	http.HandleFunc("/api/blocks/latest", s.enableCORS(s.handleCEXBlocksLatest))
	http.HandleFunc("/api/blocks/", s.enableCORS(s.handleCEXBlocksHeight)) // Handles /api/blocks/{height}
	http.HandleFunc("/api/transactions/broadcast", s.enableCORS(s.handleCEXBroadcast))
	
	http.HandleFunc("/api/relay/events", s.enableCORS(s.handleRelayEvents))
	http.HandleFunc("/api/relay/validate", s.enableCORS(s.handleRelayValidate))

	// Signature verification endpoint (Phase 1)
	http.HandleFunc("/api/sig/verify", s.enableCORS(s.handleSigVerify))

	// Nonce governance endpoints (Phase 2)
	http.HandleFunc("/api/nonce/lookup", s.enableCORS(s.handleNonceLookup))
	http.HandleFunc("/api/nonce/records", s.enableCORS(s.handleNonceRecords))

	// Distributed replay equality endpoints (Phase 3)
	http.HandleFunc("/api/replay/equality", s.enableCORS(s.handleReplayEquality))
	http.HandleFunc("/api/replay/state-root", s.enableCORS(s.handleStateRootEquality))

	// Infrastructure failure reconstruction endpoints (Phase 4)
	http.HandleFunc("/api/akashic/corrupt-simulate", s.enableCORS(s.handleCorruptSimulate))

	// Constitutional boundary endpoints (Phase 5)
	http.HandleFunc("/api/constitution/declaration", s.enableCORS(s.handleConstitutionDeclaration))
	http.HandleFunc("/api/constitution/verify-boundary", s.enableCORS(s.handleVerifyBoundary))

	// Trace continuity verification endpoint (Phase 1C)
	http.HandleFunc("/api/trace/verify", s.enableCORS(s.handleTraceVerify))

	// Replay verification endpoint (Phase 1F)
	http.HandleFunc("/api/replay/verify", s.enableCORS(s.handleReplayVerify))

	// Convergence proof endpoint (Phase 1G)
	http.HandleFunc("/api/convergence/proof", s.enableCORS(s.handleConvergenceProof))

	// KSML/CET upstream contract endpoint (Gap 5)
	http.HandleFunc("/api/ksml/submit", s.enableCORS(s.handleKSMLSubmit))

	// KarmaChain consistency + replication endpoints (Gap 2/4)
	http.HandleFunc("/api/karmachain/consistency", s.enableCORS(s.handleKarmaChainConsistency))
	http.HandleFunc("/api/karmachain/reconstruct", s.enableCORS(s.handleKarmaChainReconstruct))
	http.HandleFunc("/api/akashic/replicate", s.enableCORS(s.handleAkashicReplicate))

	// TANTRA Phase 5 — truth store verification endpoints
	http.HandleFunc("/api/tantra/verify", s.enableCORS(s.handleTantraVerify))
	http.HandleFunc("/api/tantra/records", s.enableCORS(s.handleTantraRecords))
	http.HandleFunc("/api/tantra/chain-integrity", s.enableCORS(s.handleTantraChainIntegrity))

	// AKASHIC lineage endpoints (Phase 4/6)
	http.HandleFunc("/api/akashic/lineage", s.enableCORS(s.handleAkashicLineage))
	http.HandleFunc("/api/akashic/trace", s.enableCORS(s.handleAkashicTrace))
	http.HandleFunc("/api/akashic/reconstruct", s.enableCORS(s.handleAkashicReconstruct))

	// TANTRA ecosystem status — Scope 5 observability
	http.HandleFunc("/api/tantra/status", s.enableCORS(s.handleTantraStatus))

	// Wallet key management endpoints
	http.HandleFunc("/api/wallet/keygen", s.enableCORS(s.handleWalletKeygen))
	http.HandleFunc("/api/wallet/sign", s.enableCORS(s.handleWalletSign))
	http.HandleFunc("/api/wallet/payload-hash", s.enableCORS(s.handleWalletPayloadHash))

	// AI fraud detection integration - minimal endpoints for status
	http.HandleFunc("/api/ai-fraud/status", s.enableCORS(s.handleAIFraudStatus))

	// Explorer endpoints (Phase 4)
	http.HandleFunc("/api/explorer/blocks", s.enableCORS(s.handleExplorerBlocks))
	http.HandleFunc("/api/explorer/transaction", s.enableCORS(s.handleExplorerTransaction))
	http.HandleFunc("/api/explorer/address", s.enableCORS(s.handleExplorerAddress))
	http.HandleFunc("/api/explorer/stats", s.enableCORS(s.handleExplorerStats))

	// VAJRA Phase 1 — Runtime Identity Card
	http.HandleFunc("/identity", s.enableCORS(s.handleIdentity))

	// VAJRA Phase 2 — Unified contract validation
	http.HandleFunc("/api/vajra/contracts", s.enableCORS(s.handleVAJRAContracts))

	// ML data endpoint for Yashika
	http.HandleFunc("/api/transaction-data", s.enableCORS(s.handleTransactionData))

	// Cybercrime dashboard endpoints (Alpha POC)
	http.HandleFunc("/api/cybercrime/report", s.enableCORS(s.handleCybercrimeReport))
	http.HandleFunc("/api/cybercrime/status/", s.enableCORS(s.handleCybercrimeStatus))
	http.HandleFunc("/api/cybercrime/freeze", s.enableCORS(s.handleCybercrimeFreeze))
	http.HandleFunc("/api/cybercrime/unfreeze", s.enableCORS(s.handleCybercrimeUnfreeze))
	http.HandleFunc("/api/cybercrime/stats", s.enableCORS(s.handleCybercrimeStats))

	// Health check endpoint (using handleHealth instead of duplicate handleHealthCheck)

	// Cyber Defence redirect for direct access on node port 8080
	http.HandleFunc("/defence/", s.enableCORS(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://localhost:8101/", http.StatusTemporaryRedirect)
	}))
	http.HandleFunc("/defence", s.enableCORS(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://localhost:8101/", http.StatusTemporaryRedirect)
	}))

	// Root handler to serve operations dashboard instead of 404s or dev mode
	var uiPath string
	if _, err := os.Stat("../../ui/operations-dashboard/static"); err == nil {
		uiPath = "../../ui/operations-dashboard/static" // Local dev
	} else {
		uiPath = "./ui/operations-dashboard/static" // Docker container
	}
	fs := http.FileServer(http.Dir(uiPath))

	http.HandleFunc("/", s.enableCORS(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// In production, nginx already redirects / → /wallet/.
			// When the node is hit directly (e.g. port 8080 in local dev),
			// serve the operations dashboard index instead of a blank page.
			http.ServeFile(w, r, uiPath+"/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	}))
	fmt.Printf("🌐 API Server starting on port %d\n", s.port)
	fmt.Printf("🌐 Open http://localhost:%d in your browser\n", s.port)
	addr := fmt.Sprintf("0.0.0.0:%d", s.port)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// handleTantraStatus returns live TANTRA ecosystem connectivity status.
// GET /api/tantra/status
// Proves: Wallet→PDV→Governance→Blockchain→Bucket→AKASHIC→Replay→Multi-Node
func (s *APIServer) enableCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[PANIC_RECOVERY] endpoint=%s error=%v", r.URL.Path, rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success":    false,
					"error_code": "INTERNAL_SERVER_PANIC_RECOVERED",
					"message":    fmt.Sprintf("Internal error recovered: %v", rec),
					"timestamp":  time.Now().Unix(),
				})
			}
		}()

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler(w, r)
	}
}


func (s *APIServer) getBlockchainInfo(w http.ResponseWriter, r *http.Request) {
	info := s.blockchain.GetBlockchainInfo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

func (s *APIServer) addTokens(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Address string `json:"address"`
		Token   string `json:"token"`
		Amount  uint64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   "Invalid request format",
		})
		return
	}

	err := s.blockchain.AddTokenBalance(req.Address, req.Token, req.Amount)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Added %d %s tokens to %s", req.Amount, req.Token, req.Address),
	})
}

// submitTransaction is a legacy endpoint.
// Phase 1D: now fully converged through runtime.Execute() — same canonical path as /api/relay/submit.
// Builds a schema v1 contract from the legacy payload and delegates to the runtime.
func (s *APIServer) submitTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		From   string `json:"from"`
		To     string `json:"to"`
		Token  string `json:"token"`
		Amount uint64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "Invalid request format"})
		return
	}

	// Build a canonical schema v1 contract from the legacy fields.
	contract := &schema.TxContract{
		SchemaVersion: schema.CurrentVersion,
		Type:          "token_transfer",
		From:          req.From,
		To:            req.To,
		Amount:        req.Amount,
		TokenID:       req.Token,
		Nonce:         uint64(time.Now().UnixNano()),
		Timestamp:     time.Now().Unix(),
	}

	// Immutable trace context — Phase 1C.
	tc := trace.New("")

	// Canonical runtime execution — Phase 1D.
	execResult := runtime.Execute(runtime.ExecutionRequest{
		Contract:     contract,
		Blockchain:   s.blockchain,
		TruthStore:   s.truthStore,
		AkashicStore: s.akashic,
	})

	// Trace continuity assertion.
	if execResult.TraceID != "" {
		if err := tc.Inject(execResult.TraceID); err != nil {
			s.rejectObservable(w, "TRACE_BREAK", err.Error(), tc.ID(), http.StatusInternalServerError)
			return
		}
	}

	if !execResult.Allowed {
		httpStatus := http.StatusForbidden
		if execResult.ErrorCode == "BLOCKCHAIN_REJECT" {
			httpStatus = http.StatusUnprocessableEntity
		}
		s.rejectObservable(w, execResult.ErrorCode, execResult.RejectionReason, execResult.TraceID, httpStatus)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":         true,
		"message":         "Transaction submitted successfully",
		"tx_id":           execResult.TxHash,
		"trace_id":        execResult.TraceID,
		"execution_hash":  execResult.ExecutionHash,
		"validation_hash": execResult.ValidationHash,
		"replay_hash":     execResult.ReplayHash,
		"fraud_decision":  execResult.FraudDecision,
		"block_height":    execResult.BlockHeight,
	})
}

func (s *APIServer) getWallets(w http.ResponseWriter, r *http.Request) {
	// This would integrate with the wallet service to get wallet information
	// For now, return the accounts from blockchain state
	info := s.blockchain.GetBlockchainInfo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"accounts":      info["accounts"],
		"tokenBalances": info["tokenBalances"],
	})
}

func (s *APIServer) getNodeInfo(w http.ResponseWriter, r *http.Request) {
	// Get P2P node information
	p2pNode := s.blockchain.P2PNode
	if p2pNode == nil {
		http.Error(w, "P2P node not available", http.StatusServiceUnavailable)
		return
	}

	// Build multiaddresses
	addresses := make([]string, 0)
	for _, addr := range p2pNode.Host.Addrs() {
		fullAddr := fmt.Sprintf("%s/p2p/%s", addr.String(), p2pNode.Host.ID().String())
		addresses = append(addresses, fullAddr)
	}

	nodeInfo := map[string]interface{}{
		"peer_id":   p2pNode.Host.ID().String(),
		"addresses": addresses,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodeInfo)
}

// getP2PInfo returns persistent peer identity and multiaddrs for the main dashboard
func (s *APIServer) getP2PInfo(w http.ResponseWriter, r *http.Request) {
	p2pNode := s.blockchain.P2PNode
	if p2pNode == nil || p2pNode.Host == nil {
		http.Error(w, "P2P node not available", http.StatusServiceUnavailable)
		return
	}

	localIP := chain.GetLocalIP()
	peerID := p2pNode.Host.ID().String()
	addresses := make([]string, 0)
	for _, addr := range p2pNode.Host.Addrs() {
		addrStr := addr.String()
		// Replace 0.0.0.0 with the actual local IP so other containers can dial us
		if strings.Contains(addrStr, "/ip4/0.0.0.0/") {
			addrStr = strings.Replace(addrStr, "/ip4/0.0.0.0/", "/ip4/"+localIP+"/", 1)
		}
		addresses = append(addresses, fmt.Sprintf("%s/p2p/%s", addrStr, peerID))
	}

	// Determine lastSeen from peerinfo.json if present (Docker or local path)
	lastSeen := time.Now().UTC().Format(time.RFC3339)
	for _, p := range []string{"/data/blockchain/identity/peerinfo.json", "./data/blockchain/identity/peerinfo.json"} {
		if b, err := os.ReadFile(p); err == nil {
			var v map[string]interface{}
			if err := json.Unmarshal(b, &v); err == nil {
				if ls, ok := v["lastSeen"].(string); ok {
					lastSeen = ls
				}
			}
			break
		}
	}

	// Check bridge health quickly (non-blocking UI also checks directly)
	bridgeServiceURL := os.Getenv("BRIDGE_SERVICE_URL")
	if bridgeServiceURL == "" {
		bridgeServiceURL = "http://localhost:8084"
	}
	bridgeConnected := false
	client := &http.Client{Timeout: 500 * time.Millisecond}
	if resp, err := client.Get(bridgeServiceURL + "/health"); err == nil {
		if resp.StatusCode == 200 {
			bridgeConnected = true
		}
		resp.Body.Close()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"peerId":          p2pNode.Host.ID().String(),
		"multiaddrs":      addresses,
		"bridgeConnected": bridgeConnected,
		"lastSeen":        lastSeen,
	})
}

// serveDevMode serves the developer testing page
func (s *APIServer) serveDevMode(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Blackhole Blockchain - Dev Mode</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; }
        .header { background: #e74c3c; color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; text-align: center; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 20px; }
        .card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .card h3 { margin-top: 0; color: #2c3e50; border-bottom: 2px solid #e74c3c; padding-bottom: 10px; }
        .btn { background: #3498db; color: white; border: none; padding: 12px 20px; border-radius: 4px; cursor: pointer; margin: 5px; width: 100%; }
        .btn:hover { background: #2980b9; }
        .btn-success { background: #27ae60; }
        .btn-success:hover { background: #229954; }
        .btn-warning { background: #f39c12; }
        .btn-warning:hover { background: #e67e22; }
        .btn-danger { background: #e74c3c; }
        .btn-danger:hover { background: #c0392b; }
        .form-group { margin-bottom: 15px; }
        .form-group label { display: block; margin-bottom: 5px; font-weight: bold; }
        .form-group input, .form-group select, .form-group textarea { width: 100%; padding: 8px; border: 1px solid #ddd; border-radius: 4px; box-sizing: border-box; }
        .result { margin-top: 15px; padding: 10px; border-radius: 4px; white-space: pre-wrap; word-wrap: break-word; }
        .success { background: #d4edda; color: #155724; border: 1px solid #c3e6cb; }
        .error { background: #f8d7da; color: #721c24; border: 1px solid #f5c6cb; }
        .info { background: #d1ecf1; color: #0c5460; border: 1px solid #bee5eb; }
        .loading { background: #fff3cd; color: #856404; border: 1px solid #ffeaa7; }
        .nav-links { text-align: center; margin-bottom: 20px; }
        .nav-links a { color: #3498db; text-decoration: none; margin: 0 15px; font-weight: bold; }
        .nav-links a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔧 Blackhole Blockchain - Developer Mode</h1>
            <p>Test all blockchain functionalities with detailed error output</p>
        </div>

        <div class="nav-links">
            <a id="cyberDefenceBtn" href="/defence/" target="_blank">Open Cyber Defence Dashboard</a>
        </div>
        <script>
            (function() {
                var btn = document.getElementById('cyberDefenceBtn');
                if (btn && (window.location.port === '8080' || window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1')) {
                    btn.href = 'http://' + window.location.hostname + ':8101/';
                }
            })();
        </script>

        <div class="grid">
            <!-- DEX Testing -->
            <div class="card">
                <h3>💱 DEX (Decentralized Exchange) Testing</h3>
                <form id="dexForm">
                    <div class="form-group">
                        <label>Action:</label>
                        <select id="dexAction">
                            <option value="create_pair">Create Trading Pair</option>
                            <option value="add_liquidity">Add Liquidity</option>
                            <option value="swap">Execute Swap</option>
                            <option value="get_quote">Get Swap Quote</option>
                            <option value="get_pools">Get All Pools</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>Token A:</label>
                        <input type="text" id="dexTokenA" value="BHIV" placeholder="e.g., BHIV">
                    </div>
                    <div class="form-group">
                        <label>Token B:</label>
                        <input type="text" id="dexTokenB" value="USDT" placeholder="e.g., USDT">
                    </div>
                    <div class="form-group">
                        <label>Amount A:</label>
                        <input type="number" id="dexAmountA" value="1000" placeholder="Amount of Token A">
                    </div>
                    <div class="form-group">
                        <label>Amount B:</label>
                        <input type="number" id="dexAmountB" value="5000" placeholder="Amount of Token B">
                    </div>
                    <button type="submit" class="btn btn-success">Test DEX Function</button>
                </form>
                <div id="dexResult" class="result" style="display: none;"></div>
            </div>

            <!-- Bridge Testing -->
            <div class="card">
                <h3><svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M15,3V7.59L7.59,15H4V17H7.59L15,9.59V15H17V9.59L9.59,2H15V3M17,17V21H15V17H17Z"/></svg> Cross-Chain Bridge Testing</h3>
                <form id="bridgeForm">
                    <div class="form-group">
                        <label>Action:</label>
                        <select id="bridgeAction">
                            <option value="initiate_transfer">Initiate Transfer</option>
                            <option value="confirm_transfer">Confirm Transfer</option>
                            <option value="get_status">Get Transfer Status</option>
                            <option value="get_history">Get Transfer History</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>Source Chain:</label>
                        <input type="text" id="bridgeSourceChain" value="blackhole" placeholder="e.g., blackhole">
                    </div>
                    <div class="form-group">
                        <label>Destination Chain:</label>
                        <input type="text" id="bridgeDestChain" value="ethereum" placeholder="e.g., ethereum">
                    </div>
                    <div class="form-group">
                        <label>Source Address:</label>
                        <input type="text" id="bridgeSourceAddr" placeholder="Source wallet address">
                    </div>
                    <div class="form-group">
                        <label>Destination Address:</label>
                        <input type="text" id="bridgeDestAddr" placeholder="Destination wallet address">
                    </div>
                    <div class="form-group">
                        <label>Token Symbol:</label>
                        <input type="text" id="bridgeToken" value="BHIV" placeholder="e.g., BHIV">
                    </div>
                    <div class="form-group">
                        <label>Amount:</label>
                        <input type="number" id="bridgeAmount" value="100" placeholder="Amount to transfer">
                    </div>
                    <button type="submit" class="btn btn-warning">Test Bridge Function</button>
                </form>
                <div id="bridgeResult" class="result" style="display: none;"></div>
            </div>

            <!-- Staking Testing -->
            <div class="card">
                <h3>🏦 Staking System Testing</h3>
                <form id="stakingForm">
                    <div class="form-group">
                        <label>Action:</label>
                        <select id="stakingAction">
                            <option value="stake">Stake Tokens</option>
                            <option value="unstake">Unstake Tokens</option>
                            <option value="get_stakes">Get All Stakes</option>
                            <option value="get_rewards">Calculate Rewards</option>
                            <option value="claim_rewards">Claim Rewards</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>Staker Address:</label>
                        <input type="text" id="stakingAddress" placeholder="Wallet address">
                    </div>
                    <div class="form-group">
                        <label>Token Symbol:</label>
                        <input type="text" id="stakingToken" value="BHIV" placeholder="e.g., BHIV">
                    </div>
                    <div class="form-group">
                        <label>Amount:</label>
                        <input type="number" id="stakingAmount" value="500" placeholder="Amount to stake">
                    </div>
                    <button type="submit" class="btn btn-success">Test Staking Function</button>
                </form>
                <div id="stakingResult" class="result" style="display: none;"></div>
            </div>

            <!-- Escrow Testing -->
            <div class="card">
                <h3>🔒 Escrow System Testing</h3>
                <form id="escrowForm">
                    <div class="form-group">
                        <label>Action:</label>
                        <select id="escrowAction">
                            <option value="create_escrow">Create Escrow</option>
                            <option value="confirm_escrow">Confirm Escrow</option>
                            <option value="release_escrow">Release Escrow</option>
                            <option value="cancel_escrow">Cancel Escrow</option>
                            <option value="dispute_escrow">Dispute Escrow</option>
                            <option value="get_escrow">Get Escrow Details</option>
                            <option value="get_user_escrows">Get User Escrows</option>
                        </select>
                    </div>
                    <div class="form-group">
                        <label>Sender Address:</label>
                        <input type="text" id="escrowSender" placeholder="Sender wallet address">
                    </div>
                    <div class="form-group">
                        <label>Receiver Address:</label>
                        <input type="text" id="escrowReceiver" placeholder="Receiver wallet address">
                    </div>
                    <div class="form-group">
                        <label>Arbitrator Address:</label>
                        <input type="text" id="escrowArbitrator" placeholder="Arbitrator address (optional)">
                    </div>
                    <div class="form-group">
                        <label>Token Symbol:</label>
                        <input type="text" id="escrowToken" value="BHIV" placeholder="e.g., BHIV">
                    </div>
                    <div class="form-group">
                        <label>Amount:</label>
                        <input type="number" id="escrowAmount" value="100" placeholder="Amount to escrow">
                    </div>
                    <div class="form-group">
                        <label>Escrow ID (for actions on existing escrow):</label>
                        <input type="text" id="escrowID" placeholder="Escrow ID">
                    </div>
                    <div class="form-group">
                        <label>Expiration Hours:</label>
                        <input type="number" id="escrowExpiration" value="24" placeholder="Hours until expiration">
                    </div>
                    <div class="form-group">
                        <label>Description:</label>
                        <textarea id="escrowDescription" placeholder="Escrow description" rows="3"></textarea>
                    </div>
                    <button type="submit" class="btn btn-danger">Test Escrow Function</button>
                </form>
                <div id="escrowResult" class="result" style="display: none;"></div>
            </div>

            <!-- Continue with more testing modules... -->
        </div>
    </div>

    <script>
        // DEX Testing
        document.getElementById('dexForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            await testFunction('dex', 'dexResult', {
                action: document.getElementById('dexAction').value,
                token_a: document.getElementById('dexTokenA').value,
                token_b: document.getElementById('dexTokenB').value,
                amount_a: parseInt(document.getElementById('dexAmountA').value) || 0,
                amount_b: parseInt(document.getElementById('dexAmountB').value) || 0
            });
        });

        // Bridge Testing
        document.getElementById('bridgeForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            await testFunction('bridge', 'bridgeResult', {
                action: document.getElementById('bridgeAction').value,
                source_chain: document.getElementById('bridgeSourceChain').value,
                dest_chain: document.getElementById('bridgeDestChain').value,
                source_address: document.getElementById('bridgeSourceAddr').value,
                dest_address: document.getElementById('bridgeDestAddr').value,
                token_symbol: document.getElementById('bridgeToken').value,
                amount: parseInt(document.getElementById('bridgeAmount').value) || 0
            });
        });

        // Staking Testing
        document.getElementById('stakingForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            await testFunction('staking', 'stakingResult', {
                action: document.getElementById('stakingAction').value,
                address: document.getElementById('stakingAddress').value,
                token_symbol: document.getElementById('stakingToken').value,
                amount: parseInt(document.getElementById('stakingAmount').value) || 0
            });
        });

        // Escrow Testing
        document.getElementById('escrowForm').addEventListener('submit', async (e) => {
            e.preventDefault();
            await testFunction('escrow', 'escrowResult', {
                action: document.getElementById('escrowAction').value,
                sender: document.getElementById('escrowSender').value,
                receiver: document.getElementById('escrowReceiver').value,
                arbitrator: document.getElementById('escrowArbitrator').value,
                token_symbol: document.getElementById('escrowToken').value,
                amount: parseInt(document.getElementById('escrowAmount').value) || 0,
                escrow_id: document.getElementById('escrowID').value,
                expiration_hours: parseInt(document.getElementById('escrowExpiration').value) || 24,
                description: document.getElementById('escrowDescription').value
            });
        });

        // Generic test function
        async function testFunction(module, resultId, data) {
            const resultDiv = document.getElementById(resultId);
            resultDiv.style.display = 'block';
            resultDiv.className = 'result loading';
            resultDiv.textContent = 'Testing ' + module + ' functionality...';

            try {
                const response = await fetch('/api/dev/test-' + module, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(data)
                });

                const result = await response.json();

                if (result.success) {
                    resultDiv.className = 'result success';
                    resultDiv.textContent = 'SUCCESS: ' + result.message + '\n\nData: ' + JSON.stringify(result.data, null, 2);
                } else {
                    resultDiv.className = 'result error';
                    resultDiv.textContent = 'ERROR: ' + result.error + '\n\nDetails: ' + (result.details || 'No additional details');
                }
            } catch (error) {
                resultDiv.className = 'result error';
                resultDiv.textContent = 'NETWORK ERROR: ' + error.message;
            }
        }
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// testDEX handles DEX testing requests
func getTokenNames(tokenRegistry map[string]*token.Token) []string {
	names := make([]string, 0, len(tokenRegistry))
	for name := range tokenRegistry {
		names = append(names, name)
	}
	return names
}

// Cross-Chain DEX order storage functions
func readBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, fmt.Errorf("empty request body")
	}
	defer r.Body.Close()
	buf := make([]byte, 0, 512)
	tmp := make([]byte, 512)
	for {
		n, err := r.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf, nil
}

// handleRelayStatus handles relay status requests
func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"status":  "healthy",
		"message": "Blockchain API is running",
		"data": map[string]interface{}{
			"block_height": len(s.blockchain.Blocks),
			"pending_txs":  len(s.blockchain.PendingTxs),
			"total_supply": s.blockchain.TotalSupply,
		},
	})
}

// Note: Workflow Management Handlers removed - bridge SDK runs separately

// handleTantraVerify verifies a transaction against the real blockchain state.
// GET /api/tantra/verify?tx_hash=<hash>  OR  ?trace_id=<id>
