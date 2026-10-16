# VAJRA Mainnet: Centralized Exchange (CEX) Integration Spec
*(Target: Binance / Tier-1 Exchanges)*

This document outlines the standard API integration pathways for a Centralized Exchange to securely list, process deposits, and execute withdrawals for the **BHIV** (BlackHole Infiverse) native token on the VAJRA Mainnet.

## 1. Network Constants
*   **Network Name:** VAJRA Mainnet
*   **Chain ID:** `vajra-mainnet`
*   **Native Token:** BHIV
*   **Decimals:** 18
*   **Consensus:** DPoS (Delegated Proof of Stake)
*   **Block Time:** ~3 seconds

## 2. API Endpoints (RPC)
The exchange should connect to the high-availability RPC gateway defined in our deployment topology:
**RPC Endpoint:** `https://rpc.vajra.blackholeinfiverse.com/api/`

## 3. Deposit Architecture (Exchange Receiving BHIV)
To credit users for BHIV deposits, the exchange must continuously scan the VAJRA blockchain for incoming transactions to the exchange's hot/cold wallet addresses.

### 3.1. Polling for Blocks
Exchanges should poll for finalized blocks and parse the transactions.
```http
GET /api/blocks/latest
GET /api/blocks/{height}
```
**Response Format:**
```json
{
  "success": true,
  "data": {
    "height": 1459203,
    "hash": "0xabc...",
    "transactions": [
      {
        "tx_id": "0x123...",
        "type": "TRANSFER",
        "from": "user_address",
        "to": "exchange_hot_wallet_address",
        "token_symbol": "BHIV",
        "amount": "150000000000000000000", // 150 BHIV (in wei)
        "status": "CONFIRMED"
      }
    ]
  }
}
```
*Note: A transaction should be considered final after 12 block confirmations (approx 36 seconds).*

## 4. Withdrawal Architecture (Exchange Sending BHIV)
When a user requests a withdrawal, the exchange's backend must construct, sign, and broadcast a transfer transaction to the VAJRA network.

### 4.1. Checking Hot Wallet Balance
```http
POST /api/wallets/balance
Content-Type: application/json

{
  "wallet_name": "exchange_hot_wallet",
  "password": "hot_wallet_password",
  "token_symbol": "BHIV"
}
```

### 4.2. Broadcasting a Withdrawal
```http
POST /api/wallets/transfer
Content-Type: application/json

{
  "wallet_name": "exchange_hot_wallet",
  "password": "hot_wallet_password",
  "to_address": "user_destination_address",
  "token_symbol": "BHIV",
  "amount": 50000000000000000000 // 50 BHIV (in wei)
}
```
*Security Note: For Tier-1 exchanges, we recommend utilizing the **gRPC interface** (`port 50051`) with offline transaction signing, rather than relying on the HTTP REST API.*

## 5. Offline Signing & Cold Storage (Recommended)
For cold storage, exchanges should use the VAJRA core SDK (`bridge-sdk`) to construct and sign transactions completely offline. The raw signed hexadecimal transaction can then be broadcasted to the network via:

```http
POST /api/transactions/broadcast
Content-Type: application/json

{
  "signed_tx_hex": "0xf86c098504a817c8008252089435353535353535353535353535353535353535..."
}
```

## 6. Smart Contract / Bridge Mappings
If the exchange wishes to support wrapped BHIV (`wBHIV`) on Ethereum, they must interact with the canonical ERC-20 contract address:
*   **Ethereum wBHIV Contract:** `0x[ERC20_CONTRACT_PENDING]`
*   **Decimals:** 18

(Wrapping and unwrapping between native BHIV and ERC-20 wBHIV is handled automatically by the VAJRA Cross-Chain Bridge).
