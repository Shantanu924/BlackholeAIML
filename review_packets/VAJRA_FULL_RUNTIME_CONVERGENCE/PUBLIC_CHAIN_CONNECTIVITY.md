# PUBLIC-CHAIN CONNECTIVITY & CROSS-NETWORK INTEROPERABILITY

**Document Version:** 1.0.0  
**Audit Standard:** Code-Verified Ground Truth (Rule 0.4: Never Fabricate External Integrations)

---

## 1. Public-Chain Connectivity Matrix

| Network | Chain ID | Verification State | RPC Endpoint | Bridge Asset Contract | Operational Status |
|---|---|---|---|---|---|
| **Polygon Amoy Testnet** | `80002` | `TESTNET_ONLY` | `https://rpc-amoy.polygon.technology` | `0x64Dd946E7425898FE8FaB6D36eA050071190be40` | Live Deployed Contract (`contracts/DEPLOYMENT_SUCCESS.md`) |
| **Ethereum Sepolia** | `11155111` | `TESTNET_ONLY` | `https://eth-sepolia.g.alchemy.com/v2/demo` | Configured in `bridge-sdk` | Testnet Event Listener Active |
| **Solana Devnet** | `N/A` (Devnet) | `TESTNET_ONLY` | `https://api.devnet.solana.com` | Configured in `bridge-sdk` | Testnet Event Listener Active |
| **BNB Smart Chain Testnet** | `97` | `TESTNET_ONLY` | `https://data-seed-prebtc-1-s1.binance.org:8545` | Hardhat Config (`contracts/hardhat.config.js`) | Testnet Deployment Scripts Ready |
| **Ethereum Mainnet** | `1` | `MISSING` | `NOT CONFIGURED` | `0xTBD_MAINNET_wBHIV` (Placeholder) | **NOT CONNECTED** (Staging only; awaiting production listing) |

---

## 2. Cross-Network Transfer Lifecycle

```
[ External EVM Chain (e.g. Polygon Amoy) ]
                    │
                    ▼ User deposits tokens to Bridge Contract
             Lock / Burn Event Emitted
                    │
                    ▼
          [ Bridge SDK Relayer ]
       • RPC Event Listener detects tx
       • Waits for 12 block confirmations
       • Records tx hash in bbolt DB (Replay Protection)
       • Attests event signatures via core/attest
                    │
                    ▼ HTTP POST /api/relay/submit
       [ VAJRA Relay Chain Node ]
       • Verifies relayer signature
       • Enforces sequential nonce in nonce_ledger.jsonl
       • Canonical 9-stage runtime execution
       • Mints wrapped asset / credits balance in LevelDB
       • Appends truth record to tantra_truth.jsonl
```

---

## 3. Security, Replay & Failure Recovery Guarantees

1. **Replay Prevention:**
   * Double-spend attempts are blocked across both boundaries:
     * **External Side:** `bridge-sdk` tracks processed transaction hashes inside an embedded `bbolt` Key-Value store (`bridge_v4.db`).
     * **Internal Side:** Relay chain enforces nonces via `noncestore.Store` (`nonce_ledger.jsonl`).
2. **Circuit Breakers:**
   * Both `bridge-sdk` and `core/relay-chain/dex` maintain stateful circuit breakers that trip and reject transfer routing if consecutive network RPC errors or deserialization failures exceed threshold limits.
3. **Confirmation Depth:**
   * Relayers require 12 confirmed blocks on EVM chains and 32 blocks on Solana before acknowledging transfer validity.
4. **Failure Recovery Queue:**
   * Unconfirmed or dropped relay packets are queued in `RetryQueue` with exponential backoff (`MaxRetries: 3`, `RetryDelayMs: 5000`).
