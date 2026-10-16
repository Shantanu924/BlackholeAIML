# VAJRA Mainnet: Genesis Bootstrapping Guide

This document defines the formal procedure for launching the VAJRA mainnet from scratch using a deterministic Genesis Block.

## 1. Genesis Configuration
The initial token distribution, genesis validator assignments, and cross-chain references are defined in `scripts/genesis_config.json`. 

By default, the token allocation logic is:
*   **System Treasury (`system-treasury`)**: 100,000,000 BHIV. (Used to fund ecosystem grants, community incentives, and exchange liquidity).
*   **Initial Validators (`genesis-validator-1`, etc.)**: 10,000 BHIV each. (Ensures initial DPoS consensus has enough stake to produce blocks before public delegation begins).

## 2. Bootstrapping Process
To initialize the Mainnet from zero state:

1.  **Clear Stale Data:** Ensure all `data/` and `storage/` directories are completely wiped across all validator nodes.
    ```bash
    rm -rf core/relay-chain/data/*
    rm -rf services/wallet/data/*
    ```
2.  **Load Configuration:** The node reads `genesis_config.json` on startup if the database is empty.
3.  **Genesis Minting:** The `createGenesisBlock()` function in `blockchain.go` parses the config, instantiates the Native `BHIV` asset in the `TokenRegistry`, and mints the initial balances to the target addresses without requiring transaction signatures.
4.  **Consensus Initialization:** The designated Genesis Validators parse Block 0, register their initial stakes in the `StakeLedger`, and begin the block production epoch (Block 1).

## 3. Verification
Once Node 1 boots, DevOps should verify the genesis supply via the RPC endpoint:
```bash
curl -X GET http://localhost:8080/api/status
```
Verify that the `Total Supply` matches the exact sum of allocations in `genesis_config.json`. No further tokens can be minted outside of standard DPoS inflationary block rewards.
