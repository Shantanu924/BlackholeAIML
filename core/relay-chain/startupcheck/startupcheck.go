// Package startupcheck implements production-safe default enforcement.
//
// SECURITY INVERSION (Scope 3):
//   Previous behavior: secure = opt-in (set env var to enable)
//   New behavior:      secure = default (set env var to DISABLE)
//
// New defaults (no env vars needed for production):
//   PDV_STRICT_MODE        — default ON  (set PDV_STRICT_MODE=false to disable)
//   SIGVERIFY_STRICT_MODE  — default ON  (set SIGVERIFY_STRICT_MODE=false to disable)
//   SARATHI_FAIL_CLOSED    — default ON  (set SARATHI_FAIL_CLOSED=false to disable)
//
// Unsafe mode opt-in emits a hard WARNING banner at startup.
// This makes unsafe startup observable and auditable.
package startupcheck

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// SafetyLevel classifies the startup safety posture.
type SafetyLevel string

const (
	SafetyProduction SafetyLevel = "PRODUCTION"
	SafetyDegraded   SafetyLevel = "DEGRADED"
	SafetyDev        SafetyLevel = "DEV"
)

// StartupReport is the result of the startup safety check.
type StartupReport struct {
	Level    SafetyLevel `json:"level"`
	Warnings []string    `json:"warnings"`
	Secure   []string    `json:"secure"`
}

// IsPDVStrictMode returns true when PDV strict mode is active.
// DEFAULT: true (secure). Set PDV_STRICT_MODE=false to disable.
func IsPDVStrictMode() bool {
	v := os.Getenv("PDV_STRICT_MODE")
	return v != "false" // anything except explicit "false" = strict
}

// IsSigVerifyStrictMode returns true when sigverify strict mode is active.
// DEFAULT: true (secure). Set SIGVERIFY_STRICT_MODE=false to disable.
func IsSigVerifyStrictMode() bool {
	v := os.Getenv("SIGVERIFY_STRICT_MODE")
	return v != "false"
}

// IsSarathiFailClosed returns true when Sarathi fail-closed is active.
// DEFAULT: true (secure). Set SARATHI_FAIL_CLOSED=false to disable.
func IsSarathiFailClosed() bool {
	v := os.Getenv("SARATHI_FAIL_CLOSED")
	return v != "false"
}

// Check performs the startup safety check and returns a report.
// Call this at relay node startup — before accepting any connections.
func Check() StartupReport {
	report := StartupReport{}

	pdvStrict := IsPDVStrictMode()
	sigStrict := IsSigVerifyStrictMode()
	sarathiClosed := IsSarathiFailClosed()

	if pdvStrict {
		report.Secure = append(report.Secure, "PDV_STRICT_MODE=ON (default) — agent unreachable = hard fail")
	} else {
		report.Warnings = append(report.Warnings,
			"PDV_STRICT_MODE=OFF — agents fall back to local computation (DEV ONLY, set PDV_STRICT_MODE=false explicitly)")
	}

	if sigStrict {
		report.Secure = append(report.Secure, "SIGVERIFY_STRICT_MODE=ON (default) — named addresses hard rejected")
	} else {
		report.Warnings = append(report.Warnings,
			"SIGVERIFY_STRICT_MODE=OFF — named addresses bypass signature check (DEV ONLY)")
	}

	if sarathiClosed {
		report.Secure = append(report.Secure, "SARATHI_FAIL_CLOSED=ON (default) — governance unreachable = block")
	} else {
		report.Warnings = append(report.Warnings,
			"SARATHI_FAIL_CLOSED=OFF — governance unreachable defaults to allow (DEV ONLY)")
	}

	switch len(report.Warnings) {
	case 0:
		report.Level = SafetyProduction
	case 1:
		report.Level = SafetyDegraded
	default:
		report.Level = SafetyDev
	}

	return report
}

// PrintBanner prints the startup safety banner to stdout and log.
// Must be called at relay startup before serving requests.
func PrintBanner() StartupReport {
	report := Check()

	banner := strings.Repeat("=", 70)
	fmt.Println(banner)
	fmt.Printf("  BLACKHOLE BLOCKCHAIN — STARTUP SAFETY CHECK\n")
	fmt.Printf("  Safety Level: %s\n", report.Level)
	fmt.Println(banner)

	for _, s := range report.Secure {
		fmt.Printf("  [SECURE]  %s\n", s)
	}

	for _, w := range report.Warnings {
		fmt.Printf("  [WARNING] %s\n", w)
		log.Printf("[STARTUP][UNSAFE_MODE] %s", w)
	}

	if report.Level == SafetyDev {
		fmt.Println(banner)
		fmt.Println("  !! DEV MODE ACTIVE — NOT SAFE FOR PRODUCTION !!")
		fmt.Println("  To restore production safety, unset or remove:")
		fmt.Println("    PDV_STRICT_MODE=false")
		fmt.Println("    SIGVERIFY_STRICT_MODE=false")
		fmt.Println("    SARATHI_FAIL_CLOSED=false")
		log.Printf("[STARTUP][DEV_MODE] relay started in DEV mode — %d unsafe settings active", len(report.Warnings))
	} else if report.Level == SafetyDegraded {
		fmt.Println(banner)
		fmt.Println("  !! DEGRADED MODE — 1 unsafe setting active !!")
		log.Printf("[STARTUP][DEGRADED_MODE] relay started in DEGRADED mode")
	} else {
		fmt.Println(banner)
		fmt.Println("  PRODUCTION MODE — all safety defaults active")
		log.Printf("[STARTUP][PRODUCTION_MODE] relay started in PRODUCTION mode")
	}

	fmt.Println(banner)
	return report
}
