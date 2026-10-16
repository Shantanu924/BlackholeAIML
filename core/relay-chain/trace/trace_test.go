package trace

import (
	"testing"
)

func TestTraceContext_ImmutabilityAndContinuity(t *testing.T) {
	ctx := New("trace-immutable-1")

	if ctx.ID() != "trace-immutable-1" {
		t.Errorf("Expected ID 'trace-immutable-1', got '%s'", ctx.ID())
	}

	// Assert continuity with same ID passes
	if err := ctx.AssertContinuity("trace-immutable-1", "STAGE_PDV"); err != nil {
		t.Errorf("Expected continuity check to pass, got error: %v", err)
	}

	// Assert continuity with drifted ID returns TraceBreakError
	err := ctx.AssertContinuity("trace-drifted", "STAGE_BLOCKCHAIN")
	if err == nil {
		t.Fatalf("Expected TraceBreakError on drifted ID, got nil")
	}

	// Attempting to inject different ID should fail
	if err := ctx.Inject("trace-injected-other"); err == nil {
		t.Fatalf("Expected injection of conflicting trace ID to fail, got nil")
	}
}
