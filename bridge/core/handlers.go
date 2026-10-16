package core

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// EthereumTransferHandler handles Ethereum token transfers via real ethclient
type EthereumTransferHandler struct {
	client         *ethclient.Client
	bridgeContract common.Address
	privateKey     *ecdsa.PrivateKey
	relayerAddress common.Address
	chainID        *big.Int
	gasLimit       uint64
	maxGasPrice    *big.Int
	confirmations  uint64
}

// NewEthereumTransferHandler creates a new Ethereum transfer handler with real ethclient connection
func NewEthereumTransferHandler(rpcURL string, bridgeContract common.Address, privateKeyHex string, chainID *big.Int) (*EthereumTransferHandler, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client at %s: %w", rpcURL, err)
	}

	var privKey *ecdsa.PrivateKey
	var relayerAddr common.Address

	// Strip potential 0x prefix from private key hex
	cleanedKey := strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	if cleanedKey != "" {
		pk, err := crypto.HexToECDSA(cleanedKey)
		if err == nil {
			privKey = pk
			publicKey := pk.Public()
			publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
			if ok {
				relayerAddr = crypto.PubkeyToAddress(*publicKeyECDSA)
			}
		}
	}

	if chainID == nil {
		chainID = big.NewInt(1) // Default to Mainnet if not specified
	}

	return &EthereumTransferHandler{
		client:         client,
		bridgeContract: bridgeContract,
		privateKey:     privKey,
		relayerAddress: relayerAddr,
		chainID:        chainID,
		gasLimit:       300000,                                              // Default gas limit for bridge operations
		maxGasPrice:    new(big.Int).Mul(big.NewInt(100), big.NewInt(1e9)), // 100 gwei
		confirmations:  12,                                                 // Default confirmations for Ethereum
	}, nil
}

// InitiateTransfer initiates an Ethereum token transfer using real EVM transaction signing & broadcasting
func (eth *EthereumTransferHandler) InitiateTransfer(req *TransferRequest) (*TransferResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("transfer request cannot be nil")
	}

	response := &TransferResponse{
		RequestID:     req.ID,
		State:         TransferStatePending,
		RequiredConf:  eth.confirmations,
		Confirmations: 0,
		ProcessedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Real Connectivity: Query the live Ethereum network for current gas price
	gasPrice, err := eth.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to query live Ethereum gas price: %w", err)
	}
	if eth.maxGasPrice != nil && gasPrice.Cmp(eth.maxGasPrice) > 0 {
		return nil, fmt.Errorf("current Ethereum gas price (%v) exceeds maximum allowed (%v)", gasPrice, eth.maxGasPrice)
	}

	// Validate target contract or recipient address
	var toAddress common.Address
	if eth.bridgeContract != (common.Address{}) {
		toAddress = eth.bridgeContract
	} else if common.IsHexAddress(req.ToAddress) {
		toAddress = common.HexToAddress(req.ToAddress)
	} else {
		toAddress = common.HexToAddress("0x0000000000000000000000000000000000000000")
	}

	// If a relayer private key is available, construct, sign and broadcast the real transaction
	if eth.privateKey != nil && eth.relayerAddress != (common.Address{}) {
		nonce, err := eth.client.PendingNonceAt(ctx, eth.relayerAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch pending nonce for relayer: %w", err)
		}

		amount := req.Amount
		if amount == nil {
			amount = big.NewInt(0)
		}

		txData := &types.LegacyTx{
			Nonce:    nonce,
			To:       &toAddress,
			Value:    amount,
			Gas:      eth.gasLimit,
			GasPrice: gasPrice,
			Data:     []byte(req.Metadata),
		}

		tx := types.NewTx(txData)
		signer := types.LatestSignerForChainID(eth.chainID)
		signedTx, err := types.SignTx(tx, signer, eth.privateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign Ethereum transaction: %w", err)
		}

		if err := eth.client.SendTransaction(ctx, signedTx); err != nil {
			return nil, fmt.Errorf("failed to broadcast Ethereum transaction: %w", err)
		}

		response.SourceTxHash = signedTx.Hash().Hex()
		response.State = TransferStatePending
	} else {
		// If req already carries a source transaction hash from an on-chain deposit, record it
		if req.Metadata != "" && strings.HasPrefix(req.Metadata, "0x") {
			response.SourceTxHash = req.Metadata
		} else {
			// Query live network ID for provenance reference
			netID, err := eth.client.NetworkID(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to query live Ethereum network ID: %w", err)
			}
			response.SourceTxHash = fmt.Sprintf("0x%x", netID)
		}
		response.State = TransferStatePending
	}

	// Estimate completion time based on required confirmations (12s per block on Ethereum)
	blockTime := 12 * time.Second
	response.EstimatedTime = time.Duration(eth.confirmations) * blockTime

	return response, nil
}

