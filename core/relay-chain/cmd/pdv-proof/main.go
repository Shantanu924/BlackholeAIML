// PDV distributed equality proof — runs against 3 live separate-process agents.
// Produces structured proof of:
//   1. node_A_hash == node_B_hash == node_C_hash  (equality)
//   2. DISTRIBUTED_PDV_REJECT on agent failure    (hard rejection, strict mode)
//   3. Deterministic recovery after agent restore
//
// Run AFTER starting 3 pdv-agent processes:
//   pdv-agent.exe -port 9101 -agent ExecutionAgent
//   pdv-agent.exe -port 9102 -agent ValidationAgent
//   pdv-agent.exe -port 9103 -agent ReplayAgent
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/pdv"
)

func main() {
	os.Setenv("PDV_EXECUTION_AGENT_URL", "http://localhost:9101/pdv/execute")
	os.Setenv("PDV_VALIDATION_AGENT_URL", "http://localhost:9102/pdv/validate")
	os.Setenv("PDV_REPLAY_AGENT_URL", "http://localhost:9103/pdv/replay")
	os.Setenv("PDV_STRICT_MODE", "true")

	fmt.Println("=== PDV DISTRIBUTED EQUALITY PROOF ===")
	fmt.Println("Agents : 3 separate OS processes (9101 / 9102 / 9103)")
	fmt.Println("Mode   : PDV_STRICT_MODE=true — unreachable agent = hard fail")
	fmt.Println()

	ts := time.Now().UnixNano()

	// ── PROOF 1: equality ────────────────────────────────────────────────────
	fmt.Println("--- PROOF 1: Distributed Equality (same payload → same hash) ---")
	r1 := pdv.Check(pdv.AgentRequest{
		TraceID: fmt.Sprintf("proof-eq-%d", ts),
		Type:    "token_transfer",
		From:    "alice", To: "bob",
		Amount: 100, TokenID: "BHIV", Nonce: 1,
	})
	printResult("PROOF_1", r1)
	if r1.Agreed {
		fmt.Printf("VERDICT: node_A == node_B == node_C = %s\n", r1.ExecutionHash[:16])
		fmt.Printf("         DISTRIBUTED EQUALITY CONFIRMED across 3 separate OS processes\n\n")
	} else {
		fmt.Printf("VERDICT: UNEXPECTED DIVERGENCE — %s\n\n", r1.RejectionReason)
	}

	// ── PROOF 2: hard rejection when one agent is unreachable ────────────────
	fmt.Println("--- PROOF 2: Hard Rejection on Agent Failure (strict mode) ---")
	os.Setenv("PDV_VALIDATION_AGENT_URL", "http://localhost:9199/pdv/validate") // dead port
	r2 := pdv.Check(pdv.AgentRequest{
		TraceID: fmt.Sprintf("proof-rej-%d", ts),
		Type:    "token_transfer",
		From:    "alice", To: "bob",
		Amount: 100, TokenID: "BHIV", Nonce: 2,
	})
	printResult("PROOF_2", r2)
	if !r2.Agreed {
		fmt.Printf("VERDICT: DISTRIBUTED_PDV_REJECT — hard structured rejection\n")
		fmt.Printf("         reason: %s\n\n", r2.RejectionReason)
	} else {
		fmt.Printf("VERDICT: UNEXPECTED PASS — strict mode should have rejected\n\n")
	}

	// ── PROOF 3: equality restored after agent recovery ──────────────────────
	fmt.Println("--- PROOF 3: Equality Restored After Agent Recovery ---")
	os.Setenv("PDV_VALIDATION_AGENT_URL", "http://localhost:9102/pdv/validate")
	r3 := pdv.Check(pdv.AgentRequest{
		TraceID: fmt.Sprintf("proof-rec-%d", ts),
		Type:    "token_transfer",
		From:    "alice", To: "bob",
		Amount: 100, TokenID: "BHIV", Nonce: 3,
	})
	printResult("PROOF_3", r3)
	if r3.Agreed {
		fmt.Printf("VERDICT: Equality restored — node_A == node_B == node_C = %s\n", r3.ExecutionHash[:16])
		fmt.Printf("         System recovers deterministically after agent recovery\n\n")
	} else {
		fmt.Printf("VERDICT: UNEXPECTED DIVERGENCE\n\n")
	}

	// ── SUMMARY ──────────────────────────────────────────────────────────────
	p1 := verdict(r1.Agreed)
	p2 := verdict(!r2.Agreed)
	p3 := verdict(r3.Agreed)
	fmt.Println("=== PROOF SUMMARY ===")
	fmt.Printf("[%s] Proof 1 — Distributed equality across 3 separate OS processes\n", p1)
	fmt.Printf("[%s] Proof 2 — Hard structured rejection on agent failure (strict mode)\n", p2)
	fmt.Printf("[%s] Proof 3 — Deterministic recovery after agent restoration\n", p3)

	if p1 == "PASS" && p2 == "PASS" && p3 == "PASS" {
		fmt.Println("\nVERDICT: ALL PROOFS PASSED — Distributed PDV hardening COMPLETE")
		os.Exit(0)
	}
	fmt.Println("\nVERDICT: SOME PROOFS FAILED")
	os.Exit(1)
}

func printResult(label string, r pdv.DistributedResult) {
	data, _ := json.MarshalIndent(r, "  ", "  ")
	fmt.Printf("[%s]\n  %s\n", label, string(data))
}

func verdict(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}
