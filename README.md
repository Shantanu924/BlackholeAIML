# 🚀 BlackHole Blockchain

**A high-performance, multi-token blockchain with AI-powered fraud detection, TANTRA enforcement layer, and cross-chain bridge capabilities.**

## 🎯 **Goal: List BHIV Token on Cryptocurrency Exchanges**

**Current Status: PRODUCTION DEPLOYED — `https://vajra.blackholeinfiverse.com`**

> ✅ VAJRA Phase 1-5 complete. Constitutional runtime convergence validated. Live signed transactions, truth store, and AKASHIC lineage verified on production VM.

---

## ⚡ **Quick Start**

### **Production (Live)**
- **Dashboard:** https://vajra.blackholeinfiverse.com
- **Blockchain API:** https://vajra.blackholeinfiverse.com/api/
- **Web Wallet:** https://vajra.blackholeinfiverse.com/wallet/
- **Bridge Dashboard:** https://vajra.blackholeinfiverse.com/bridge/
- **Fraud Detection:** https://vajra.blackholeinfiverse.com/defence/

### **Local Dev**
```bash
# Start all services
docker compose -f docker-compose.yml up -d

# Check status
docker compose ps

# View logs
docker compose logs -f
```

---

## 🔥 **Key Features**

### **✅ Multi-Token Support**
- **BHIV** - Native blockchain token
- **ETH** - Ethereum integration
- **USDT** - Stablecoin support
- **SOL** - Solana integration

### **✅ TANTRA Enforcement Layer** (NEW)
- **Triple-hash validation** - ExecutionAgent, ValidationAgent, ReplayAgent
- **Fraud service integration** - Real-time transaction verification
- **Hard rejection on mismatch** - HTTP 403 for failed validation
- **Truth store** - Append-only audit trail (tantra_truth.jsonl)
- **Blockchain verification** - Phase 5 on-chain validation
- **Trace ID injection** - Full transaction traceability

### **✅ AI-Powered Fraud Detection**
- Real-time transaction monitoring
- Suspicious wallet flagging
- ML-based pattern recognition
- Automatic transaction blocking
- Integration with TANTRA enforcement

### **✅ Cross-Chain Bridge**
- Ethereum ↔ BlackHole transfers
- Real-time event listening
- Secure cross-chain operations
- Multi-chain token support
- Replay protection with BoltDB
- Circuit breaker patterns

### **✅ DEX & OTC Trading**
- Decentralized exchange functionality
- OTC trade verification
- Multi-signature support
- Liquidity pool management
- Slippage protection

### **✅ Professional Web Wallet**
- Browser-based interface
- Multi-token management
- Transaction history
- Real-time balance updates
- Faucet integration

### **✅ High Performance**
- 1000+ TPS capability
- 6-second block times
- Concurrent processing
- Optimized storage (LevelDB)
- P2P networking with peer discovery

---

## 📊 **System Architecture**

```
┌─────────────────┐    ┌─────────────────────────────────────┐    ┌─────────────────┐
│   Web Wallet    │    │         Blockchain Core             │    │  Fraud Service  │
│   (Port 3001)   │    │         (Port 8080)                 │    │   (Port 9090)   │
└─────────────────┘    │                                     │    └─────────────────┘
         │             │  ┌──────────────────────────────┐  │             │
         │             │  │   TANTRA Enforcement Layer   │  │             │
         │             │  │  - ExecutionAgent            │  │             │
         │ ── TX ─────▶│  │  - ValidationAgent ──────────┼──┼─── Check ──▶
         │             │  │  - ReplayAgent               │  │             │
         │             │  │  - Hash Equality Gate        │◀─┼── Allow/Block
         │             │  │  - Truth Store               │  │             │
         │             │  └──────────────────────────────┘  │
         │◀── Result ──│                                     │
         
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Token Faucet   │    │  Cross-Chain    │    │   Bridge SDK    │    │   External      │
│   (Port 3002)   │    │     Bridge      │    │   (Port 8084)   │    │    Chains       │
└─────────────────┘    └─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │                       │
         │ ── Token Requests ────▶                       │                       │
         │                       │ ── Bridge TXs ───────▶                       │
         │                       │                       │ ── Listen Events ────▶
         │                       │                       │◀── ETH/SOL Events ───│
```

---

## 🛠️ **Development**

### **Prerequisites**
- Go 1.19+
- Node.js 16+
- Git

### **Build from Source**
```bash
# Clone repository
git clone <repository-url>
cd blackhole-blockchain

# Build blockchain
cd core/relay-chain/cmd/relay
go build -o relay.exe

# Build wallet service
cd services/wallet
go build -o wallet.exe

# Build faucet service
cd services/validator-faucet
go build -o faucet.exe
```

