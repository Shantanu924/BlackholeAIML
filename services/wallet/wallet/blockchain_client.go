package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	"crypto/sha256"
	"encoding/hex"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/escrow"
	"github.com/Shivam-Patel-G/blackhole-blockchain/services/wallet/tantra"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
)

// BridgeEvent represents a bridge event notification
type BridgeEvent struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	SourceChain string `json:"source_chain"`
	DestChain   string `json:"dest_chain"`
	TokenSymbol string `json:"token_symbol"`
	Amount      uint64 `json:"amount"`
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	Status      string `json:"status"`
	TxHash      string `json:"tx_hash"`
	Timestamp   int64  `json:"timestamp"`
}

// BridgeEventSubscription manages bridge event subscriptions
type BridgeEventSubscription struct {
	WalletAddress string
	EventChannel  chan BridgeEvent
	Active        bool
}

// BlockchainClient handles communication with the blockchain
type BlockchainClient struct {
	P2PHost             host.Host
	ConnectedPeers      []string
	APIEndpoint         string
	BridgeEndpoint      string
	BridgeSubscriptions map[string]*BridgeEventSubscription
	bridgeMutex         sync.RWMutex
	tantra              *tantra.Runtime
}

// BalanceQuery represents a balance query request
type BalanceQuery struct {
	Address     string `json:"address"`
	TokenSymbol string `json:"token_symbol"`
}

// BalanceResponse represents a balance query response
type BalanceResponse struct {
	Success bool   `json:"success"`
	Balance uint64 `json:"balance"`
	Error   string `json:"error,omitempty"`
}

// NewBlockchainClient creates a new client to interact with the blockchain
func NewBlockchainClient(port int) (*BlockchainClient, error) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", port+1000)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create P2P host: %v", err)
	}

	return &BlockchainClient{
		P2PHost:             h,
		ConnectedPeers:      make([]string, 0),
		APIEndpoint:         "",
		BridgeEndpoint:      "http://localhost:8084",
		BridgeSubscriptions: make(map[string]*BridgeEventSubscription),
	}, nil
}

// ConnectToBlockchain connects to an existing blockchain node
func (client *BlockchainClient) ConnectToBlockchain(peerAddr string) error {
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %v", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to get peer info: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.P2PHost.Connect(ctx, *info); err != nil {
		return fmt.Errorf("failed to connect to blockchain node: %v", err)
	}

	client.ConnectedPeers = append(client.ConnectedPeers, peerAddr)
	fmt.Printf("✅ Connected to blockchain node: %s\n", peerAddr)
	return nil
}

// GetTokenBalance returns the balance of a specific token for an address
func (client *BlockchainClient) GetTokenBalance(address, tokenSymbol string) (uint64, error) {
	apiURL := client.APIEndpoint
	if apiURL == "" {
		apiURL = os.Getenv("BLOCKCHAIN_API_URL")
		if apiURL == "" {
			apiURL = "http://localhost:8080"
		}
	}

	url := fmt.Sprintf("%s/api/balance/query", apiURL)
	payload := map[string]string{"address": address, "token_symbol": tokenSymbol}
	jsonData, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, fmt.Errorf("balance query failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to parse balance response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		return 0, fmt.Errorf("balance query unsuccessful")
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("invalid balance response format")
	}

	balance, ok := data["balance"].(float64)
	if !ok {
		return 0, nil
	}
	return uint64(balance), nil
}

