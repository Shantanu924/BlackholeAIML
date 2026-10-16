# WALLET CUSTODY & SECURITY MODEL SPECIFICATION

**Document Version:** 1.0.0  
**Status:** Code-Verified Custody & Security Architecture  
**Audit Standard:** Strict Transparency (Rule 0.2 & Step 1.4)

---

## 1. Custody Model Classification

> [!CAUTION]
> **Explicit Custody Declaration:**  
> The VAJRA Wallet backend (`services/wallet`) implements a **CUSTODIAL (Server-Side Custody with Database Persistence & Embedded Local Fallback)** architecture.  
> **Signing Key Access:** The backend infrastructure stores encrypted private keys in database collections and decrypts them into server memory to compute ECDSA signatures during transaction execution. It is **NOT** a client-side non-custodial wallet.

---

## 2. Cryptographic Key Lifecycle

```
 User Registration / Login
         │
         ▼ (Password + Random Salt)
   Argon2id KDF (Time: 3, Memory: 64MB, Threads: 4)
         │
         ├─────────────────────────────────────────┐
         ▼                                         ▼
   Password Hash                           AES-256 Encryption Key
  (Stored in DB)                                   │
                                                   ▼
   BIP-39 Mnemonic Generation               AES-256-GCM Encryption
         │                                         │
         ▼                                         ▼
   BIP-32 HD Key Derivation ──────────────► Encrypted Private Key
  (Secp256k1 / btcec/v2)                   (Stored in MongoDB/BadgerDB)
```

### 2.1 Key Generation
* **Mnemonic Phrase:** 12/24-word BIP-39 mnemonic generated using cryptographically secure entropy (`crypto/rand`).
* **HD Derivation:** BIP-32 root seed and derivation path for generating Secp256k1 private/public key pairs.
* **Key Format:** Compressed public keys (33 bytes) and raw 32-byte private keys managed via `github.com/btcsuite/btcd/btcec/v2`.

### 2.2 Key Encryption & Storage
* **Encryption Standard:** AES-256 in Galois/Counter Mode (**GCM**) with unique 12-byte initialization vectors (`nonce`) generated per encryption call.
* **Key Derivation (KDF):** User password and cryptographic salt are passed through **Argon2id** (`argon2.IDKey`) with 3 iterations, 64MB memory, 4 parallel threads, and 32-byte output.
* **Storage Engines:**
  * **Primary:** MongoDB `wallets` collection (fields: `encrypted_priv_key`, `encrypted_mnemonic`, `public_key`, `address`).
  * **Relational:** PostgreSQL 15 `wallets` table for institutional ledger integration.
  * **Local Fallback:** Embedded BadgerDB (`data/badger`) activated automatically if primary database connectivity fails.
  * **Master Key:** Local file fallback `data/master.key` for administrative recovery operations.

---

## 3. Transaction Signing & Execution Flow

1. **User Authentication:** User submits credentials to `POST /api/auth/login`. Upon password validation via constant-time comparison (`subtleCompare`), a signed JWT session token is returned.
2. **Transaction Intent:** Client submits transfer parameters (recipient, token symbol, amount) to `POST /api/wallet/transfer`.
3. **Key Decryption:** Backend decrypts the user's private key using the session-derived AES symmetric key.
4. **ECDSA Signing:** The transaction payload hash is signed using `btcec.PrivateKey.ToECDSA()` and `ecdsa.Sign()`, producing signature components $(r, s)$.
5. **Node Submission:** The signed transaction is serialized and broadcasted to the VAJRA Relay Chain via `POST http://blackhole-node-1:8080/api/relay/submit`.
6. **Confirmation & Indexing:** The transaction ID is tracked, state receipts are stored, and history is updated across MongoDB and PostgreSQL.

---

## 4. Security Controls & Defense Measures

* **Timing Attack Prevention:** Password hash verification utilizes `subtleCompare` constant-time byte comparisons (`services/wallet/wallet/wallet.go:182`).
* **Replay Protection:** Unique incremental nonces enforced by both the wallet service and the relay chain `noncestore` ledger (`nonce_ledger.jsonl`).
* **Rate Limiting:** IP and account-level rate limits configured in Nginx and Go HTTP middleware.
* **Sanction & Threat Filtering:** Transactions route through Sarathi pre-flight fraud check before blockchain submission.
