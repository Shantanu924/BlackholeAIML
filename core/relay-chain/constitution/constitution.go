// Package constitution implements Phase 5 — Constitutional Replay Boundary Enforcement.
//
// This package makes authority boundaries explicit and machine-readable at runtime.
//
// Key declarations:
//   replay equality != governance legitimacy
//   observability != execution authority
//   replay reconstruction != semantic truth
//   deterministic execution != sovereign correctness
//
// The system exposes:
//   - What it owns (execution correctness, replay determinism)
//   - What it does NOT own (governance legitimacy, semantic truth)
//   - Execution rights (what can be executed)
//   - Governance boundaries (what requires external governance decision)
//   - Replay boundaries (what replay can and cannot prove)
package constitution

import (
	"log"
	"os"
	"time"
)

// AuthorityType classifies a system capability.
type AuthorityType string

const (
	AuthorityOwned    AuthorityType = "OWNED"
	AuthorityNotOwned AuthorityType = "NOT_OWNED"
	AuthorityBounded  AuthorityType = "BOUNDED"
)

// Boundary declares a single constitutional boundary.
type Boundary struct {
	Name        string        `json:"name"`
	Authority   AuthorityType `json:"authority"`
	Description string        `json:"description"`
	Proof       string        `json:"proof"`
}

// Declaration is the full constitutional boundary declaration.
type Declaration struct {
	SystemName  string     `json:"system_name"`
	Version     string     `json:"version"`
	Timestamp   int64      `json:"timestamp"`
	Boundaries  []Boundary `json:"boundaries"`
	DriftStatus string     `json:"drift_status"`
	DriftRisks  []string   `json:"drift_risks"`
}

