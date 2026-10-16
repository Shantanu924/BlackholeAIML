# VAJRA Mainnet Operational Handover Document

**Document Specification:** Formal Engineering & Operations Handover  
**From:** Prakash Kumar (VAJRA Blockchain Core, Mainnet & Production Runtime Owner)  
**To:** SRE, Node Operators & TANTRA Operational Governance Council  
**Constitutional Domain:** Trust Layer / Settlement Substrate  
**Status:** Production Certified  

---

## 1. Executive Handover Summary

This document formally hands over the operational stewardship of the **VAJRA Sovereign Blockchain and Trust Substrate**.

As of commit `355cfc0`, the core blockchain implementation is production-hardened, fully verified across 26 Go packages (78 tests) and 45 Python security tests (100% pass rate), protected by global panic recovery middleware, and integrated with the TANTRA governance ecosystem.

---

## 2. Component Ownership Handover Matrix

| Subsystem / Domain | Lead Custodian | Primary Contact / Role | Responsibilities Transferred |
|---|---|---|---|
| **Consensus & Relay Chain** | Prakash Kumar | Blockchain Core Lead | Core Go code, PoS algorithms, block mining triggers, token supply invariants. |
| **Node Topology & P2P** | SRE Lead | Infrastructure Lead | Docker cluster maintenance, host firewall ports (`30303`–`30307`), bootnode uptime. |
| **Public Gateway & RPC** | DevOps Team | Web3 Gateway Lead | Nginx reverse proxy, Let's Encrypt SSL/TLS certs, Cloudflare DDoS policies. |
| **Storage & Backups** | DBA / SRE | Storage Custodian | LevelDB directory maintenance (`blockchaindb_*`), daily snapshots via `scripts/snapshot_recovery.sh`. |
| **Identity & SSO Auth** | Pravesh Team | Identity Cell | Pravesh SSO service on port 8087, cross-subdomain cookie configurations. |
| **Cybersecurity Freeze** | Security Cell | Incident Lead | Authorizing and emitting address freeze notices via `/api/cybercrime/freeze`. |
| **AI Intelligence & Sarathi** | Shantanu Bhukan | AI Lead | Sarathi decision engine, predictive fraud scoring, external webhook feeds. |
| **Production Support** | Ishan Shirode | Associate Blockchain Eng | Go codebase maintenance, node monitoring, release deployment support. |

---

## 3. Operational Secrets & Keys Custody

* **Validator Private Keys:** Stored in secure host directory `/opt/vajra/config/` (never committed to public git).
* **Governance RS256 Key Pair:**
  - Public Key: Exported via `/api/v1/auth/public-key` and `/api/v1/auth/jwks`.
  - Private Key: Injected at runtime via environment variable `JWT_PRIVATE_KEY_PATH`.
* **Database Volumes:** Persistent LevelDB instances mapped to host paths `blockchaindb_8080/` through `8084/`.

---

## 4. Critical Operational Rules (Dos and Don'ts)

### DO:
1. **Always verify node peer count:** Ensure each node reports `peer_count >= 1` via `/api/blockchain/info`.
2. **Execute daily snapshots:** Validate that `scripts/snapshot_recovery.sh` runs successfully at 02:00 UTC.
3. **Monitor the 3-Agent PDV Gate:** Check `tantra_truth.jsonl` to ensure $H_{\text{exec}} == H_{\text{val}} == H_{\text{replay}}$.
4. **Follow the versioned release procedure:** Always deploy via git tags and execute rollback procedures (`scripts/rollback.sh`) if `/api/health` reports degradation.

### DO NOT:
1. **Never alter token supply cap:** The 1 Billion `BHIV` cap in `core/relay-chain/token/token.go` is an immutable economic invariant.
2. **Never bypass ErrorBoundaryMiddleware:** All public API endpoints must remain shielded by panic recovery wrappers.
3. **Never allow unauthenticated address unfreezes:** Address unfreezes must carry cryptographic authorization from the Cybersecurity Cell.
4. **Never treat local tests as mainnet proof:** Always verify running Docker containers on the live network host.
