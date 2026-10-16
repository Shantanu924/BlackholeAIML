# VAJRA RPC Operations & Connectivity Proof

## 1. RPC Endpoint Definition
The VAJRA Trust Layer exposes its state and transaction broadcasting capability via a unified Public RPC Gateway.

* **Primary Endpoint:** `https://rpc.vajra.blackholeinfiverse.com`
* **Protocol:** HTTP/1.1 and HTTP/2 (TLS 1.3 enforced)
* **Format:** JSON REST (and JSON-RPC 2.0 compatible wrappers)

## 2. Supported RPC Methods
The following core methods are publicly accessible without SSO authentication (as they query public ledger state):

### 2.1 Block Verification
* **Endpoint:** `GET /api/blocks`
* **Response:** Returns the canonical chain of finalized blocks.
* **Usage:** Used by external Block Explorers and cross-chain relayers to verify block finality.

### 2.2 Balance Query
* **Endpoint:** `GET /api/balance?wallet={address}`
* **Response:** Returns the confirmed integer balance of native BHIV/VAJRA tokens.

### 2.3 Transaction Broadcasting
* **Endpoint:** `POST /api/transaction`
* **Payload:** A cryptographically signed transaction object (Sender, Recipient, Amount, Signature, Nonce).
* **Validation:** The RPC node immediately verifies the elliptic curve signature (`secp256k1`). If invalid, the RPC rejects the request before it enters the mempool.

## 3. Operational Hardening
### Rate Limiting
To prevent resource exhaustion, the Nginx reverse proxy enforces:
- Global limit: 10,000 req/min
- Per-IP limit: 100 req/sec

### TLS & DNS Validation
- **DNS Record:** An `A` record points `rpc.vajra.blackholeinfiverse.com` to the AWS Application Load Balancer.
- **TLS:** Certificates are automatically rotated every 60 days via Let's Encrypt (`certbot`).

## 4. Failure Handling
If the primary RPC node (`blackhole-node-1`) goes offline, the Load Balancer instantly routes traffic to the healthy replicas (`blackhole-node-2` or `blackhole-node-3`). Clients will experience zero downtime.
