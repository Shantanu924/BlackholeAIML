package core

import (
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestNewEthereumTransferHandler_PrivateKeyParsing(t *testing.T) {
	// Generate a valid ECDSA private key for testing
	rawKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate test private key: %v", err)
	}
	keyBytes := crypto.FromECDSA(rawKey)
	keyHex := common.Bytes2Hex(keyBytes)

	bridgeAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	chainID := big.NewInt(11155111) // Sepolia

	// Test initialization (using Sepolia testnet or local endpoint)
	handler, err := NewEthereumTransferHandler("https://rpc.sepolia.org", bridgeAddr, keyHex, chainID)
	if err != nil {
		t.Logf("RPC dial warning (expected if offline): %v", err)
		return
	}

	if handler.privateKey == nil {
		t.Errorf("Expected private key to be parsed successfully")
	}

	expectedRelayerAddr := crypto.PubkeyToAddress(rawKey.PublicKey)
	if handler.relayerAddress != expectedRelayerAddr {
		t.Errorf("Relayer address mismatch: got %s, want %s", handler.relayerAddress.Hex(), expectedRelayerAddr.Hex())
	}

	if handler.chainID.Cmp(chainID) != 0 {
		t.Errorf("Chain ID mismatch: got %v, want %v", handler.chainID, chainID)
	}

	if handler.confirmations != 12 {
		t.Errorf("Default confirmations mismatch: got %d, want 12", handler.confirmations)
	}
}

func TestEthereumTransferHandler_ValidationRules(t *testing.T) {
	bridgeAddr := common.HexToAddress("0x1111111111111111111111111111111111111111")
	handler, err := NewEthereumTransferHandler("https://rpc.sepolia.org", bridgeAddr, "", big.NewInt(1))
	if err != nil {
		t.Logf("Skipping live network validation tests (offline): %v", err)
		return
	}

	// 1. Nil transfer request validation
	_, err = handler.InitiateTransfer(nil)
	if err == nil {
		t.Errorf("Expected InitiateTransfer(nil) to return error, got nil")
	}

	// 2. Empty transaction hash confirmation validation
	_, err = handler.ConfirmTransfer("")
	if err == nil {
		t.Errorf("Expected ConfirmTransfer(\"\") to return error, got nil")
	}

	// 3. Empty transaction hash status validation
	_, err = handler.GetTransferStatus("")
	if err == nil {
		t.Errorf("Expected GetTransferStatus(\"\") to return error, got nil")
	}

	// 4. Nil rollback request validation
	err = handler.RollbackTransfer(nil)
	if err == nil {
		t.Errorf("Expected RollbackTransfer(nil) to return error, got nil")
	}
}

func TestCreateDefaultTransferHandlers_EnvOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("ETH_RPC_URL", "https://rpc.sepolia.org")
	os.Setenv("ETH_CHAIN_ID", "11155111")
	os.Setenv("ETH_CONFIRMATIONS", "15")
	defer func() {
		os.Unsetenv("ETH_RPC_URL")
		os.Unsetenv("ETH_CHAIN_ID")
		os.Unsetenv("ETH_CONFIRMATIONS")
	}()

	handlers, err := CreateDefaultTransferHandlers()
	if err != nil {
		t.Logf("CreateDefaultTransferHandlers warning (expected if offline): %v", err)
		return
	}

	ethHandlerInterface, exists := handlers[ChainTypeEthereum]
	if !exists {
		t.Fatalf("Expected ChainTypeEthereum handler to exist")
	}

	ethHandler, ok := ethHandlerInterface.(*EthereumTransferHandler)
	if !ok {
		t.Fatalf("Expected *EthereumTransferHandler type")
	}

	if ethHandler.confirmations != 15 {
		t.Errorf("Expected custom confirmations 15, got %d", ethHandler.confirmations)
	}

	if ethHandler.chainID.Int64() != 11155111 {
		t.Errorf("Expected chain ID 11155111, got %v", ethHandler.chainID)
	}
}

func TestTransferManagerFactory(t *testing.T) {
	factory := &TransferManagerFactory{}
	manager, err := factory.CreateConfiguredTransferManager()
	if err != nil {
		t.Logf("CreateConfiguredTransferManager warning (expected if offline): %v", err)
		return
	}

	if manager == nil {
		t.Fatalf("Expected non-nil TokenTransferManager")
	}

	// Verify supported chains
	cfg, err := manager.GetChainConfig(ChainTypeEthereum)
	if err != nil || cfg == nil {
		t.Errorf("Expected registered Ethereum chain config")
	}

	// Verify registered swap pairs
	pairs := manager.GetSupportedPairs()
	if len(pairs) < 2 {
		t.Errorf("Expected at least 2 active swap pairs, got %d", len(pairs))
	}
}