// GetTransactionsForAddress fetches confirmed transaction history from the relay chain
func (client *BlockchainClient) GetTransactionsForAddress(address string) ([]*TransactionRecord, error) {
	apiURL := client.APIEndpoint
	if apiURL == "" {
		apiURL = os.Getenv("BLOCKCHAIN_API_URL")
		if apiURL == "" {
			apiURL = "http://localhost:8080"
		}
	}

	url := fmt.Sprintf("%s/api/transactions/history?address=%s", apiURL, address)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("transaction history query failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse transaction history response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		return nil, fmt.Errorf("transaction history query unsuccessful")
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		// No transactions or invalid format
		return []*TransactionRecord{}, nil
	}

	var records []*TransactionRecord
	for _, item := range data {
		txMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		
		record := &TransactionRecord{
			WalletAddr: address,
			Status:     "confirmed",
		}
		
		if id, ok := txMap["id"].(string); ok {
			record.TxHash = id
		}
		if t, ok := txMap["type"].(string); ok {
			record.Type = t
		}
		if from, ok := txMap["from"].(string); ok {
			record.From = from
		}
		if to, ok := txMap["to"].(string); ok {
			record.To = to
		}
		if amount, ok := txMap["amount"].(float64); ok {
			record.Amount = uint64(amount)
		}
		if token, ok := txMap["token"].(string); ok {
			record.TokenSymbol = token
		}
		if timestamp, ok := txMap["timestamp"].(float64); ok {
			record.Timestamp = time.Unix(int64(timestamp), 0)
		}
		if block, ok := txMap["block"].(float64); ok {
			record.BlockHeight = uint64(block)
		}
		records = append(records, record)
	}

	return records, nil
}

// tantraRuntime returns the singleton TANTRA runtime for this client.
// A single runtime preserves the nonce sequence across calls.
func (client *BlockchainClient) tantraRuntime() *tantra.Runtime {
	if client.tantra == nil {
		apiURL := client.APIEndpoint
		if apiURL == "" {
			apiURL = os.Getenv("BLOCKCHAIN_API_URL")
			if apiURL == "" {
				apiURL = "http://localhost:8080"
			}
		}
		client.tantra = tantra.NewRuntime(apiURL)
	}
	return client.tantra
}

// signAndSubmitIntent builds the intent, signs the canonical payload, and submits it.
func (client *BlockchainClient) signAndSubmitIntent(req tantra.IntentRequest, privateKey []byte) (*tantra.SubmissionResult, error) {
	intent, err := client.tantraRuntime().BuildIntent(req)
	if err != nil {
		return nil, fmt.Errorf("TANTRA build error: %v", err)
	}

	// Reconstruct the payload EXACTLY as the relay chain sigverify expects it (without the signature)
	payload := tantra.CanonicalPayload(intent.TraceID, intent.Type, intent.From, intent.To, intent.TokenID, "", intent.Amount, intent.Fee, intent.Nonce)
	
	// Hash the payload
	hashBytes := sha256.Sum256(payload)

	// Sign the hash
	privKey, _ := btcec.PrivKeyFromBytes(privateKey)
	sig := ecdsa.Sign(privKey, hashBytes[:])
	
	// Encode signature as DER hex
	intent.Signature = hex.EncodeToString(sig.Serialize())

	// Submit the signed intent
	return client.tantraRuntime().Submit(intent)
}

// TransferTokens transfers tokens via the canonical TANTRA runtime.
// Phase 2B: NO direct blockchain submission. ALL transfers go through
// /api/relay/submit → PDV → Governance → Blockchain → Bucket → AKASHIC.
func (client *BlockchainClient) TransferTokens(from, to, tokenSymbol string, amount uint64, privateKey []byte) error {
	if amount == 0 {
		return fmt.Errorf("WALLET_VIOLATION: amount must be greater than 0")
	}
	if to == "" {
		return fmt.Errorf("WALLET_VIOLATION: recipient address required")
	}

	result, err := client.signAndSubmitIntent(tantra.IntentRequest{
		From:    from,
		To:      to,
		Amount:  amount,
		TokenID: tokenSymbol,
		Type:    "token_transfer",
	}, privateKey)
	if err != nil {
		return fmt.Errorf("TANTRA submission error: %v", err)
	}
	if !result.Success {
		return fmt.Errorf("TANTRA rejected: [%s] %s (trace=%s)",
			result.ErrorCode, result.RejectionReason, result.TraceID)
	}

	fmt.Printf("✅ Transfer accepted: trace=%s tx=%s height=%d\n",
		result.TraceID, result.TransactionID, result.BlockHeight)
	return nil
}

// StakeTokens stakes tokens via the canonical TANTRA runtime.
func (client *BlockchainClient) StakeTokens(address, tokenSymbol string, amount uint64, privateKey []byte) error {
	result, err := client.signAndSubmitIntent(tantra.IntentRequest{
		From:    address,
		To:      "staking_contract",
		Amount:  amount,
		TokenID: tokenSymbol,
		Type:    "stake_deposit",
	}, privateKey)
	if err != nil {
		return fmt.Errorf("TANTRA submission error: %v", err)
	}
	if !result.Success {
		return fmt.Errorf("TANTRA rejected: [%s] %s", result.ErrorCode, result.RejectionReason)
	}
	fmt.Printf("✅ Stake accepted: trace=%s tx=%s\n", result.TraceID, result.TransactionID)
	return nil
}

