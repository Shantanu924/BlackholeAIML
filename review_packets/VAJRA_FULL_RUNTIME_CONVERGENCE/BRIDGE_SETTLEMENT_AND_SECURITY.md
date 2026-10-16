# VAJRA Bridge Settlement and Security Architecture

## 1. Overview
The VAJRA cross-chain bridge is responsible for secure asset locking, minting, and burning across connected chains (primarily Ethereum, Polkadot, and Solana). It operates on a multi-signature relay node consensus model.

## 2. Settlement Process (BlackHole to Ethereum)
1. **Initiation:** User submits a cross-chain transfer via the VAJRA Wallet, specifying the asset (e.g., BHIV) and destination address.
2. **Pre-Validation:** The Bridge Service verifies the user's balance and checks for valid smart contract allowances on the canonical token.
3. **Locking:** The native asset (BHIV) is locked into the `bridge_contract` on the VAJRA network (`TransferFrom`).
4. **Relay Consensus:** At least 2 out of 3 Relay Nodes must independently verify the locking transaction and provide cryptographic signatures.
5. **Destination Minting:** The signed payload is submitted to the EVM target contract, which verifies the signatures and mints wrapped tokens (e.g., eBHIV) to the user's Ethereum address.

## 3. Real Provider Connection & Contract Identity
* **Ethereum RPC Provider:** Uses the external provider defined via `ETH_RPC_URL` for real mainnet/testnet verification.
* **Token Mappings:**
  * Native BHIV -> `0x[ERC20_CONTRACT_PENDING]`
  * USDT -> `0xdAC17F958D2ee523a2206206994597C13D831ec7` (Ethereum Mainnet)

## 4. Security & Replay Protection
* **Nonce Tracking:** Every bridge transaction contains a strictly increasing nonce per user to prevent duplicate payload submission.
* **Trace IDs:** All bridge transactions inherit the canonical `trace_id` for end-to-end replayability in `tantra_truth.jsonl`.
* **Multi-Signature Verification:** The destination smart contract natively implements `ecrecover` to guarantee that tokens cannot be minted without explicit relay node quorum.
* **Failure Recovery:** If a bridge transaction fails to confirm on the destination chain within the TTL, the locked assets can be unlocked on the source chain by submitting a cryptographically proven failure receipt.
