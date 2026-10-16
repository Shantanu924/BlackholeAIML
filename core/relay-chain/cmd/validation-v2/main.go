// Independent Non-Manipulable Validation Layer v2 — Scope 8
//
// Tests:
//   V1. Cross-host PDV equality (agents on separate URLs)
//   V2. Network partition detection (agent unreachable → hard reject)
//   V3. Nonce race condition (concurrent submissions, at most 1 accepted)
//   V4. Recovery after partition (agents restored → equality resumes)
//   V5. Distributed replay determinism (same payload, 3 runs, same hash)
//   V6. Cross-node nonce coordinator (same nonce rejected from node-B)
//   V7. Startup safety posture (production defaults verified)
//   V8. Tamper-evident suite proof (mutating any result changes suite_hash)
//
// Run standalone:
//   go run cmd/validation-v2/main.go
//
// Run against live relay + coordinator:
//   go run cmd/validation-v2/main.go -relay http://localhost:8080 -coord http://localhost:9200 -out v2_results.jsonl
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/noncecoord"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/pdv"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/startupcheck"
)

type V2Result struct {
	TestID     string `json:"test_id"`
	Name       string `json:"name"`
	Passed     bool   `json:"passed"`
	Actual     string `json:"actual"`
	Timestamp  int64  `json:"timestamp"`
	ResultHash string `json:"result_hash"`
}

func hashResult(r V2Result) string {
	raw := fmt.Sprintf("%s|%v|%s|%d", r.TestID, r.Passed, r.Actual, r.Timestamp)
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}

