# VAJRA Go Engineering Learning & Reference Guide

**Author / Contributor:** Ishan Shirode  
**Role:** VAJRA Go + Blockchain Engineering Apprenticeship and Production Support  
**Lead / Technical Direction:** Prakash Kumar  
**Target Codebase:** `BHIV-Engineering-Exchange/bhiv-VAJRA_Blockchain` (`core/relay-chain`)  

---

## 1. Executive Summary & Objective

This document formalizes the Go engineering foundations required for contributing safely to the VAJRA blockchain runtime. In the VAJRA architecture, Go is used for high-performance, deterministic consensus, P2P networking, transaction routing, and cryptographic enforcement.

---

## 2. Go Core Concepts Applied to VAJRA

### 2.1 Syntax, Basic Types & Fixed-Size Arithmetic
In blockchain runtimes, integer overflow, memory safety, and deterministic arithmetic are vital:
- **Integer Types:** `uint64` for block heights, nonces, balances, and timestamps (avoiding negative values in financial ledgers).
- **Byte Slices (`[]byte`):** Used for cryptographic hashes (`sha256.Sum256`), signatures (DER/ECDSA `r, s`), and public keys.
- **Hex Encoding:** Standard conversion via `encoding/hex` (`hex.EncodeToString`, `hex.DecodeString`) for transaction hashes and addresses.

### 2.2 Structs and Interfaces
VAJRA decouples components via clean Go interfaces to allow modular testing and isolated subsystem execution:
- **`BlockchainWriter` Interface** (`core/relay-chain/runtime/runtime.go`):
  ```go
  type BlockchainWriter interface {
      ProcessTransactionFromRuntime(traceID, txType, from, to, tokenID string, amount, fee, nonce uint64, timestamp int64) (string, uint64, error)
  }
  ```
- **Transaction Struct** (`core/relay-chain/chain/transaction.go`):
  ```go
  type Transaction struct {
      ID        string          `json:"id"`
      Type      TransactionType `json:"type"`
      From      string          `json:"from"`
      To        string          `json:"to"`
      Amount    uint64          `json:"amount"`
      TokenID   string          `json:"token_id"`
      Data      []byte          `json:"data"`
      Timestamp int64           `json:"timestamp"`
      Nonce     uint64          `json:"nonce"`
      Signature []byte          `json:"signature"`
      Fee       uint64          `json:"fee"`
      GasLimit  uint64          `json:"gas_limit"`
      GasPrice  uint64          `json:"gas_price"`
      PublicKey []byte          `json:"public_key"`
  }
  ```

### 2.3 Packages and Dependency Management
- **Module Path:** `github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain` (defined in `core/relay-chain/go.mod`).
- **Standardized Imports:**
  - Standard library (`crypto/ecdsa`, `crypto/sha256`, `net/http`, `context`, `sync`, `time`)
  - External libraries (`github.com/libp2p/go-libp2p`, `github.com/btcsuite/btcd/btcec/v2`, `github.com/gorilla/mux`)

### 2.4 Error Handling & Zero-Panic Invariance
In blockchain runtimes, panics crash node consensus. VAJRA enforces:
- Explicit error propagation: `if err != nil { return nil, fmt.Errorf("context: %w", err) }`
- Error boundaries & fallback defaults rather than unexpected `panic()` in production paths.

### 2.5 Goroutines, Channels, and Concurrency Controls
- **Mining Loop:** Background worker continuously proposing and committing blocks (`miningLoop(ctx, bc, validator, nodeID)`).
- **Sync Chain:** Concurrent P2P synchronization ticker running every 5 seconds.
- **Mutexes & Thread Safety:** `sync.RWMutex` protecting `Blockchain.Blocks`, `PendingTxs`, `StakeLedger`, and `AccountNonces`.

### 2.6 Context & Graceful Shutdown
- Root lifecycle managed with `context.WithCancel(context.Background())`.
- Signal handling via `os/signal` listening for `os.Interrupt` (SIGINT/SIGTERM), ensuring state is flushed to disk before shutdown.

### 2.7 HTTP Servers, Routing, and REST APIs
- Standard Go HTTP server with `gorilla/mux` or `net/http`.
- CORS middleware, JSON serialization (`json.Marshal`, `json.Unmarshal`), and REST endpoints serving `/api/v1/*`.

### 2.8 Deterministic State Serialization
- **Gob Registration:** `chain.RegisterGobTypes()` ensures binary encoding compatibility across nodes.
- **JSON Contracts:** Strict ordering and hashing in `CalculateHash()` for canonical block and transaction digests.

---

## 3. Go Apprenticeship Checklist for Ishan Shirode

| # | Topic | VAJRA Implementation File | Competency Demonstrated |
|---|---|---|---|
| 1 | Types & Structs | `chain/transaction.go`, `chain/block.go` | Defined strict schemas and fixed-size fields |
| 2 | Interfaces | `runtime/runtime.go` | Implemented `BlockchainWriter` interface |
| 3 | Concurrency | `cmd/relay/main.go`, `chain/p2p.go` | Managed mining routines, sync workers, and channels |
| 4 | Cryptography | `crypto/crypto.go`, `sigverify/sigverify.go` | ECDSA keygen, signing, and verification |
| 5 | Error Handling | `runtime/runtime.go`, `api/server.go` | Structured failure stages and rejection codes |
| 6 | HTTP/JSON APIs | `api/server.go` | Developed REST endpoints and status handlers |
| 7 | Unit & Integration Testing | `runtime/runtime_test.go`, `chain/peer_identity_test.go` | Wrote deterministic test assertions |