### **Testing Against Production**
```bash
# Health check
curl https://vajra.blackholeinfiverse.com/api/health

# All 4 participants reachable
curl https://vajra.blackholeinfiverse.com/api/vajra/contracts

# Generate keypair
curl -X POST https://vajra.blackholeinfiverse.com/api/wallet/keygen

# Verify TANTRA transaction
curl "https://vajra.blackholeinfiverse.com/api/tantra/verify?tx_hash=<tx_id>"

# AKASHIC lineage
curl "https://vajra.blackholeinfiverse.com/api/akashic/lineage?tx_hash=<tx_id>"
```

---

## 📚 **Documentation**

### **Core Documentation**
- **[API Documentation](docs/API_DOCUMENTATION.md)** - Complete API reference (production URLs)
- **[Runtime Guide](docs/RUNTIME_GUIDE.md)** - TANTRA pipeline, PDV, AKASHIC, authority boundaries
- **[Production Deployment](docs/PRODUCTION_DEPLOYMENT_GUIDE.md)** - Docker/CI/CD deployment guide
- **[VAJRA API Contracts](VAJRA_API_CONTRACTS.md)** - Constitutional contracts for all 4 participants

### **Review Packets**
- **[Phase 4+5 Review Packet](review-packets/PHASE4_5_REVIEW_PACKET.md)** - Live evidence, certification
- **[Sprint Review Packet](review-packets/REVIEW_PACKET.md)** - Phase 1+2 completion

### **Bridge & Cross-Chain**
- **[Bridge SDK Documentation](bridge-sdk/docs/README.md)** - SDK usage

### **Wallet & Services**
- **[Wallet Web UI Guide](docs/WALLET_WEB_UI_GUIDE.md)** - Web wallet usage

---

## 🏦 **Exchange Listing Progress**

### **✅ Completed Requirements**
- [x] Multi-token blockchain (BHIV, ETH, USDT, SOL)
- [x] TANTRA enforcement layer (100% complete)
- [x] AI fraud detection (integrated with TANTRA)
- [x] Cross-chain bridge (90% complete)
- [x] Professional API (REST + gRPC)
- [x] Web wallet interface
- [x] DEX & OTC trading
- [x] Performance optimization
- [x] Docker deployment
- [x] Monitoring & alerting

### **🟡 Remaining Tasks**
- [ ] PDV agents as separate containers (currently single-process fallback)
- [ ] Cross-node nonce coordinator deployment
- [ ] Wallet /register path routing fix
- [ ] 1000+ TPS stress testing
- [ ] Security audit report
- [ ] Token whitepaper completion

---

## 🤝 **Team**

### **Blockchain Core (BHC)**
- **Shivam** - Token logic, approvals, event emission, core blockchain
- **Nihal** - DEX engine, OTC trading, pool health, swap mechanics
- **Sairam** - Staking core, validator logic, reward flows
- **Jay** - Wallet service, key management, validator onboarding

### **Cybercrime / AI Enforcement (AIxLaw)**
- **Prakash Yadav** - TANTRA enforcement integration (ExecutionAgent, ValidationAgent, ReplayAgent)
- **Keval** - Cybercrime smart contract lead, freeze/unfreeze logic
- **Aryan** - Event hooking, logging, multi-chain freeze logic
- **Yashika** - ML audit, violation detection, pattern recognition

### **Cross-Chain & Integration**
- **Shantanu** - Bridge SDK, cross-chain sync, relay infrastructure

### **DevOps & Quality**
- **Vinayak** - Docker, PRs, GitOps, integration testing, code review

---

## 📄 **License**

MIT License - See LICENSE file for details

---

## ⚠️ **Known Gaps**

1. **PDV single-process fallback** — `PDV_STRICT_MODE=false`, agents run inside node process
2. **Nonce coordinator not cross-node** — per-node nonce store only
3. **Wallet /register redirect** — goes to `/register` instead of `/wallet/register`
4. **state_root_hash empty** — AKASHIC entries have empty state root

See [Phase 4+5 Review Packet](review-packets/PHASE4_5_REVIEW_PACKET.md) for full details.

---

## 🚀 **Get Started**

1. **Visit https://vajra.blackholeinfiverse.com** — Live production dashboard
2. **Read [Runtime Guide](docs/RUNTIME_GUIDE.md)** — Understand TANTRA pipeline
3. **Read [API Documentation](docs/API_DOCUMENTATION.md)** — Submit transactions
4. **Read [Phase 4+5 Review Packet](review-packets/PHASE4_5_REVIEW_PACKET.md)** — Full evidence

**PRODUCTION DEPLOYED — Constitutional Runtime Convergence Complete** ✅
