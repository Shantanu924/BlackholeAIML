package startupcheck

import (
	"os"
	"testing"
)

func TestStartupCheck_DefaultProduction(t *testing.T) {
	// Clear any overrides to test default secure posture
	os.Unsetenv("PDV_STRICT_MODE")
	os.Unsetenv("SIGVERIFY_STRICT_MODE")
	os.Unsetenv("SARATHI_FAIL_CLOSED")

	report := Check()
	if report.Level != SafetyProduction {
		t.Errorf("Expected default safety level to be PRODUCTION, got %s", report.Level)
	}

	if len(report.Warnings) != 0 {
		t.Errorf("Expected 0 warnings in default production mode, got %d", len(report.Warnings))
	}

	if len(report.Secure) != 3 {
		t.Errorf("Expected 3 secure flags active, got %d", len(report.Secure))
	}
}

func TestStartupCheck_DevModeDetection(t *testing.T) {
	os.Setenv("PDV_STRICT_MODE", "false")
	os.Setenv("SIGVERIFY_STRICT_MODE", "false")
	os.Setenv("SARATHI_FAIL_CLOSED", "false")
	defer func() {
		os.Unsetenv("PDV_STRICT_MODE")
		os.Unsetenv("SIGVERIFY_STRICT_MODE")
		os.Unsetenv("SARATHI_FAIL_CLOSED")
	}()

	report := Check()
	if report.Level != SafetyDev {
		t.Errorf("Expected safety level DEV when all flags are false, got %s", report.Level)
	}

	if len(report.Warnings) != 3 {
		t.Errorf("Expected 3 warnings in DEV mode, got %d", len(report.Warnings))
	}
}

func TestStartupCheck_DegradedModeDetection(t *testing.T) {
	os.Setenv("PDV_STRICT_MODE", "false")
	os.Unsetenv("SIGVERIFY_STRICT_MODE")
	os.Unsetenv("SARATHI_FAIL_CLOSED")
	defer os.Unsetenv("PDV_STRICT_MODE")

	report := Check()
	if report.Level != SafetyDegraded {
		t.Errorf("Expected safety level DEGRADED with 1 warning, got %s", report.Level)
	}
}
