package bridgesdk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/bridge-sdk/core/attest"
)

type EventRoot struct {
	ID        string    `json:"id"`
	Chain     string    `json:"chain"`
	RootHash  string    `json:"root_hash"`
	Timestamp time.Time `json:"timestamp"`
	Events    []Event   `json:"events"`
}

func NewEventRoot(id, chain string) *EventRoot {
	return &EventRoot{
		ID:        id,
		Chain:     chain,
		RootHash:  generateRootHash(id),
		Timestamp: time.Now(),
		Events:    make([]Event, 0),
	}
}

func (er *EventRoot) AddEvent(e Event) {
	er.Events = append(er.Events, e)
}

func (er *EventRoot) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"id":        er.ID,
		"chain":     er.Chain,
		"event_count": len(er.Events),
		"duration":  time.Since(er.Timestamp).String(),
		"avg_latency": calculateAvgLatency(er.Events),
	}
}

// Save persists the event root JSON and generates the attestation bundle
func (er *EventRoot) Save(rootsDir string) error {
	err := os.MkdirAll(rootsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create roots directory: %w", err)
	}

	data, err := json.MarshalIndent(er, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal event root: %w", err)
	}

	filename := filepath.Join(rootsDir, fmt.Sprintf("event_root_%s.json", er.RootHash))
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write event root file: %w", err)
	}

	// Hook the attestation bundle creation alongside the roots
	return attest.WriteAttestation(rootsDir, er.RootHash, er.Chain)
}

func generateRootHash(id string) string {
	h := sha256.Sum256([]byte(id + time.Now().String()))
	return hex.EncodeToString(h[:])
}

func calculateAvgLatency(events []Event) time.Duration {
	if len(events) == 0 {
		return 0
	}
	var total time.Duration
	for _, e := range events {
		// Calculate processing time from timestamp to processed time
		if e.ProcessedAt != nil {
			total += e.ProcessedAt.Sub(e.Timestamp)
		}
	}
	if total == 0 {
		return 0
	}
	return total / time.Duration(len(events))
}
