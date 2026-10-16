# EXCHANGE LISTING & INTEGRATION READINESS SPECIFICATION

**Document Version:** 1.0.0  
**Integration Status:** **Technical Integration Ready (Application Staging Stage)**  
**Audit Standard:** Strict Transparency (Rule 0.4: Never Claim Live Exchange Listing Without Direct Authorization)

---

## 1. Exchange Listing Status Declaration

> [!IMPORTANT]
> **Listing Status:** **APPLICATION & TECHNICAL INTEGRATION READY**  
> **Exchange Trading State:** **NOT STARTED / APPLICATION READY**  
> No centralized exchange (Binance, Coinbase, KuCoin, OKX) listing is currently active or enabled. The engineering objective is to provide the full technical integration surface, standardized polling APIs, deposit/withdrawal infrastructure, and security documentation required for CEX technical onboarding.

---

## 2. Standardized CEX Integration API Surface

The VAJRA node provides dedicated REST endpoints (`core/relay-chain/api/cex_handlers.go`) designed specifically for centralized exchange deposit listeners, balance checkers, and withdrawal broadcasting.

### 2.1 Polling & Block Inspection

#### `GET /api/blocks/latest`
Returns the latest finalized block with full confirmed transaction metadata.
```json
{
  "success": true,
  "data": {
    "height": 1042,
    "hash": "8f3b2a9c...",
    "transactions": [
      {
        "tx_id": "tx_1787184000_1",
        "type": 1,
        "from": "03a1b2c3...",
        "to": "02d4e5f6...",
        "token_symbol": "BHIV",
        "amount": "5000000000000000000",
        "status": "CONFIRMED"
      }
    ]
  }
}
```

#### `GET /api/blocks/{height}`
Retrieves a specific historical block by integer height for continuous indexing and reconciliation.

#### `GET /api/balance?address={address}&token={symbol}`
Queries the authoritative token balance for a specific wallet address from LevelDB.

### 2.2 Withdrawal & Transaction Broadcast

#### `POST /api/transactions/broadcast`
Accepts a raw signed transaction payload (JSON or hex-encoded) directly into the node's mempool (`txPool`).
```json
// Request
{
  "signed_tx_hex": "7b226964223a22...227d"
}

// Response (200 OK)
{
  "success": true,
  "tx_id": "tx_9a8b7c6d...",
  "status": "PENDING"
}
```

---

## 3. Exchange Operational Architecture

```
                      ┌────────────────────────────┐
                      │    Centralized Exchange    │
                      │   (Deposit/Withdrawal Svc) │
                      └──────┬──────────────▲──────┘
                             │              │
        Polls Blocks & Deposits             Broadcasts Withdrawals
        GET /api/blocks/latest              POST /api/transactions/broadcast
                             │              │
                             ▼              │
                      ┌─────────────────────┴──────┐
                      │  VAJRA Node RPC Gateway    │
                      │      (Port 8080 / 443)     │
                      └─────────────┬──────────────┘
                                    │
                                    ▼
                      ┌────────────────────────────┐
                      │   Relay Chain Mempool &    │
                      │  LevelDB Immutable Ledger  │
                      └────────────────────────────┘
```

### 3.1 Deposit Processing
1. Exchange assigns a unique user deposit address (Secp256k1 public key).
2. Exchange deposit daemon polls `GET /api/blocks/latest`.
3. Upon identifying incoming `TokenTransfer` to an exchange deposit address, credits user account after 1 block confirmation.

### 3.2 Withdrawal Processing
1. Exchange hot wallet creates transaction with current nonce and signs with hot wallet private key.
2. Exchange submits transaction via `POST /api/transactions/broadcast`.
3. Node mempool includes transaction into the next block (5–6s block time).

---

## 4. Tokenomics & Asset Parameters for Listing

* **Asset Name:** BlackHole Infiverse (`BHIV`)
* **Decimals:** 18
* **Circulating Supply at Genesis:** 100,010,000 BHIV
* **Maximum Supply Cap:** 1,000,000,000 BHIV (Strictly hardcoded in `token.go`)
* **Transfer Fee / Tax:** 0% on-chain transfer tax (standard network transaction fee only).
* **KYT / AML Security:** Integrated Sarathi threat screening and address quarantine capabilities (`core/relay-chain/cybersecurity/cybercrime.go`).