// TransferTokensWithEscrow transfers tokens using escrow for added security.
// The escrow creation itself is submitted via the canonical TANTRA runtime.
func (client *BlockchainClient) TransferTokensWithEscrow(from, to, arbitrator, tokenSymbol string, amount uint64, expirationHours int, description string, privateKey []byte) (*escrow.EscrowContract, error) {
	if amount == 0 {
		return nil, fmt.Errorf("WALLET_VIOLATION: amount must be greater than 0")
	}
	if from == "" || to == "" {
		return nil, fmt.Errorf("WALLET_VIOLATION: sender and receiver addresses required")
	}

	contract, err := client.CreateEscrow(from, to, arbitrator, tokenSymbol, amount, expirationHours, description, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create escrow: %v", err)
	}

	fmt.Printf("✅ Escrow transfer initiated: %s\n", contract.ID)
	return contract, nil
}

// SubscribeToBridgeEvents subscribes a wallet to bridge events
func (client *BlockchainClient) SubscribeToBridgeEvents(walletAddress string) error {
	client.bridgeMutex.Lock()
	defer client.bridgeMutex.Unlock()

	if subscription, exists := client.BridgeSubscriptions[walletAddress]; exists && subscription.Active {
		return fmt.Errorf("wallet %s already subscribed to bridge events", walletAddress)
	}

	subscription := &BridgeEventSubscription{
		WalletAddress: walletAddress,
		EventChannel:  make(chan BridgeEvent, 100),
		Active:        true,
	}

	client.BridgeSubscriptions[walletAddress] = subscription
	go client.bridgeEventListener(subscription)

	fmt.Printf("✅ Wallet %s subscribed to bridge events\n", walletAddress)
	return nil
}

// UnsubscribeFromBridgeEvents unsubscribes a wallet from bridge events
func (client *BlockchainClient) UnsubscribeFromBridgeEvents(walletAddress string) error {
	client.bridgeMutex.Lock()
	defer client.bridgeMutex.Unlock()

	subscription, exists := client.BridgeSubscriptions[walletAddress]
	if !exists {
		return fmt.Errorf("wallet %s not subscribed to bridge events", walletAddress)
	}

	subscription.Active = false
	close(subscription.EventChannel)
	delete(client.BridgeSubscriptions, walletAddress)

	fmt.Printf("✅ Wallet %s unsubscribed from bridge events\n", walletAddress)
	return nil
}