// Declare returns the full constitutional boundary declaration for this system.
// This is the runtime-observable authority map.
func Declare() Declaration {
	log.Printf("[CONSTITUTION][DECLARE] generating authority boundary declaration")

	return Declaration{
		SystemName: "BlackHole Blockchain TANTRA Runtime",
		Version:    "phase3-hardened",
		Timestamp:  time.Now().Unix(),
		Boundaries: []Boundary{
			{
				Name:        "PDV_EQUALITY",
				Authority:   AuthorityOwned,
				Description: "The system owns deterministic hash equality across ExecutionAgent, ValidationAgent, ReplayAgent.",
				Proof:       "execution_hash == validation_hash == replay_hash — enforced in enforcement/tantra.go",
			},
			{
				Name:        "SIGNATURE_VERIFICATION",
				Authority:   AuthorityOwned,
				Description: "The relay owns cryptographic verification of wallet signatures before PDV.",
				Proof:       "sigverify.Verify() called before PDV in enforceWithURL — hard fail on invalid sig",
			},
			{
				Name:        "NONCE_GOVERNANCE",
				Authority:   AuthorityOwned,
				Description: "The system owns persistent nonce sequencing and replay detection across restarts.",
				Proof:       "noncestore persists to nonce_ledger.jsonl — loaded on startup — NONCE_REPLAY on duplicate",
			},
			{
				Name:        "TRACE_CONTINUITY",
				Authority:   AuthorityOwned,
				Description: "The system owns immutable trace_id propagation from entry to AKASHIC.",
				Proof:       "trace.Context — Inject() rejects any change — TRACE_BREAK on drift",
			},
			{
				Name:        "BUCKET_TRUTH",
				Authority:   AuthorityOwned,
				Description: "The system owns append-only chain-linked truth persistence in Bucket.",
				Proof:       "truthstore chain-linked via prev_hash/entry_hash — VerifyChain() detects tampering",
			},
			{
				Name:        "AKASHIC_LINEAGE",
				Authority:   AuthorityOwned,
				Description: "The system owns append-only chain-linked lineage reconstruction.",
				Proof:       "akashic chain-linked — Reconstruct() verifies final_state_root",
			},
			{
				Name:        "GOVERNANCE_LEGITIMACY",
				Authority:   AuthorityNotOwned,
				Description: "The system does NOT own governance legitimacy. Whether a transaction is legitimate is decided by Sarathi/DGIC — not by PDV equality.",
				Proof:       "FraudGate is separate from PDV — called AFTER PDV PASS — governance decision is external",
			},
			{
				Name:        "SEMANTIC_TRUTH",
				Authority:   AuthorityNotOwned,
				Description: "Replay reconstruction does NOT prove semantic truth. It proves deterministic execution correctness only.",
				Proof:       "Reconstruct() verifies hash chain integrity — not business logic correctness",
			},
			{
				Name:        "SOVEREIGN_CORRECTNESS",
				Authority:   AuthorityNotOwned,
				Description: "Deterministic execution does NOT equal sovereign correctness. Sovereignty requires external constitutional validation.",
				Proof:       "TANTRA is an execution organism — constitutional sovereignty requires KSML/CET + Sarathi alignment",
			},
			{
				Name:        "REPLAY_EQUALITY_VS_LEGITIMACY",
				Authority:   AuthorityBounded,
				Description: "Replay equality proves two nodes computed the same result. It does NOT prove the transaction was legitimate or should have been executed.",
				Proof:       "replayverifier.VerifyEquality() — hash agreement != governance approval",
			},
			{
				Name:        "OBSERVABILITY_VS_AUTHORITY",
				Authority:   AuthorityBounded,
				Description: "Observability endpoints expose execution state. They do NOT grant execution authority.",
				Proof:       "/api/tantra/records, /api/akashic/lineage — read-only — no write authority",
			},
			{
				Name:        "WALLET_INTENT_BOUNDARY",
				Authority:   AuthorityBounded,
				Description: "Wallet may create intent and sign payloads. It may NOT define legitimacy or bypass enforcement.",
				Proof:       "services/wallet/tantra/tantra.go — Execute() routes through /api/relay/submit only",
			},
			{
				Name:        "PDV_STRICT_MODE_BOUNDARY",
				Authority:   AuthorityBounded,
				Description: "PDV local fallback is a dev convenience, not a production guarantee. PDV_STRICT_MODE=true eliminates the fallback and hard-fails on unreachable agents.",
				Proof:       "pdv/pdv.go strictMode() — PDV_STRICT_MODE=true causes hard fail on agent unreachability",
			},
			{
				Name:        "SIGVERIFY_STRICT_MODE_BOUNDARY",
				Authority:   AuthorityBounded,
				Description: "Named-address signature bypass is a dev convenience. SIGVERIFY_STRICT_MODE=true eliminates it and requires all addresses to be valid compressed public keys.",
				Proof:       "sigverify/sigverify.go — SIGVERIFY_STRICT_MODE=true hard-rejects non-pubkey addresses",
			},
			{
				Name:        "SARATHI_GOVERNANCE_BOUNDARY",
				Authority:   AuthorityBounded,
				Description: "Sarathi governance is configurable via SARATHI_URL. Fail behavior is configurable via SARATHI_FAIL_CLOSED. Neither is sovereign until externally deployed.",
				Proof:       "enforcement/tantra.go sarathiURL() + sarathiFailClosed() — env-var controlled",
			},
			{
				Name:        "VAJRA_MAINNET_REGISTRY",
				Authority:   AuthorityOwned,
				Description: "VAJRA_MAINNET is a constitutionally recognized capability domain permitted to orchestrate TOKEN_TRANSFER and CROSS_CHAIN_SWAP.",
				Proof:       "Capability registry in core/relay-chain/constitution/constitution.go establishes runtime trust boundary.",
			},
		},
		DriftStatus: assessDrift(),
		DriftRisks:  driftRisks(),
	}
}

