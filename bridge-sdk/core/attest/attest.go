package attest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// AttestationBundle represents the cross-chain transaction validation evidence
type AttestationBundle struct {
	RootHash    string    `json:"root_hash"`
	Timestamp   time.Time `json:"timestamp"`
	Signatures  []string  `json:"signatures"` // Stubs for multi-signature signers
	TargetChain string    `json:"target_chain"`
	Version     string    `json:"version"`
}

// WriteAttestation writes a JSON attestation file to the given directory
func WriteAttestation(rootsDir string, rootHash string, targetChain string) error {
	err := os.MkdirAll(rootsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create roots directory: %w", err)
	}

	bundle := AttestationBundle{
		RootHash:    rootHash,
		Timestamp:   time.Now(),
		Signatures:  []string{"signer_1_sig_stub", "signer_2_sig_stub"},
		TargetChain: targetChain,
		Version:     "v1alpha1",
	}

	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal attestation: %w", err)
	}

	filename := filepath.Join(rootsDir, fmt.Sprintf("attestation_%s.json", rootHash))
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write attestation file: %w", err)
	}

	return nil
}
