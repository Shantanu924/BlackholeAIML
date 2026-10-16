# TRADING & LIQUIDITY READINESS SPECIFICATION

**Document Version:** 1.0.0  
**Status:** Internal AMM / OTC Trading Implemented; External DEX Ready for Deployment  
**Audit Standard:** Code-Verified Ground Truth (Rules 0.1 – 0.4)

---

## 1. Trading Infrastructure Overview

| Subsystem | Implementation File | Verification State | Operational Features |
|---|---|---|---|
| **Native AMM DEX** | `core/relay-chain/dex/dex.go` | `PRODUCTION_INTERNAL_ONLY` | Constant product AMM ($x \cdot y = k$), LP share accounting, 0.3% fee rate, 5% max slippage tolerance. |
| **Cross-Chain DEX Router** | `core/relay-chain/dex/cross_chain_dex.go` | `TESTNET_ONLY` | Cross-chain swap routing between VAJRA native assets and bridged EVM/Solana tokens. |
| **OTC Trading Engine** | `core/relay-chain/otc/otc.go` | `PRODUCTION_INTERNAL_ONLY` | P2P order creation, matching, order cancellation, and trade settlement. |
| **Native Escrow Manager** | `core/relay-chain/escrow/escrow.go` | `PRODUCTION_INTERNAL_ONLY` | Atomic custody lock, conditional release, and automated refund triggers. |
| **External DEX Liquidity** | Uniswap / PancakeSwap / QuickSwap | `MISSING` | **No live liquidity pools exist.** External liquidity awaits mainnet asset deployment. |

---

## 2. Automated Market Maker (AMM) Mechanics

### 2.1 Pricing Formula & Invariant
* **Constant Product Invariant:**
  $$k = \text{Reserve}_A \times \text{Reserve}_B$$
* **Swap Calculation (with 0.3% fee):**
  $$\Delta y = \frac{\text{Reserve}_B \times \Delta x \times (1 - \text{Fee})}{\text{Reserve}_A + \Delta x \times (1 - \text{Fee})}$$
* **Slippage Enforcement:** If the calculated price impact exceeds `MaxSlippageThreshold` (5.0%), the swap transaction is rejected automatically (`core/relay-chain/dex/dex.go:68`).

### 2.2 Liquidity Pool Management
* **Pool Struct:** `LiquidityPool` tracks `ReserveA`, `ReserveB`, `TotalShares`, `FeeRate`, and `LastUpdated`.
* **LP Tokens:** Proportional shares minted to liquidity providers upon `AddLiquidity()`, burnt upon `RemoveLiquidity()`.

---

## 3. Over-the-Counter (OTC) & Escrow Engine

1. **Order Creation (`POST /api/otc/create`):** Maker specifies offer token/amount, requested token/amount, and counterparty constraint (public or direct address).
2. **Escrow Lock:** Offered tokens are transferred atomically into the `EscrowManager` contract.
3. **Matching & Execution (`POST /api/otc/match`):** Taker deposits requested funds, triggering atomic two-way settlement.
4. **Cancellation (`POST /api/otc/cancel`):** Unfilled orders return escrowed assets back to maker.

---

## 4. Market Readiness Roadmap

```
[ Internal AMM / OTC Ledger ] ──(Verified)──► [ Polygon Amoy Testnet DEX Testing ] ──► [ External Liquidity Pool Launch ]
   • Code: PASS                                   • Contract: 0x64Dd946E7425898FE...        • Target: Uniswap v3 / QuickSwap
   • Slippage: 5% Max                             • Test Swaps: Operational                 • Pair: BHIV / USDT
   • OTC Escrow: PASS                             • Test LP: Active                         • Status: Ready for Staging
```
