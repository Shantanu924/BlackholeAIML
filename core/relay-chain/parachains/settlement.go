package parachains

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
)

// CrossLayerMessage represents a message sent between Layer 1 and Layer 2
type CrossLayerMessage struct {
	ID          string      `json:"id"`
	SourceChain string      `json:"source_chain"` // e.g., "L1", "L2-EVM", "L2-WASM"
	TargetChain string      `json:"target_chain"`
	PayloadType string      `json:"payload_type"` // e.g., "TOKEN_TRANSFER", "CONTRACT_CALL"
	Payload     []byte      `json:"payload"`
	Sender      string      `json:"sender"`
	Receiver    string      `json:"receiver"`
	Status      MessageStatus `json:"status"`
	Timestamp   int64       `json:"timestamp"`
	Proof       []byte      `json:"proof"` // Cryptographic proof of validity from source chain
}

// MessageStatus defines the state of a cross-layer message
type MessageStatus string

const (
	MessagePending   MessageStatus = "PENDING"
	MessageVerified  MessageStatus = "VERIFIED"
	MessageProcessed MessageStatus = "PROCESSED"
	MessageFailed    MessageStatus = "FAILED"
)

// SettlementEngine handles cross-layer message settlement
type SettlementEngine struct {
	mu           sync.RWMutex
	messages     map[string]*CrossLayerMessage
	trustedRelayers map[string]bool
}

// NewSettlementEngine initializes a new Settlement Engine
func NewSettlementEngine() *SettlementEngine {
	return &SettlementEngine{
		messages:        make(map[string]*CrossLayerMessage),
		trustedRelayers: make(map[string]bool),
	}
}

// RegisterRelayer authorizes a new relayer to submit messages
func (se *SettlementEngine) RegisterRelayer(address string) {
	se.mu.Lock()
	defer se.mu.Unlock()
	se.trustedRelayers[address] = true
}

// SubmitMessage submits a message from another layer for settlement
func (se *SettlementEngine) SubmitMessage(relayer string, msg *CrossLayerMessage) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	if !se.trustedRelayers[relayer] {
		return fmt.Errorf("unauthorized relayer: %s", relayer)
	}

	// Generate ID if missing
	if msg.ID == "" {
		hashInput := fmt.Sprintf("%s:%s:%d:%s", msg.SourceChain, msg.TargetChain, msg.Timestamp, msg.Sender)
		hash := sha256.Sum256([]byte(hashInput))
		msg.ID = fmt.Sprintf("%x", hash)
	}

	if _, exists := se.messages[msg.ID]; exists {
		return fmt.Errorf("message %s already exists", msg.ID)
	}

	msg.Status = MessagePending
	se.messages[msg.ID] = msg

	return nil
}

// VerifyMessage validates the proof of a cross-layer message
func (se *SettlementEngine) VerifyMessage(msgID string) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	msg, exists := se.messages[msgID]
	if !exists {
		return fmt.Errorf("message %s not found", msgID)
	}

	if msg.Status != MessagePending {
		return fmt.Errorf("message is not pending (current status: %s)", msg.Status)
	}

	// TODO: Actually verify the proof using ZK proofs or light client validation
	// For now, we assume structural validity based on our trusted relayers
	if len(msg.Proof) == 0 {
		msg.Status = MessageFailed
		return fmt.Errorf("invalid proof for message %s", msgID)
	}

	msg.Status = MessageVerified
	return nil
}

// ProcessMessage executes the payload of a verified message
func (se *SettlementEngine) ProcessMessage(msgID string, processor func(payload []byte) error) error {
	se.mu.Lock()
	defer se.mu.Unlock()

	msg, exists := se.messages[msgID]
	if !exists {
		return fmt.Errorf("message %s not found", msgID)
	}

	if msg.Status != MessageVerified {
		return fmt.Errorf("message %s is not verified", msgID)
	}

	err := processor(msg.Payload)
	if err != nil {
		msg.Status = MessageFailed
		return fmt.Errorf("failed to process message %s: %v", msgID, err)
	}

	msg.Status = MessageProcessed
	return nil
}

// GetMessage retrieves a message by its ID
func (se *SettlementEngine) GetMessage(msgID string) (*CrossLayerMessage, error) {
	se.mu.RLock()
	defer se.mu.RUnlock()

	msg, exists := se.messages[msgID]
	if !exists {
		return nil, fmt.Errorf("message %s not found", msgID)
	}

	// Return a copy to avoid race conditions
	b, _ := json.Marshal(msg)
	var copy CrossLayerMessage
	json.Unmarshal(b, &copy)

	return &copy, nil
}
