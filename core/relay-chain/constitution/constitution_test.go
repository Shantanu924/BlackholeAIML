package constitution

import (
	"os"
	"testing"
)

func TestConstitution_Declaration(t *testing.T) {
	dec := Declare()

	if dec.SystemName == "" {
		t.Errorf("Expected non-empty system name")
	}

	if len(dec.Boundaries) == 0 {
		t.Fatalf("Expected non-empty boundary declarations")
	}

	// Verify Category A (OWNED), Category B (NOT_OWNED), and Category C (BOUNDED) exist
	hasOwned := false
	hasNotOwned := false
	hasBounded := false

	for _, b := range dec.Boundaries {
		if b.Authority == AuthorityOwned {
			hasOwned = true
		}
		if b.Authority == AuthorityNotOwned {
			hasNotOwned = true
		}
		if b.Authority == AuthorityBounded {
			hasBounded = true
		}
	}

	if !hasOwned || !hasNotOwned || !hasBounded {
		t.Errorf("Expected all 3 authority types (OWNED, NOT_OWNED, BOUNDED) in declaration")
	}
}

func TestConstitution_VerifyBoundary(t *testing.T) {
	// 1. REPLAY_EQUALITY_VS_LEGITIMACY should always be true by structural design
	intact, msg := VerifyBoundary("REPLAY_EQUALITY_VS_LEGITIMACY")
	if !intact {
		t.Errorf("Expected REPLAY_EQUALITY_VS_LEGITIMACY to be intact, got: %s", msg)
	}

	// 2. OBSERVABILITY_VS_AUTHORITY should be intact
	intact2, _ := VerifyBoundary("OBSERVABILITY_VS_AUTHORITY")
	if !intact2 {
		t.Errorf("Expected OBSERVABILITY_VS_AUTHORITY to be intact")
	}

	// 3. PDV_STRICT_MODE_BOUNDARY default should be intact
	os.Unsetenv("PDV_STRICT_MODE")
	intact3, _ := VerifyBoundary("PDV_STRICT_MODE_BOUNDARY")
	if !intact3 {
		t.Errorf("Expected PDV_STRICT_MODE_BOUNDARY to be intact by default")
	}
}
