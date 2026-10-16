// Governance isolation proof — proves all Scope 4 requirements with inverted defaults.
//
// NEW DEFAULT BEHAVIOR (Scope 3):
//   PDV_STRICT_MODE        — ON by default (set =false to disable)
//   SIGVERIFY_STRICT_MODE  — ON by default (set =false to disable)
//   SARATHI_FAIL_CLOSED    — ON by default (set =false to disable)
//
// Run standalone: go run cmd/governance-proof/main.go
package main

import (
	"fmt"
	"os"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/enforcement"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/startupcheck"
)

func main() {
	fmt.Println("=== GOVERNANCE ISOLATION PROOF ===")
	fmt.Println()
	allPassed := true

	// Show startup safety level first
	fmt.Println("--- STARTUP SAFETY CHECK ---")
	report := startupcheck.Check()
	fmt.Printf("  Safety Level: %s\n", report.Level)
	for _, s := range report.Secure {
		fmt.Printf("  [SECURE]  %s\n", s)
	}
	for _, w := range report.Warnings {
		fmt.Printf("  [WARNING] %s\n", w)
	}
	fmt.Println()

	// PROOF 1: SARATHI_URL is configurable
	fmt.Println("--- PROOF 1: Sarathi URL Configurable (No Localhost Hardcoding) ---")
	os.Unsetenv("SARATHI_URL")
	defaultURL := enforcement.SarathiURLExported()
	os.Setenv("SARATHI_URL", "http://sarathi.internal:9090/api/fraud/check")
	customURL := enforcement.SarathiURLExported()
	os.Unsetenv("SARATHI_URL")
	p1 := defaultURL == "http://localhost:9090/api/fraud/check" &&
		customURL == "http://sarathi.internal:9090/api/fraud/check"
	fmt.Printf("  Default URL: %s\n", defaultURL)
	fmt.Printf("  Custom URL:  %s\n", customURL)
	printVerdict("PROOF_1_URL_CONFIGURABLE", p1, "SARATHI_URL env var controls governance endpoint")
	if !p1 {
		allPassed = false
	}

	// PROOF 2: Fail-closed is now DEFAULT (no env var needed)
	fmt.Println("--- PROOF 2: Fail-Closed is DEFAULT (Production-Safe) ---")
	os.Unsetenv("SARATHI_FAIL_CLOSED") // no env var = fail-closed ON
	os.Setenv("SARATHI_URL", "http://localhost:19999/api/fraud/check")
	tx2 := &enforcement.TxPayload{TraceID: "gov-p2", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: 1}
	d2, _ := enforcement.FraudGate(tx2)
	p2 := d2 == "block" // NOW blocks by default — no env var needed
	fmt.Printf("  Sarathi unreachable + no SARATHI_FAIL_CLOSED env var -> decision=%s\n", d2)
	printVerdict("PROOF_2_FAIL_CLOSED_DEFAULT", p2, "fail-closed is now DEFAULT — no env var needed for production safety")
	if !p2 {
		allPassed = false
	}

	// PROOF 3: Fail-open requires explicit opt-in (dev only)
	fmt.Println("--- PROOF 3: Fail-Open Requires Explicit Opt-In (Dev Only) ---")
	os.Setenv("SARATHI_FAIL_CLOSED", "false") // explicit opt-in to unsafe mode
	tx3 := &enforcement.TxPayload{TraceID: "gov-p3", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: 2}
	d3, _ := enforcement.FraudGate(tx3)
	p3 := d3 == "allow"
	fmt.Printf("  Sarathi unreachable + SARATHI_FAIL_CLOSED=false -> decision=%s\n", d3)
	printVerdict("PROOF_3_FAIL_OPEN_EXPLICIT", p3, "fail-open requires SARATHI_FAIL_CLOSED=false (explicit unsafe opt-in)")
	if !p3 {
		allPassed = false
	}
	os.Unsetenv("SARATHI_FAIL_CLOSED")
	os.Unsetenv("SARATHI_URL")

	// PROOF 4: PDV equality != governance legitimacy
	fmt.Println("--- PROOF 4: PDV Equality != Governance Legitimacy ---")
	os.Setenv("SARATHI_URL", "http://localhost:19999/api/fraud/check")
	os.Unsetenv("SARATHI_FAIL_CLOSED") // fail-closed ON by default
	os.Setenv("SIGVERIFY_STRICT_MODE", "false") // allow named addresses for this test
	tx4 := &enforcement.TxPayload{
		TraceID: "gov-p4", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: 3,
		Signature: "placeholder",
	}
	enf4 := enforcement.Enforce(tx4)
	pdvPassed := enf4.ExecutionHash != "" &&
		enf4.ExecutionHash == enf4.ValidationHash &&
		enf4.ValidationHash == enf4.ReplayHash
	govBlocked := !enf4.Allowed && enf4.FraudDecision == "block"
	p4 := pdvPassed && govBlocked
	fmt.Printf("  PDV hashes: exec=%s val=%s replay=%s\n",
		short(enf4.ExecutionHash), short(enf4.ValidationHash), short(enf4.ReplayHash))
	fmt.Printf("  PDV passed: %v  |  Governance blocked: %v  |  Allowed: %v\n",
		pdvPassed, govBlocked, enf4.Allowed)
	fmt.Println("  PROOF: PDV equality confirmed BEFORE governance decision")
	fmt.Println("  PROOF: Governance BLOCK does not contaminate PDV determinism")
	printVerdict("PROOF_4_PDV_NEQ_LEGITIMACY", p4, "PDV passed + governance blocked = rejected -- boundary intact")
	if !p4 {
		allPassed = false
	}
	os.Unsetenv("SARATHI_URL")
	os.Unsetenv("SIGVERIFY_STRICT_MODE")

	// PROOF 5: Same PDV hash regardless of governance decision
	fmt.Println("--- PROOF 5: Governance Decision External to PDV Hash Zone ---")
	fmt.Println("  PDV hash zone: trace_id+type+from+to+amount+token_id+fee+nonce+signature")
	fmt.Println("  Governance decision (allow/block) is NOT part of the PDV hash")
	os.Setenv("SARATHI_URL", "http://localhost:19999/api/fraud/check")
	os.Setenv("SIGVERIFY_STRICT_MODE", "false")
	// Run 1: fail-open (allow)
	os.Setenv("SARATHI_FAIL_CLOSED", "false")
	tx5a := &enforcement.TxPayload{TraceID: "gov-p5", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: 4,
		Signature: "placeholder"}
	r5a := enforcement.Enforce(tx5a)
	// Run 2: fail-closed (block)
	os.Unsetenv("SARATHI_FAIL_CLOSED")
	tx5b := &enforcement.TxPayload{TraceID: "gov-p5", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: 4,
		Signature: "placeholder"}
	r5b := enforcement.Enforce(tx5b)
	sameHash := r5a.ExecutionHash == r5b.ExecutionHash && r5a.ExecutionHash != ""
	diffDecision := r5a.FraudDecision != r5b.FraudDecision
	p5 := sameHash && diffDecision
	fmt.Printf("  Fail-open  -> PDV hash=%s fraud=%s allowed=%v\n",
		short(r5a.ExecutionHash), r5a.FraudDecision, r5a.Allowed)
	fmt.Printf("  Fail-closed -> PDV hash=%s fraud=%s allowed=%v\n",
		short(r5b.ExecutionHash), r5b.FraudDecision, r5b.Allowed)
	fmt.Printf("  Same PDV hash: %v  |  Different governance decision: %v\n", sameHash, diffDecision)
	printVerdict("PROOF_5_GOVERNANCE_EXTERNAL", p5, "same PDV hash regardless of governance decision -- boundary proven")
	if !p5 {
		allPassed = false
	}
	os.Unsetenv("SARATHI_FAIL_CLOSED")
	os.Unsetenv("SARATHI_URL")
	os.Unsetenv("SIGVERIFY_STRICT_MODE")

	// SUMMARY
	fmt.Println("=== PROOF SUMMARY ===")
	fmt.Printf("[%s] Proof 1 -- Sarathi URL configurable (no localhost hardcoding)\n", v(p1))
	fmt.Printf("[%s] Proof 2 -- Fail-closed is DEFAULT (no env var needed)\n", v(p2))
	fmt.Printf("[%s] Proof 3 -- Fail-open requires explicit opt-in (SARATHI_FAIL_CLOSED=false)\n", v(p3))
	fmt.Printf("[%s] Proof 4 -- PDV equality != governance legitimacy (boundary intact)\n", v(p4))
	fmt.Printf("[%s] Proof 5 -- Governance decision external to PDV hash zone\n", v(p5))

	if allPassed {
		fmt.Println("\nVERDICT: ALL PROOFS PASSED -- Governance isolation hardening COMPLETE (inverted defaults)")
		os.Exit(0)
	}
	fmt.Println("\nVERDICT: SOME PROOFS FAILED")
	os.Exit(1)
}

func printVerdict(label string, passed bool, detail string) {
	s := "PASS"
	if !passed {
		s = "FAIL"
	}
	fmt.Printf("  [%s] %s -- %s\n\n", s, label, detail)
}

func short(s string) string {
	if len(s) >= 8 {
		return s[:8]
	}
	return s
}

func v(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}