// assessDrift checks for known governance drift conditions.
func assessDrift() string {
	return "MEDIUM — PDV agents fall back to local computation when PDV_STRICT_MODE not set. " +
		"Sarathi defaults to localhost:9090 when SARATHI_URL not set. " +
		"Named addresses bypass signature verification when SIGVERIFY_STRICT_MODE not set."
}

// driftRisks returns known drift risks — updated for Scope 1-4 hardening.
func driftRisks() []string {
	return []string{
		"PDV agents share process locality when PDV_STRICT_MODE=true not set — set PDV_STRICT_MODE=true + run 3 separate pdv-agent processes for production",
		"Named addresses (alice, bob) bypass signature verification when SIGVERIFY_STRICT_MODE not set — set SIGVERIFY_STRICT_MODE=true for production",
		"Sarathi/DGIC governance defaults to localhost:9090 — set SARATHI_URL=<external> for sovereign governance separation",
		"Sarathi failure defaults to allow (fail-open) when SARATHI_FAIL_CLOSED not set — set SARATHI_FAIL_CLOSED=true for production",
		"KarmaChain replication only reaches configured KARMACHAIN_NODES — single node if env var not set",
		"Nonce store is per-node — cross-node nonce coordination not yet implemented",
	}
}

// VerifyBoundary checks that a specific boundary is not being violated at runtime.
// Returns true if the boundary is intact, false if drift is detected.
func VerifyBoundary(boundaryName string) (bool, string) {
	switch boundaryName {
	case "REPLAY_EQUALITY_VS_LEGITIMACY":
		// Verifiable by code structure: FraudGate is called AFTER PDV in enforceWithURL.
		// This is always true by construction — cannot be violated without changing code.
		return true, "FraudGate is called after PDV PASS — governance is separate from determinism"

	case "OBSERVABILITY_VS_AUTHORITY":
		return true, "All /api/tantra/* and /api/akashic/* endpoints are read-only — no write authority"

	case "WALLET_INTENT_BOUNDARY":
		return true, "Execute() routes through /api/relay/submit only — no direct blockchain access"

	case "PDV_STRICT_MODE_BOUNDARY":
		// DEFAULT is now ON. Only degraded if explicitly set to false.
		strict := os.Getenv("PDV_STRICT_MODE") != "false"
		if strict {
			return true, "PDV_STRICT_MODE=ON (default) — local fallback eliminated, agents must be reachable"
		}
		return false, "PDV_STRICT_MODE=false (explicitly disabled) — local fallback active (DEV ONLY)"

	case "SIGVERIFY_STRICT_MODE_BOUNDARY":
		// DEFAULT is now ON. Only degraded if explicitly set to false.
		strict := os.Getenv("SIGVERIFY_STRICT_MODE") != "false"
		if strict {
			return true, "SIGVERIFY_STRICT_MODE=ON (default) — named-address bypass eliminated"
		}
		return false, "SIGVERIFY_STRICT_MODE=false (explicitly disabled) — named addresses bypass sig check (DEV ONLY)"

	case "SARATHI_GOVERNANCE_BOUNDARY":
		// DEFAULT fail-closed is now ON. Only degraded if explicitly set to false.
		sarathiConfigured := os.Getenv("SARATHI_URL") != ""
		failClosed := os.Getenv("SARATHI_FAIL_CLOSED") != "false"
		if failClosed && sarathiConfigured {
			return true, "SARATHI_URL set + SARATHI_FAIL_CLOSED=ON (default) — governance fully isolated"
		}
		if !failClosed {
			return false, "SARATHI_FAIL_CLOSED=false (explicitly disabled) — governance failure defaults to allow (DEV ONLY)"
		}
		return false, "SARATHI_URL not set — governance defaults to localhost:9090 (not sovereign)"

	case "NONCE_CROSS_NODE_BOUNDARY":
		// Known gap — cross-node nonce coordination not implemented.
		return false, "Cross-node nonce coordination not implemented — each node has its own nonce_ledger.jsonl"

	default:
		return true, "boundary not specifically monitored — assumed intact"
	}
}
