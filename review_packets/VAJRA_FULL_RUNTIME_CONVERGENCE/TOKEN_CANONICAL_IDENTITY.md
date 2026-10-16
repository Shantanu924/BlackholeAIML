# TOKEN CANONICAL IDENTITY & PARAMETER SPECIFICATION

**Document Version:** 1.0.0  
**Asset Name:** BlackHole Infiverse (`BHIV`)  
**Audit Standard:** Code-Verified Ground Truth (Rules 0.1 – 0.4)

---

## 1. Native Token Specification (VAJRA Relay Chain)

| Parameter | Specification | Code Evidence |
|---|---|---|
| **Token Name** | BlackHole Infiverse | `core/relay-chain/chain/blockchain.go:272` |
| **Token Symbol** | `BHIV` | `core/relay-chain/chain/blockchain.go:272` |
| **Decimals** | 18 | `core/relay-chain/token/token.go:33` |
| **Maximum Supply Cap** | **1,000,000,000 BHIV** ($10^9$) | `core/relay-chain/token/token.go:37` |
| **Canonical Asset ID** | `0x0000000000000000000000000000000000000000_BHIV` | `core/relay-chain/chain/blockchain.go:274` |
| **Network ID** | `vajra-mainnet` | `core/relay-chain/chain/blockchain.go:273` |
| **Genesis Treasury Allocation** | 100,000,000 BHIV to `system-treasury` | `scripts/genesis_config.json:8` |
| **Genesis Validator Allocation** | 10,000 BHIV to `genesis-validator-1..3` | `scripts/genesis_config.json:13-24` |
| **Block Reward Allocation** | 10 BHIV base per minted block | `core/relay-chain/consensus/pos.go:38` |
| **Dynamic Inflation Reduction** | Starts at 50% supply, scales down up to 80% | `core/relay-chain/consensus/pos.go:154-165` |
| **Minting Authority** | `adminAddresses["system"] = true` only | `core/relay-chain/token/mint.go:42` |
| **Burning Authority** | Token holder self-burn with balance check | `core/relay-chain/token/burn.go:36` |
| **Transfer Controls** | Global Pause (`paused`) & Emergency Mode (`emergencyMode`) | `core/relay-chain/token/token.go:25-26` |

---

## 2. EVM Testnet Token Deployments

### 2.1 Polygon Amoy Testnet Deployment
* **Network:** Polygon Amoy Testnet (Chain ID `80002`)
* **Contract Address:** `0x64Dd946E7425898FE8FaB6D36eA050071190be40`
* **Deployment Transaction:** `0x17063f75bbbd7833ac3962e1065671f3921dabb082617b325c9043d7c86ee690`
* **Deployer Address:** `0xe5C146a2FCD0844818604a74Cc75F2a4aE25579c`
* **Initial Supply Minted on Amoy:** 10,000,000 BHIV
* **Maximum Supply Cap:** 1,000,000,000 BHIV
* **Amoy Explorer Link:** https://amoy.polygonscan.com/address/0x64Dd946E7425898FE8FaB6D36eA050071190be40
* **Contract Standard:** OpenZeppelin ERC20 (`BHXToken.sol` in `contracts/BHXToken.sol`)

### 2.2 Public Mainnet Status
> [!IMPORTANT]
> **No live Ethereum Mainnet or BNB Chain Mainnet contract is currently deployed.**  
> The genesis bridge configuration explicitly marks Ethereum mainnet as `0xTBD_MAINNET_wBHIV` until production exchange liquidity and regulatory requirements are finalized.

---

## 3. Bridged External Assets Supported on Relay Chain

| Asset Symbol | Asset Name | Decimals | Max Supply | Canonical ID / Origin Contract | Network ID |
|---|---|---|---|---|---|
| **ETH** | Ethereum | 18 | 100,000,000 | `0x0000000000000000000000000000000000000000` | `ethereum-mainnet` |
| **USDT** | Tether USD | 6 | 1,000,000,000 | `0xdAC17F958D2ee523a2206206994597C13D831ec7` | `ethereum-mainnet` |

---

## 4. Supply Invariant & Audit Rule
* The token implementation includes an independent supply auditing function `ValidateSupply()` (`core/relay-chain/token/mint.go:95`) which validates that:
  $$\sum_{\text{all accounts}} \text{balance}_i \equiv \text{totalSupply} \le \text{maxSupply}$$
* Any minting operation that causes total supply to exceed $1,000,000,000$ hard-fails with `mint amount would exceed maximum supply`.
