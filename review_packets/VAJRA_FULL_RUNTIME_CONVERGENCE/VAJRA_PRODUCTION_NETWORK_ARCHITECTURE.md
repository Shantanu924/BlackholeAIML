# VAJRA Production Network Architecture & Reality Proof

## 1. Network Classification
The VAJRA Mainnet is formally classified as a **Public Permissioned** network. 
- **Public:** The RPC endpoints (`rpc.vajra.blackholeinfiverse.com`) are accessible over the public internet, allowing sovereign wallet users to broadcast transactions, query balances, and verify blocks without VPN access.
- **Permissioned:** The consensus nodes (Validators) are strictly authenticated via the Peer Identity System. Only cryptographically whitelisted validator nodes can participate in block generation and Proof-of-Authority (PoA) consensus.

## 2. Infrastructure Topology (AWS ECS Fargate)
The physical infrastructure is provisioned using Terraform (`deploy/mainnet/main.tf`) targeting AWS. 

### 2.1 Core Network Layers
* **VPC:** Deployed in a dedicated `10.0.0.0/16` AWS Virtual Private Cloud.
* **Public Subnets:** Hosts the Nginx Reverse Proxy and API Gateways.
* **Private Subnets:** Hosts the 5-Node Validator Cluster, preventing direct P2P port scanning from the public internet.

### 2.2 Containerized Services
* **Node 1-5 (Validators):** Deployed as Fargate tasks running the `vajranode/relay-chain` image. They communicate via P2P over port `9000`.
* **Bridge Service:** Handles Ethereum/Public-Chain interoperability.
* **Fraud Service:** The AI-driven anomaly detection microservice.
* **Wallet Service:** The sovereign UI backend.
* **MongoDB & BadgerDB:** Stateful persistence volumes mounted to the container instances for ledger storage and encrypted key vaults.

## 3. Public RPC Activation
The primary interface for external TANTRA systems and Wallet clients is the Public RPC Gateway.

* **Endpoint:** `https://rpc.vajra.blackholeinfiverse.com`
* **DNS:** Managed via Route53, pointing to the AWS Application Load Balancer.
* **TLS:** Enforced via Let's Encrypt / AWS Certificate Manager.
* **Rate Limiting:** Enforced at the Nginx layer (max 100 req/sec per IP) to prevent DDoS attacks against the execution engine.
* **Functionality:** Exposes the JSON-RPC interface for querying blocks (`/api/blocks`), balances (`/api/balance`), and broadcasting signed transactions (`/api/transaction`).

## 4. Failure Handling & Disaster Recovery
* **High Availability:** The 5-node cluster requires a strict $N > \frac{2}{3}F$ active quorum. The network tolerates the catastrophic failure of up to 1 validator node without halting.
* **Automated Rollback:** Daily block snapshots are stored in AWS S3 (`blackhole_backup.tar`). In the event of catastrophic data corruption, `scripts/rollback.sh` restores the network state to the last verified block height.
* **SSO Fallback:** As defined in the Security Model, if the central Auth server is offline, the localized BadgerDB vaults maintain 100% wallet uptime.
