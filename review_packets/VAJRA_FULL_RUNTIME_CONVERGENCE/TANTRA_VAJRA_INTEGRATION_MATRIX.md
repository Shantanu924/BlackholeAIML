# TANTRA & VAJRA Integration Matrix

This matrix explicitly defines the authority boundaries and live connections between the VAJRA Trust Layer and the surrounding TANTRA Constitutional Ecosystem.

## 1. TMS (TANTRA Management System)
* **Status:** ✅ INTEGRATED
* **VAJRA-side Owner:** Prakash
* **Adjacent Owner:** TMS Owner
* **Integration Rule:** TMS governs the ecosystem strategy and dictates VAJRA's placement. VAJRA implements this role as the sole permanent settlement layer for cross-chain execution.
* **Evidence:** VAJRA is integrated into the core architecture definitions as the non-bypassable settlement layer.

## 2. GC (Governance Council)
* **Status:** ✅ INTEGRATED
* **VAJRA-side Owner:** Prakash
* **Adjacent Owner:** GC Owner
* **Integration Rule:** VAJRA enforces legitimacy restrictions, but does not self-authorize. The Fraud Service delegates legitimacy verification to the GC before executing massive anomaly rollbacks.
* **Evidence:** The Fraud/Security AI service requires secondary human/GC approval for any permanent slashing event against a validator.

## 3. MDU (Master Data Universe)
* **Status:** ✅ INTEGRATED
* **VAJRA-side Owner:** Prakash
* **Adjacent Owner:** MDU Owner
* **Integration Rule:** MDU owns the schemas. VAJRA implements strictly versioned semantic identities in its JSON-RPC structure.
* **Evidence:** VAJRA RPC API matches the MDU v1 canonical transactional schema exactly (Sender, Recipient, Signature, Nonce, AssetID).

## 4. Bucket (Evidence Store)
* **Status:** ✅ INTEGRATED
* **VAJRA-side Owner:** Prakash
* **Adjacent Owner:** Bucket Owner
* **Integration Rule:** VAJRA records cryptographic trust records (block hashes). Bucket stores the canonical off-chain evidence (e.g., KYC files or dispute records) linked via IPFS/CID to the VAJRA block.
* **Evidence:** The `TANTRA_BUCKET_URL` environment variable is actively wired in the mainnet ECS task definition (`deploy/mainnet/main.tf`).

## 5. Replay (Simulation & Traceability)
* **Status:** ✅ INTEGRATED
* **VAJRA-side Owner:** Prakash
* **Adjacent Owner:** Replay Owner
* **Integration Rule:** VAJRA generates deterministic state transitions that can be exactly replayed.
* **Evidence:** `nonce_ledger.jsonl` and structured execution logs allow the Replay Engine to reconstruct exactly who signed what transaction, when, and with what evidence.

## 6. InsightFlow / Pravah (Observability)
* **Status:** ✅ INTEGRATED
* **VAJRA-side Owner:** Prakash
* **Adjacent Owner:** Pravah Owner
* **Integration Rule:** VAJRA must emit live telemetry.
* **Evidence:** The AWS ECS deployment utilizes AWS CloudWatch (`awslogs` log driver in `main.tf`) which is ingested into the central InsightFlow Grafana dashboards for health and metric tracing.

## 7. Constitutional Registries
* **Status:** ✅ INTEGRATED
* **Integration Rule:** VAJRA participates in all relevant canonical registries.
* **Evidence:** 
  * `Runtime Registry`: VAJRA is registered as the Primary Ledger.
  * `Execution Registry`: All transactions are recorded.
  * `Evidence Registry`: Block hashes are pushed to Bucket.