// GetBridgeEvents retrieves bridge events for a specific wallet
func (client *BlockchainClient) GetBridgeEvents(walletAddress string) ([]BridgeEvent, error) {
	url := fmt.Sprintf("%s/api/bridge/events?wallet=%s", client.BridgeEndpoint, walletAddress)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bridge events: %v", err)
	}
	defer resp.Body.Close()

	var response struct {
		Success bool          `json:"success"`
		Data    []BridgeEvent `json:"data"`
		Error   string        `json:"error"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode bridge events response: %v", err)
	}

	if !response.Success {
		return nil, fmt.Errorf("bridge API error: %s", response.Error)
	}

	return response.Data, nil
}

// HandleBridgeNotification processes incoming bridge event notifications
func (client *BlockchainClient) HandleBridgeNotification(event BridgeEvent) error {
	client.bridgeMutex.RLock()
	defer client.bridgeMutex.RUnlock()

	for _, address := range []string{event.FromAddress, event.ToAddress} {
		if subscription, exists := client.BridgeSubscriptions[address]; exists && subscription.Active {
			select {
			case subscription.EventChannel <- event:
			default:
				fmt.Printf("⚠️ Bridge event channel full for wallet %s\n", address)
			}
		}
	}
	return nil
}

func (client *BlockchainClient) bridgeEventListener(subscription *BridgeEventSubscription) {
	for subscription.Active {
		select {
		case event, ok := <-subscription.EventChannel:
			if !ok {
				return
			}
			fmt.Printf("🌉 Bridge event for wallet %s: type=%s amount=%d %s\n",
				subscription.WalletAddress, event.Type, event.Amount, event.TokenSymbol)
		case <-time.After(30 * time.Second):
			if subscription.Active {
				client.pollBridgeEvents(subscription.WalletAddress)
			}
		}
	}
}

func (client *BlockchainClient) pollBridgeEvents(walletAddress string) {
	events, err := client.GetBridgeEvents(walletAddress)
	if err != nil {
		return
	}
	client.bridgeMutex.RLock()
	subscription, exists := client.BridgeSubscriptions[walletAddress]
	client.bridgeMutex.RUnlock()

	if exists && subscription.Active {
		for _, event := range events {
			select {
			case subscription.EventChannel <- event:
			default:
			}
		}
	}
}

// sendMessageToPeer sends a P2P message to a peer (used for non-transaction messages only)
func (client *BlockchainClient) sendMessageToPeer(peerAddr string, data []byte) error {
	maddr, err := multiaddr.NewMultiaddr(peerAddr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr: %v", err)
	}

	info, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return fmt.Errorf("failed to get peer info: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.P2PHost.NewStream(ctx, info.ID, "/blackhole/1.0.0")
	if err != nil {
		return fmt.Errorf("failed to open stream: %v", err)
	}
	defer stream.Close()

	_, err = stream.Write(data)
	return err
}

// GetConnectedPeers returns the list of connected peer addresses
func (client *BlockchainClient) GetConnectedPeers() []string {
	return client.ConnectedPeers
}

// IsConnected returns true if the client is connected to at least one blockchain node
func (client *BlockchainClient) IsConnected() bool {
	return len(client.ConnectedPeers) > 0
}

// extractAPIPortFromPeer extracts the API port from a peer address
func (client *BlockchainClient) extractAPIPortFromPeer(peerAddr string) string {
	if strings.Contains(peerAddr, "/tcp/3000/") {
		return "8080"
	}
	if strings.Contains(peerAddr, "/tcp/3001/") {
		return "8081"
	}
	return ""
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// ===== ESCROW OPERATIONS =====

// CreateEscrow creates a new escrow contract via blockchain API
func (client *BlockchainClient) CreateEscrow(sender, receiver, arbitrator, tokenSymbol string, amount uint64, expirationHours int, description string, privateKey []byte) (*escrow.EscrowContract, error) {
	apiURL := os.Getenv("BLOCKCHAIN_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	escrowData := map[string]interface{}{
		"action":           "create_escrow",
		"sender":           sender,
		"receiver":         receiver,
		"arbitrator":       arbitrator,
		"token_symbol":     tokenSymbol,
		"amount":           amount,
		"expiration_hours": expirationHours,
		"description":      description,
	}

	response, err := client.sendEscrowRequestViaHTTP(escrowData, "8080")
	if err != nil {
		return nil, fmt.Errorf("failed to create escrow: %v", err)
	}

	escrowID, ok := response["escrow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response: missing escrow_id")
	}

	contract := &escrow.EscrowContract{
		ID:           escrowID,
		Sender:       sender,
		Receiver:     receiver,
		Arbitrator:   arbitrator,
		TokenSymbol:  tokenSymbol,
		Amount:       amount,
		Status:       escrow.EscrowPending,
		Description:  description,
		CreatedAt:    time.Now().Unix(),
		ExpiresAt:    time.Now().Add(time.Duration(expirationHours) * time.Hour).Unix(),
		Signatures:   make(map[string]bool),
		RequiredSigs: 2,
		Conditions:   make(map[string]interface{}),
	}

	fmt.Printf("✅ Escrow created: %s\n", escrowID)
	return contract, nil
}

// ConfirmEscrow confirms an escrow contract
func (client *BlockchainClient) ConfirmEscrow(escrowID, confirmer string, privateKey []byte) error {
	_, err := client.sendEscrowRequestViaHTTP(map[string]interface{}{
		"action":    "confirm_escrow",
		"escrow_id": escrowID,
		"confirmer": confirmer,
	}, "8080")
	return err
}

// ReleaseEscrow releases funds from an escrow to the receiver
func (client *BlockchainClient) ReleaseEscrow(escrowID, releaser string, privateKey []byte) error {
	_, err := client.sendEscrowRequestViaHTTP(map[string]interface{}{
		"action":    "release_escrow",
		"escrow_id": escrowID,
		"releaser":  releaser,
	}, "8080")
	return err
}

// CancelEscrow cancels an escrow and returns funds to sender
func (client *BlockchainClient) CancelEscrow(escrowID, canceller string, privateKey []byte) error {
	_, err := client.sendEscrowRequestViaHTTP(map[string]interface{}{
		"action":    "cancel_escrow",
		"escrow_id": escrowID,
		"canceller": canceller,
	}, "8080")
	return err
}

// sendEscrowRequestViaHTTP sends escrow request via HTTP API
func (client *BlockchainClient) sendEscrowRequestViaHTTP(escrowData map[string]interface{}, port string) (map[string]interface{}, error) {
	apiURL := os.Getenv("BLOCKCHAIN_API_URL")
	if apiURL == "" {
		apiURL = fmt.Sprintf("http://localhost:%s", port)
	}
	url := apiURL + "/api/escrow/request"

	jsonData, err := json.Marshal(escrowData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal escrow data: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		errorMsg := "unknown error"
		if msg, ok := response["error"].(string); ok {
			errorMsg = msg
		}
		return nil, fmt.Errorf("escrow request failed: %s", errorMsg)
	}

	return response, nil
}

// parseEscrowStatus converts string status to EscrowStatus
func parseEscrowStatus(status string) escrow.EscrowStatus {
	switch status {
	case "pending":
		return escrow.EscrowPending
	case "confirmed":
		return escrow.EscrowConfirmed
	case "released":
		return escrow.EscrowReleased
	case "cancelled":
		return escrow.EscrowCancelled
	case "disputed":
		return escrow.EscrowDisputed
	default:
		return escrow.EscrowPending
	}
}

// GetEscrowDetails gets details of an escrow contract
func (client *BlockchainClient) GetEscrowDetails(escrowID string) (*escrow.EscrowContract, error) {
	response, err := client.sendEscrowRequestViaHTTP(map[string]interface{}{
		"action":    "get_escrow",
		"escrow_id": escrowID,
	}, "8080")
	if err != nil {
		return nil, err
	}
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid escrow details response")
	}
	return &escrow.EscrowContract{
		ID:          escrowID,
		Sender:      fmt.Sprintf("%v", data["sender"]),
		Receiver:    fmt.Sprintf("%v", data["receiver"]),
		Arbitrator:  fmt.Sprintf("%v", data["arbitrator"]),
		TokenSymbol: fmt.Sprintf("%v", data["token_symbol"]),
		Amount:      uint64(data["amount"].(float64)),
		Status:      parseEscrowStatus(fmt.Sprintf("%v", data["status"])),
		Description: fmt.Sprintf("%v", data["description"]),
		Signatures:  make(map[string]bool),
		Conditions:  make(map[string]interface{}),
	}, nil
}

// GetUserEscrows gets all escrows where the user is involved
func (client *BlockchainClient) GetUserEscrows(userAddress string) ([]*escrow.EscrowContract, error) {
	response, err := client.sendEscrowRequestViaHTTP(map[string]interface{}{
		"action":       "get_user_escrows",
		"user_address": userAddress,
	}, "8080")
	if err != nil {
		return nil, err
	}
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return []*escrow.EscrowContract{}, nil
	}
	escrowsRaw, ok := data["escrows"].([]interface{})
	if !ok {
		return []*escrow.EscrowContract{}, nil
	}
	var contracts []*escrow.EscrowContract
	for _, e := range escrowsRaw {
		ed, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		contracts = append(contracts, &escrow.EscrowContract{
			ID:          fmt.Sprintf("%v", ed["id"]),
			Sender:      fmt.Sprintf("%v", ed["sender"]),
			Receiver:    fmt.Sprintf("%v", ed["receiver"]),
			TokenSymbol: fmt.Sprintf("%v", ed["token_symbol"]),
			Amount:      uint64(ed["amount"].(float64)),
			Status:      parseEscrowStatus(fmt.Sprintf("%v", ed["status"])),
			Signatures:  make(map[string]bool),
			Conditions:  make(map[string]interface{}),
		})
	}
	return contracts, nil
}
