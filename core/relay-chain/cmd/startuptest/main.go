package main

import (
	"fmt"
	"os"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/startupcheck"
)

func main() {
	fmt.Println("=== TEST 1: Default (production-safe) ===")
	os.Unsetenv("PDV_STRICT_MODE")
	os.Unsetenv("SIGVERIFY_STRICT_MODE")
	os.Unsetenv("SARATHI_FAIL_CLOSED")
	startupcheck.PrintBanner()

	fmt.Println("\n=== TEST 2: All disabled (dev mode) ===")
	os.Setenv("PDV_STRICT_MODE", "false")
	os.Setenv("SIGVERIFY_STRICT_MODE", "false")
	os.Setenv("SARATHI_FAIL_CLOSED", "false")
	startupcheck.PrintBanner()

	fmt.Println("\n=== TEST 3: One disabled (degraded) ===")
	os.Unsetenv("PDV_STRICT_MODE")
	os.Unsetenv("SIGVERIFY_STRICT_MODE")
	os.Setenv("SARATHI_FAIL_CLOSED", "false")
	startupcheck.PrintBanner()
}
