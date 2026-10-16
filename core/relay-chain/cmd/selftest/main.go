// Selftest runner — executes all 12 self-validation tests against a live relay node.
//
// Usage:
//   go run cmd/selftest/main.go -relay http://localhost:8080
//   go run cmd/selftest/main.go -relay http://localhost:8080 -out results.jsonl
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/selftest"
)

func main() {
	relayURL := flag.String("relay", "http://localhost:8080", "relay node URL")
	outFile := flag.String("out", "", "output file path (default: selftest_results_<ts>.jsonl)")
	flag.Parse()

	runner := selftest.New(*relayURL)
	suite := runner.Run()

	// Write tamper-evident results file.
	if err := selftest.WriteResults(suite, *outFile); err != nil {
		log.Printf("[SELFTEST] Warning: could not write results file: %v", err)
	}

	// Print summary to stdout.
	fmt.Printf("\n=== SELFTEST SUITE RESULT ===\n")
	fmt.Printf("Relay:       %s\n", suite.RelayURL)
	fmt.Printf("Total:       %d\n", suite.TotalTests)
	fmt.Printf("Passed:      %d\n", suite.Passed)
	fmt.Printf("Failed:      %d\n", suite.Failed)
	fmt.Printf("Suite Hash:  %s\n", suite.SuiteHash)
	fmt.Printf("\n--- Per-Test Results ---\n")
	for _, r := range suite.Results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		fmt.Printf("[%s] %s — %s\n", status, r.TestID, r.Name)
		if !r.Passed {
			fmt.Printf("       Expected: %s\n", r.Expected)
			fmt.Printf("       Actual:   %s\n", r.Actual)
		}
	}
	fmt.Printf("\nResult hash (tamper-evident): %s\n", suite.SuiteHash)

	// Also print full JSON for audit.
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(suite)

	if suite.Failed > 0 {
		os.Exit(1)
	}
}
