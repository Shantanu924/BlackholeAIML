# BlackHole Blockchain — API Documentation

**Production Domain:** `https://vajra.blackholeinfiverse.com`  
**Last Updated:** 2026-08-07  
**Schema Version:** v1

---

## Base URLs

| Service          | Production URL                                    | Local Dev         |
|------------------|---------------------------------------------------|-------------------|
| Blockchain Node  | `https://vajra.blackholeinfiverse.com/api/`       | `http://localhost:8080` |
| Wallet Service   | `https://vajra.blackholeinfiverse.com/wallet/`    | `http://localhost:9000` |
| Bridge SDK       | `https://vajra.blackholeinfiverse.com/bridge/`    | `http://localhost:8084` |
| Fraud Detection  | `https://vajra.blackholeinfiverse.com/defence/`   | `http://localhost:9090` |

---

## TANTRA Transaction Pipeline

All transactions submitted via `POST /api/relay/submit` pass through the canonical TANTRA pipeline:

```
Schema Validation → Nonce Governance → Signature Verification →
Distributed PDV → Fraud Gate (Sarathi) → Blockchain Write →
Truth Store → AKASHIC Lineage
```

### Submit Transaction

```http
POST /api/relay/submit
Content-Type: application/json

{
  "schema_version": "v1",
  "trace_id":       "optional-client-trace-id",
  "type":           "token_transfer",
  "from":           "<compressed-secp256k1-pubkey-hex>",
  "to":             "<recipient-address>",
  "amount":         100,
  "token_id":       "BHIV",
  "nonce":          1,
  "timestamp":      1786101903,
  "signature":      "<btcec-ecdsa-der-hex>"
}
```

**Response (success):**
```json
{
  "success":          true,
  "trace_id":         "phase4evidence01",
  "transaction_id":   "8dd91efe07ba3fd5d8ddf4103ef3f7181ca06fee3e78da30abfabf0aec32e8f8",
  "schema_version":   "v1",
  "signature_valid":  true,
  "fraud_decision":   "allow",
  "execution_hash":   "15f38dba...",
  "validation_hash":  "15f38dba...",
  "replay_hash":      "15f38dba...",
  "payload_hash":     "1e957454...",
  "block_height":     1,
  "status":           "pending",
  "submitted_at":     1786101903
}
```

**Error codes:**
| Code               | Meaning                                      |
|--------------------|----------------------------------------------|
| `SCHEMA_VIOLATION` | Missing/invalid fields                       |
| `SIGNATURE_REJECT` | Invalid btcec signature or pubkey mismatch   |
| `NONCE_REPLAY`     | Nonce already used by this address           |
| `BLOCKCHAIN_REJECT`| Insufficient balance or state error          |
| `FRAUD_BLOCK`      | Transaction blocked by fraud detection       |

---

## Wallet Signing Endpoints

### Generate Keypair

```http
POST /api/wallet/keygen
```

**Response:**
```json
{
  "success":     true,
  "address":     "<compressed-pubkey-hex>",
  "public_key":  "<compressed-pubkey-hex>",
  "private_key": "<32-byte-hex>",
  "warning":     "Store private_key securely. It will not be shown again."
}
```

### Sign Transaction Payload

```http
POST /api/wallet/sign
Content-Type: application/json

{
  "private_key": "<32-byte-hex>",
  "trace_id":    "my-trace-id",
  "type":        "token_transfer",
  "from":        "<compressed-pubkey-hex>",
  "to":          "<recipient>",
  "amount":      100,
  "token_id":    "BHIV",
  "nonce":       1
}
```

**Response:**
```json
{
  "success":      true,
  "signature":    "<btcec-ecdsa-der-hex>",
  "payload_hash": "<sha256-hex>"
}
```

> Note: `timestamp` is excluded from the signature hash — include it only in the submit body.

---

## VAJRA Identity & Contracts

### Runtime Identity Card

```http
GET /identity
```

Returns the service's constitutional identity — authority, non-authority, adjacent participants.

### VAJRA Contracts Probe

```http
GET /api/vajra/contracts
```

Probes all 4 participant `/identity` endpoints and returns reachability + identity for each.

---

## TANTRA Audit Endpoints

### Verify Transaction

```http
GET /api/tantra/verify?tx_hash=<transaction_id>
```

**Response:**
```json
{
  "success": true,
  "result": {
    "found":        true,
    "on_chain":     true,
    "hashes_match": true,
    "record": {
      "trace_id":        "...",
      "execution_hash":  "...",
      "validation_hash": "...",
      "replay_hash":     "...",
      "fraud_decision":  "allow",
      "prev_hash":       "genesis",
      "entry_hash":      "..."
    }
  }
}
```

### TANTRA Records

```http
GET /api/tantra/records
```

### AKASHIC Lineage

```http
GET /api/akashic/lineage?tx_hash=<transaction_id>
```

Returns the full immutable append-only chain of entries for a transaction.

---

## Blockchain Info Endpoints

### Health

```http
GET /api/health
```

### Blockchain State

```http
GET /api/blockchain/info
```

### Node Info

```http
GET /api/node/info
```

### P2P Info

```http
GET /api/p2p/info
```

### Balance

```http
GET /api/balance?address=<address>&token=<token_id>
```

---

## Admin Endpoints

### Add Tokens (Dev/Admin only)

```http
POST /api/admin/add-tokens
Content-Type: application/json

{
  "address": "<address>",
  "token":   "BHIV",
  "amount":  1000
}
```

---

## Bridge Endpoints

```http
GET  /bridge/health
GET  /bridge/stats
GET  /bridge/status
GET  /bridge/transactions
POST /bridge/relay/eth
POST /bridge/relay/sol
GET  /bridge/identity
```

---

## Fraud Detection Endpoints

```http
POST /defence/api/fraud/check
GET  /defence/api/fraud/status
POST /defence/api/fraud/block
GET  /defence/health
GET  /defence/identity
```

---

## Wallet Service Endpoints

```http
POST /wallet/api/register
POST /wallet/api/login
GET  /wallet/api/wallets
POST /wallet/api/wallets/create
POST /wallet/api/wallets/transfer
GET  /wallet/api/wallets/balance
GET  /wallet/identity
```

---

## Error Response Format

```json
{
  "success":          false,
  "error_code":       "SCHEMA_VIOLATION",
  "rejection_reason": "schema violation: field=payload reason=...",
  "trace_id":         "fbdc62d3170c8f28"
}
```

## Success Response Format

```json
{
  "success": true,
  "data":    {},
  "message": "..."
}
```