func suiteHash(results []V2Result) string {
	h := sha256.New()
	for _, r := range results {
		h.Write([]byte(r.ResultHash))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func record(id, name string, passed bool, actual string) V2Result {
	r := V2Result{
		TestID:    id,
		Name:      name,
		Passed:    passed,
		Actual:    actual,
		Timestamp: time.Now().Unix(),
	}
	r.ResultHash = hashResult(r)
	return r
}

func main() {
	relayURL := flag.String("relay", "", "relay URL (optional, enables live relay tests)")
	coordURL := flag.String("coord", "http://localhost:9200", "nonce coordinator URL")
	outFile := flag.String("out", "", "output JSONL file (optional)")
	flag.Parse()

	fmt.Println("=== INDEPENDENT VALIDATION LAYER v2 (Scope 8) ===")
	fmt.Println("Tamper-evident: result_hash = SHA-256(test_id+passed+actual+ts)")
	fmt.Println()

	var results []V2Result
	ts := time.Now().Unix()

	// V1: Cross-host PDV equality
	fmt.Println("--- V1: Cross-Host PDV Equality ---")
	os.Setenv("PDV_EXECUTION_AGENT_URL", "http://localhost:9101/pdv/execute")
	os.Setenv("PDV_VALIDATION_AGENT_URL", "http://localhost:9102/pdv/validate")
	os.Setenv("PDV_REPLAY_AGENT_URL", "http://localhost:9103/pdv/replay")
	req1 := pdv.AgentRequest{TraceID: "v2-v1", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: uint64(ts + 1)}
	r1 := pdv.Check(req1)
	v1 := record("V1", "Cross-Host PDV Equality",
		r1.Agreed && r1.ExecutionHash != "",
		fmt.Sprintf("agreed=%v exec=%s val=%s replay=%s",
			r1.Agreed, short(r1.ExecutionHash), short(r1.ValidationHash), short(r1.ReplayHash)))
	results = append(results, v1)
	fmt.Printf("  [%s] %s\n\n", pass(v1.Passed), v1.Name)

	// V2: Network partition — all agents unreachable → hard reject (strict ON by default)
	fmt.Println("--- V2: Network Partition Detection ---")
	os.Setenv("PDV_EXECUTION_AGENT_URL", "http://localhost:19901/pdv/execute")
	os.Setenv("PDV_VALIDATION_AGENT_URL", "http://localhost:19902/pdv/validate")
	os.Setenv("PDV_REPLAY_AGENT_URL", "http://localhost:19903/pdv/replay")
	os.Unsetenv("PDV_STRICT_MODE") // strict ON by default (startupcheck inversion)
	req2 := pdv.AgentRequest{TraceID: "v2-v2", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: uint64(ts + 2)}
	r2 := pdv.Check(req2)
	v2 := record("V2", "Network Partition Detection",
		!r2.Agreed && r2.RejectionReason != "",
		fmt.Sprintf("agreed=%v reason=%s", r2.Agreed, truncate(r2.RejectionReason, 60)))
	results = append(results, v2)
	fmt.Printf("  [%s] %s\n\n", pass(v2.Passed), v2.Name)

	// V3: Nonce race condition — 5 concurrent submissions, at most 1 accepted
	fmt.Println("--- V3: Nonce Race Condition (Concurrent Submissions) ---")
	os.Setenv("NONCE_COORDINATOR_URL", *coordURL)
	nonce3 := uint64(ts + 300000)
	if *relayURL != "" {
		// Live relay test
		type concR struct {
			status int
		}
		concResults := make([]concR, 5)
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				body := fmt.Sprintf(`{"schema_version":"v1","type":"token_transfer","from":"alice","to":"bob","amount":1,"token_id":"BHIV","nonce":%d,"timestamp":%d}`,
					nonce3, ts)
				s, _ := postHTTP(*relayURL+"/api/relay/submit", body)
				concResults[idx] = concR{s}
			}(i)
		}
		wg.Wait()
		accepted := 0
		for _, cr := range concResults {
			if cr.status == 200 {
				accepted++
			}
		}
		v3 := record("V3", "Nonce Race Condition (at most 1 accepted)",
			accepted <= 1,
			fmt.Sprintf("accepted=%d/5 via relay (expected <=1)", accepted))
		results = append(results, v3)
		fmt.Printf("  [%s] %s — accepted=%d/5\n\n", pass(v3.Passed), v3.Name, accepted)
	} else {
		// Standalone: prove via noncecoord directly
		type concR struct{ err error }
		concResults := make([]concR, 5)
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				err := noncecoord.CheckWithCoordinator("alice", nonce3,
					fmt.Sprintf("v3-%d", idx), fmt.Sprintf("node-%d", idx))
				concResults[idx] = concR{err}
			}(i)
		}
		wg.Wait()
		accepted := 0
		for _, cr := range concResults {
			if cr.err == nil {
				accepted++
			}
		}
		v3 := record("V3", "Nonce Race Condition (at most 1 accepted)",
			accepted <= 1,
			fmt.Sprintf("accepted=%d/5 via coordinator (expected <=1)", accepted))
		results = append(results, v3)
		fmt.Printf("  [%s] %s — accepted=%d/5\n\n", pass(v3.Passed), v3.Name, accepted)
	}
	os.Unsetenv("NONCE_COORDINATOR_URL")

	// V4: Recovery after partition — restore agents, equality resumes
	fmt.Println("--- V4: Recovery After Partition ---")
	os.Setenv("PDV_EXECUTION_AGENT_URL", "http://localhost:9101/pdv/execute")
	os.Setenv("PDV_VALIDATION_AGENT_URL", "http://localhost:9102/pdv/validate")
	os.Setenv("PDV_REPLAY_AGENT_URL", "http://localhost:9103/pdv/replay")
	req4 := pdv.AgentRequest{TraceID: "v2-v4", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: uint64(ts + 4)}
	r4 := pdv.Check(req4)
	v4 := record("V4", "Recovery After Partition",
		r4.Agreed,
		fmt.Sprintf("agreed=%v exec=%s (agents restored)", r4.Agreed, short(r4.ExecutionHash)))
	results = append(results, v4)
	fmt.Printf("  [%s] %s\n\n", pass(v4.Passed), v4.Name)

	// V5: Distributed replay determinism — same payload, 3 runs, identical hashes
	fmt.Println("--- V5: Distributed Replay Determinism (3 runs) ---")
	req5 := pdv.AgentRequest{TraceID: "v2-v5", Type: "token_transfer",
		From: "alice", To: "bob", Amount: 100, TokenID: "BHIV", Nonce: uint64(ts + 5)}
	r5a := pdv.Check(req5)
	r5b := pdv.Check(req5)
	r5c := pdv.Check(req5)
	allSame := r5a.ExecutionHash == r5b.ExecutionHash &&
		r5b.ExecutionHash == r5c.ExecutionHash &&
		r5a.ExecutionHash != ""
	v5 := record("V5", "Distributed Replay Determinism",
		allSame,
		fmt.Sprintf("run1=%s run2=%s run3=%s equal=%v",
			short(r5a.ExecutionHash), short(r5b.ExecutionHash), short(r5c.ExecutionHash), allSame))
	results = append(results, v5)
	fmt.Printf("  [%s] %s\n\n", pass(v5.Passed), v5.Name)

	// V6: Cross-node nonce coordinator — same nonce rejected from node-B
	fmt.Println("--- V6: Cross-Node Nonce Coordinator ---")
	os.Setenv("NONCE_COORDINATOR_URL", *coordURL)
	nonce6 := uint64(ts + 600000)
	err6a := noncecoord.CheckWithCoordinator("bob", nonce6, "v6-a", "node-A")
	err6b := noncecoord.CheckWithCoordinator("bob", nonce6, "v6-b", "node-B")
	v6 := record("V6", "Cross-Node Nonce Coordinator",
		err6a == nil && err6b != nil,
		fmt.Sprintf("node-A err=%v node-B err=%v", err6a, err6b))
	results = append(results, v6)
	fmt.Printf("  [%s] %s\n\n", pass(v6.Passed), v6.Name)
	os.Unsetenv("NONCE_COORDINATOR_URL")

	// V7: Startup safety posture — production defaults verified (Scope 3 inversion)
	fmt.Println("--- V7: Startup Safety Posture (Production Defaults) ---")
	os.Unsetenv("PDV_STRICT_MODE")
	os.Unsetenv("SIGVERIFY_STRICT_MODE")
	os.Unsetenv("SARATHI_FAIL_CLOSED")
	report := startupcheck.Check()
	v7 := record("V7", "Startup Safety Posture (production defaults)",
		report.Level == startupcheck.SafetyProduction,
		fmt.Sprintf("level=%s warnings=%d secure=%d",
			report.Level, len(report.Warnings), len(report.Secure)))
	results = append(results, v7)
	fmt.Printf("  [%s] %s — level=%s\n\n", pass(v7.Passed), v7.Name, report.Level)

	// V8: Tamper-evident suite proof — mutating any result changes suite_hash
	fmt.Println("--- V8: Tamper-Evident Suite Proof ---")
	sh := suiteHash(results)
	mutated := results[0]
	mutated.Passed = !mutated.Passed
	mutated.ResultHash = hashResult(mutated)
	mutatedResults := make([]V2Result, len(results))
	copy(mutatedResults, results)
	mutatedResults[0] = mutated
	mutatedHash := suiteHash(mutatedResults)
	tamperDetectable := sh != mutatedHash
	v8 := record("V8", "Tamper-Evident Suite Proof",
		tamperDetectable,
		fmt.Sprintf("suite_hash=%s tamper_changes_hash=%v", sh[:16], tamperDetectable))
	results = append(results, v8)
	fmt.Printf("  [%s] %s\n\n", pass(v8.Passed), v8.Name)

	// Final suite hash (includes V8)
	finalHash := suiteHash(results)

	// Summary
	passed := 0
	fmt.Println("=== VALIDATION v2 SUMMARY ===")
	for _, r := range results {
		fmt.Printf("[%s] %s — %s\n", pass(r.Passed), r.TestID, r.Name)
		if r.Passed {
			passed++
		}
	}
	fmt.Printf("\nPassed: %d/%d\n", passed, len(results))
	fmt.Printf("Suite Hash: %s\n", finalHash)
	fmt.Println("Any manipulation of any result changes the suite hash — detectable immediately.")

	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err == nil {
			enc := json.NewEncoder(f)
			for _, r := range results {
				enc.Encode(r)
			}
			enc.Encode(map[string]interface{}{
				"suite_hash": finalHash,
				"passed":     passed,
				"total":      len(results),
				"timestamp":  time.Now().Unix(),
			})
			f.Close()
			fmt.Printf("Results written to %s\n", *outFile)
		}
	}

	if passed == len(results) {
		fmt.Println("\nVERDICT: ALL VALIDATION v2 TESTS PASSED")
		fmt.Println("         Manipulation is easy to detect, difficult to conceal.")
		os.Exit(0)
	}
	fmt.Printf("\nVERDICT: %d/%d PASSED\n", passed, len(results))
	os.Exit(1)
}

func postHTTP(url, body string) (int, string) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

func pass(p bool) string {
	if p {
		return "PASS"
	}
	return "FAIL"
}

func short(s string) string {
	if len(s) >= 16 {
		return s[:16]
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