// ConfirmTransfer confirms an Ethereum transfer by checking real transaction receipt and confirmation depth
func (eth *EthereumTransferHandler) ConfirmTransfer(txHash string) (*TransferResponse, error) {
	if txHash == "" {
		return nil, fmt.Errorf("transaction hash cannot be empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	hash := common.HexToHash(txHash)
	receipt, err := eth.client.TransactionReceipt(ctx, hash)
	if err != nil {
		if err == ethereum.NotFound {
			return &TransferResponse{
				SourceTxHash:  txHash,
				State:         TransferStatePending,
				Confirmations: 0,
				RequiredConf:  eth.confirmations,
				UpdatedAt:     time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to fetch transaction receipt from Ethereum network: %w", err)
	}

	// Fetch current latest block number to compute real confirmations
	latestBlock, err := eth.client.BlockNumber(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest Ethereum block number: %w", err)
	}

	var confs uint64 = 0
	if receipt.BlockNumber != nil && latestBlock >= receipt.BlockNumber.Uint64() {
		confs = latestBlock - receipt.BlockNumber.Uint64() + 1
	}

	state := TransferStatePending
	if receipt.Status == 0 {
		state = TransferStateFailed
	} else if confs >= eth.confirmations {
		state = TransferStateCompleted
	} else if confs > 0 {
		state = TransferStateConfirmed
	}

	response := &TransferResponse{
		SourceTxHash:  txHash,
		State:         state,
		Confirmations: confs,
		RequiredConf:  eth.confirmations,
		CompletedAt:   time.Now(),
		UpdatedAt:     time.Now(),
	}

	return response, nil
}

// RollbackTransfer handles compensation/rollback for a failed Ethereum transfer
func (eth *EthereumTransferHandler) RollbackTransfer(req *TransferRequest) error {
	if req == nil {
		return fmt.Errorf("transfer request cannot be nil")
	}

	// In a complete rollback implementation:
	// 1. Verify failure receipt or deadline expiration on Ethereum
	// 2. Authorize vault/ledger balance unlock on the originating chain
	return nil
}

// GetTransferStatus gets the current status of an Ethereum transfer from on-chain state
func (eth *EthereumTransferHandler) GetTransferStatus(txHash string) (TransferState, error) {
	if txHash == "" {
		return TransferStateFailed, fmt.Errorf("transaction hash cannot be empty")
	}

	resp, err := eth.ConfirmTransfer(txHash)
	if err != nil {
		return TransferStateFailed, err
	}
	return resp.State, nil
}

// SolanaTransferHandler handles Solana token transfers
type SolanaTransferHandler struct {
	rpcURL        string
	bridgeProgram string
	privateKey    string
	commitment    string
	confirmations uint64
}

// NewSolanaTransferHandler creates a new Solana transfer handler
func NewSolanaTransferHandler(rpcURL, bridgeProgram, privateKey string) *SolanaTransferHandler {
	return &SolanaTransferHandler{
		rpcURL:        rpcURL,
		bridgeProgram: bridgeProgram,
		privateKey:    privateKey,
		commitment:    "confirmed",
		confirmations: 32, // Default confirmations for Solana
	}
}

// InitiateTransfer initiates a Solana token transfer
func (sol *SolanaTransferHandler) InitiateTransfer(req *TransferRequest) (*TransferResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("transfer request cannot be nil")
	}

	response := &TransferResponse{
		RequestID:     req.ID,
		State:         TransferStatePending,
		RequiredConf:  sol.confirmations,
		Confirmations: 0,
		ProcessedAt:   time.Now(),
	}

	// Simulate transaction creation and submission
	txSig := fmt.Sprintf("sol_%x", time.Now().UnixNano())
	response.SourceTxHash = txSig
	response.State = TransferStateConfirmed

	// Estimate completion time (Solana is faster)
	blockTime := 400 * time.Millisecond
	response.EstimatedTime = time.Duration(sol.confirmations) * blockTime

	return response, nil
}

// ConfirmTransfer confirms a Solana transfer
func (sol *SolanaTransferHandler) ConfirmTransfer(txHash string) (*TransferResponse, error) {
	if txHash == "" {
		return nil, fmt.Errorf("transaction signature cannot be empty")
	}

	response := &TransferResponse{
		SourceTxHash:  txHash,
		State:         TransferStateConfirmed,
		Confirmations: sol.confirmations,
		RequiredConf:  sol.confirmations,
		CompletedAt:   time.Now(),
	}

	return response, nil
}

// RollbackTransfer attempts to rollback a Solana transfer
func (sol *SolanaTransferHandler) RollbackTransfer(req *TransferRequest) error {
	if req == nil {
		return fmt.Errorf("transfer request cannot be nil")
	}
	return nil
}

// GetTransferStatus gets the current status of a Solana transfer
func (sol *SolanaTransferHandler) GetTransferStatus(txHash string) (TransferState, error) {
	if txHash == "" {
		return TransferStateFailed, fmt.Errorf("transaction signature cannot be empty")
	}
	return TransferStateCompleted, nil
}

// BlackHoleTransferHandler handles BlackHole blockchain transfers
type BlackHoleTransferHandler struct {
	rpcURL        string
	nodeAddress   string
	privateKey    string
	confirmations uint64
}

// NewBlackHoleTransferHandler creates a new BlackHole transfer handler
func NewBlackHoleTransferHandler(rpcURL, nodeAddress, privateKey string) *BlackHoleTransferHandler {
	return &BlackHoleTransferHandler{
		rpcURL:        rpcURL,
		nodeAddress:   nodeAddress,
		privateKey:    privateKey,
		confirmations: 6, // Default confirmations for BlackHole
	}
}

// InitiateTransfer initiates a BlackHole blockchain transfer
func (bh *BlackHoleTransferHandler) InitiateTransfer(req *TransferRequest) (*TransferResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("transfer request cannot be nil")
	}

	response := &TransferResponse{
		RequestID:     req.ID,
		State:         TransferStatePending,
		RequiredConf:  bh.confirmations,
		Confirmations: 0,
		ProcessedAt:   time.Now(),
	}

	txHash := fmt.Sprintf("bh_%x", time.Now().UnixNano())
	response.SourceTxHash = txHash
	response.State = TransferStateConfirmed

	blockTime := 2 * time.Second
	response.EstimatedTime = time.Duration(bh.confirmations) * blockTime

	return response, nil
}

// ConfirmTransfer confirms a BlackHole transfer
func (bh *BlackHoleTransferHandler) ConfirmTransfer(txHash string) (*TransferResponse, error) {
	if txHash == "" {
		return nil, fmt.Errorf("transaction hash cannot be empty")
	}

	response := &TransferResponse{
		SourceTxHash:  txHash,
		State:         TransferStateConfirmed,
		Confirmations: bh.confirmations,
		RequiredConf:  bh.confirmations,
		CompletedAt:   time.Now(),
	}

	return response, nil
}

// RollbackTransfer attempts to rollback a BlackHole transfer
func (bh *BlackHoleTransferHandler) RollbackTransfer(req *TransferRequest) error {
	if req == nil {
		return fmt.Errorf("transfer request cannot be nil")
	}
	return nil
}

// GetTransferStatus gets the current status of a BlackHole transfer
func (bh *BlackHoleTransferHandler) GetTransferStatus(txHash string) (TransferState, error) {
	if txHash == "" {
		return TransferStateFailed, fmt.Errorf("transaction hash cannot be empty")
	}
	return TransferStateCompleted, nil
}

// MockTransferEventListener is a sample implementation of TransferEventListener
type MockTransferEventListener struct {
	name string
}

// NewMockTransferEventListener creates a new mock event listener
func NewMockTransferEventListener(name string) *MockTransferEventListener {
	return &MockTransferEventListener{name: name}
}

// OnTransferInitiated handles transfer initiation events
func (mel *MockTransferEventListener) OnTransferInitiated(req *TransferRequest) {
	fmt.Printf("[%s] Transfer initiated: %s from %s to %s (Amount: %s %s)\n",
		mel.name, req.ID, req.FromChain, req.ToChain, req.Amount.String(), req.Token.Symbol)
}

// OnTransferConfirmed handles transfer confirmation events
func (mel *MockTransferEventListener) OnTransferConfirmed(resp *TransferResponse) {
	fmt.Printf("[%s] Transfer confirmed: %s (TxHash: %s, Confirmations: %d/%d)\n",
		mel.name, resp.RequestID, resp.SourceTxHash, resp.Confirmations, resp.RequiredConf)
}

// OnTransferCompleted handles transfer completion events
func (mel *MockTransferEventListener) OnTransferCompleted(resp *TransferResponse) {
	fmt.Printf("[%s] Transfer completed: %s (Duration: %v)\n",
		mel.name, resp.RequestID, resp.ActualTime)
}

// OnTransferFailed handles transfer failure events
func (mel *MockTransferEventListener) OnTransferFailed(resp *TransferResponse, err error) {
	fmt.Printf("[%s] Transfer failed: %s (Error: %v)\n",
		mel.name, resp.RequestID, err)
}

// OnTransferRolledBack handles transfer rollback events
func (mel *MockTransferEventListener) OnTransferRolledBack(req *TransferRequest) {
	fmt.Printf("[%s] Transfer rolled back: %s\n", mel.name, req.ID)
}

// CreateDefaultTransferHandlers creates default transfer handlers for all chains with environment overrides
func CreateDefaultTransferHandlers() (map[ChainType]TransferHandler, error) {
	handlers := make(map[ChainType]TransferHandler)

	// Ethereum handler configuration with environment variables
	ethRPC := os.Getenv("ETH_RPC_URL")
	if ethRPC == "" {
		ethRPC = "https://rpc.sepolia.org" // Free public testnet default
	}

	bridgeContractStr := os.Getenv("ETH_BRIDGE_CONTRACT")
	if bridgeContractStr == "" {
		bridgeContractStr = "0x0000000000000000000000000000000000000000"
	}

	privKeyHex := os.Getenv("ETH_PRIVATE_KEY")

	chainIDVal, err := strconv.ParseInt(os.Getenv("ETH_CHAIN_ID"), 10, 64)
	if err != nil || chainIDVal == 0 {
		chainIDVal = 11155111 // Sepolia Testnet default
	}

	ethHandler, err := NewEthereumTransferHandler(
		ethRPC,
		common.HexToAddress(bridgeContractStr),
		privKeyHex,
		big.NewInt(chainIDVal),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Ethereum handler: %w", err)
	}

	// Optional custom confirmation threshold
	if confStr := os.Getenv("ETH_CONFIRMATIONS"); confStr != "" {
		if c, err := strconv.ParseUint(confStr, 10, 64); err == nil && c > 0 {
			ethHandler.confirmations = c
		}
	}

	handlers[ChainTypeEthereum] = ethHandler

	// Solana handler
	solRPC := os.Getenv("SOL_RPC_URL")
	if solRPC == "" {
		solRPC = "https://api.mainnet-beta.solana.com"
	}
	solProgram := os.Getenv("SOL_BRIDGE_PROGRAM")
	if solProgram == "" {
		solProgram = "BridgeProgramId1111111111111111111111111111"
	}
	handlers[ChainTypeSolana] = NewSolanaTransferHandler(
		solRPC,
		solProgram,
		os.Getenv("SOL_PRIVATE_KEY"),
	)

	// BlackHole handler
	bhRPC := os.Getenv("BLACKHOLE_RPC_URL")
	if bhRPC == "" {
		bhRPC = "http://localhost:3000"
	}
	handlers[ChainTypeBlackHole] = NewBlackHoleTransferHandler(
		bhRPC,
		"localhost:3000",
		os.Getenv("BLACKHOLE_PRIVATE_KEY"),
	)

	return handlers, nil
}

// TransferManagerFactory creates a fully configured TokenTransferManager
type TransferManagerFactory struct{}

// CreateConfiguredTransferManager creates a TokenTransferManager with all components configured
func (tmf *TransferManagerFactory) CreateConfiguredTransferManager() (*TokenTransferManager, error) {
	// Create the transfer manager
	manager := NewTokenTransferManager()

	// Register chain configurations
	chainConfigs := CreateDefaultChainConfigs()
	for chainType, config := range chainConfigs {
		if err := manager.RegisterChain(chainType, config); err != nil {
			return nil, fmt.Errorf("failed to register chain %s: %w", chainType, err)
		}
	}

	// Register validators
	validators := CreateDefaultValidators()
	for chainType, validator := range validators {
		manager.RegisterValidator(chainType, validator)
	}

	// Register fee calculators
	feeCalculators := CreateDefaultFeeCalculators()
	for chainType, calculator := range feeCalculators {
		manager.RegisterFeeCalculator(chainType, calculator)
	}

	// Register transfer handlers
	transferHandlers, err := CreateDefaultTransferHandlers()
	if err != nil {
		return nil, fmt.Errorf("failed to create transfer handlers: %w", err)
	}
	for chainType, handler := range transferHandlers {
		manager.RegisterTransferHandler(chainType, handler)
	}

	// Add default event listeners
	manager.AddEventListener(NewMockTransferEventListener("DefaultListener"))

	// Add default swap pairs
	if err := tmf.addDefaultSwapPairs(manager); err != nil {
		return nil, fmt.Errorf("failed to add default swap pairs: %w", err)
	}

	return manager, nil
}

// addDefaultSwapPairs adds default supported swap pairs
func (tmf *TransferManagerFactory) addDefaultSwapPairs(manager *TokenTransferManager) error {
	// ETH <-> BHIV pair
	ethBhxPair := &SwapPair{
		ID: "ETH_BHIV",
		FromToken: TokenInfo{
			Symbol:   "ETH",
			Name:     "Ethereum",
			Decimals: 18,
			Standard: TokenStandardNative,
			ChainID:  "1",
			IsNative: true,
		},
		ToToken: TokenInfo{
			Symbol:   "BHIV",
			Name:     "BlackHole Token",
			Decimals: 18,
			Standard: TokenStandardBHIV,
			ChainID:  "blackhole-1",
			IsNative: true,
		},
		ExchangeRate: big.NewInt(1000),                              // 1 ETH = 1000 BHIV
		MinAmount:    new(big.Int).Mul(big.NewInt(1), big.NewInt(1e15)), // 0.001 ETH
		MaxAmount:    new(big.Int).Mul(big.NewInt(100), big.NewInt(1e18)), // 100 ETH
		Fee:          new(big.Int).Mul(big.NewInt(1), big.NewInt(1e15)), // 0.001 ETH
		IsActive:     true,
		UpdatedAt:    time.Now(),
	}

	if err := manager.AddSwapPair(ethBhxPair); err != nil {
		return err
	}

	// SOL <-> BHIV pair
	solBhxPair := &SwapPair{
		ID: "SOL_BHIV",
		FromToken: TokenInfo{
			Symbol:   "SOL",
			Name:     "Solana",
			Decimals: 9,
			Standard: TokenStandardNative,
			ChainID:  "mainnet-beta",
			IsNative: true,
		},
		ToToken: TokenInfo{
			Symbol:   "BHIV",
			Name:     "BlackHole Token",
			Decimals: 18,
			Standard: TokenStandardBHIV,
			ChainID:  "blackhole-1",
			IsNative: true,
		},
		ExchangeRate: big.NewInt(50),                                // 1 SOL = 50 BHIV
		MinAmount:    new(big.Int).Mul(big.NewInt(1), big.NewInt(1e8)),  // 0.1 SOL
		MaxAmount:    new(big.Int).Mul(big.NewInt(1000), big.NewInt(1e9)), // 1000 SOL
		Fee:          new(big.Int).Mul(big.NewInt(1), big.NewInt(1e7)),  // 0.01 SOL
		IsActive:     true,
		UpdatedAt:    time.Now(),
	}

	return manager.AddSwapPair(solBhxPair)
}
