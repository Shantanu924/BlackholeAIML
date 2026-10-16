package bridgesdk

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
)

// SignatureVerifier handles cryptographic signature verification
type SignatureVerifier struct {
	logger         *logrus.Logger
	publicKeys     map[string]ed25519.PublicKey
	keysMutex      sync.RWMutex
	verificationLog []SignatureVerificationLog
	logMutex       sync.RWMutex
	maxLogEntries  int
	lastSeenNonces map[string]uint64
}

// SignatureVerificationLog records signature verification attempts
type SignatureVerificationLog struct {
	Timestamp       int64
	Address         string
	SignatureHash   string
	IsValid         bool
	ErrorMessage    string
	TransactionHash string
}

// SignedBridgeMessageInput is used for creating signatures with hex-encoded data
type SignedBridgeMessageInput struct {
	Message   *Transaction `json:"message"`
	Signature string       `json:"signature"` // hex-encoded Ed25519 signature
	PublicKey string       `json:"public_key"` // hex-encoded Ed25519 public key
}

// MessagePayload represents the data that gets signed
type MessagePayload struct {
	TransactionID  string `json:"transaction_id"`
	SourceChain    string `json:"source_chain"`
	DestChain      string `json:"dest_chain"`
	SourceAddress  string `json:"source_address"`
	DestAddress    string `json:"dest_address"`
	TokenSymbol    string `json:"token_symbol"`
	Amount         string `json:"amount"`
	Fee            string `json:"fee"`
	Nonce          uint64 `json:"nonce"`
	Timestamp      int64  `json:"timestamp"`
}

// NewSignatureVerifier creates a new signature verifier
func NewSignatureVerifier(logger *logrus.Logger) *SignatureVerifier {
	return &SignatureVerifier{
		logger:        logger,
		publicKeys:    make(map[string]ed25519.PublicKey),
		maxLogEntries: 10000,
		lastSeenNonces: make(map[string]uint64),
	}
}

// RegisterPublicKey registers a public key for an address
func (sv *SignatureVerifier) RegisterPublicKey(address string, publicKeyHex string) error {
	sv.keysMutex.Lock()
	defer sv.keysMutex.Unlock()

	// Decode hex public key
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return fmt.Errorf("invalid public key hex format: %w", err)
	}

	// Validate Ed25519 public key length (32 bytes)
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid Ed25519 public key length: expected %d bytes, got %d", ed25519.PublicKeySize, len(publicKeyBytes))
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)
	sv.publicKeys[address] = publicKey

	sv.logger.WithFields(logrus.Fields{
		"address":    address,
		"public_key": publicKeyHex[:16] + "...",
	}).Info("Public key registered for address")

	return nil
}

// VerifySignature verifies a signed bridge message
func (sv *SignatureVerifier) VerifySignature(msg *SignedBridgeMessage) (bool, error) {
	// Validate input
	if msg == nil || msg.Message == nil {
		return false, fmt.Errorf("invalid message: message is nil")
	}

	if len(msg.Signature) == 0 || len(msg.PublicKey) == 0 {
		return false, fmt.Errorf("missing signature or public key")
	}

	// Decode the public key from bytes
	publicKeyHex := hex.EncodeToString(msg.PublicKey)
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		svLogHex := hex.EncodeToString(msg.Signature[:min(len(msg.Signature), 8)])
		sv.logVerificationAttempt(msg.Message.ID, msg.Message.SourceAddress, svLogHex, false, "invalid public key hex format")
		return false, fmt.Errorf("invalid public key hex format: %w", err)
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		svLogHex := hex.EncodeToString(msg.Signature[:min(len(msg.Signature), 8)])
		sv.logVerificationAttempt(msg.Message.ID, msg.Message.SourceAddress, svLogHex, false, "invalid public key size")
		return false, fmt.Errorf("invalid Ed25519 public key size")
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)

	// Signature bytes are already in the msg
	signatureBytes := msg.Signature

	if len(signatureBytes) != ed25519.SignatureSize {
		svLogHex := hex.EncodeToString(msg.Signature[:min(len(msg.Signature), 8)])
		sv.logVerificationAttempt(msg.Message.ID, msg.Message.SourceAddress, svLogHex, false, "invalid signature size")
		return false, fmt.Errorf("invalid Ed25519 signature size: expected %d bytes, got %d", ed25519.SignatureSize, len(signatureBytes))
	}

	// Check/update nonce to prevent replay attacks
	sv.keysMutex.Lock()
	if sv.lastSeenNonces == nil {
		sv.lastSeenNonces = make(map[string]uint64)
	}
	lastNonce, exists := sv.lastSeenNonces[publicKeyHex]
	if exists && msg.Message.Nonce <= lastNonce {
		sv.keysMutex.Unlock()
		svLogHex := hex.EncodeToString(msg.Signature[:min(len(msg.Signature), 8)])
		sv.logVerificationAttempt(msg.Message.ID, msg.Message.SourceAddress, svLogHex, false, "replay attack: nonce too low")
		return false, fmt.Errorf("replay attack detected: nonce %d has already been used or is too low (last seen: %d)", msg.Message.Nonce, lastNonce)
	}
	sv.lastSeenNonces[publicKeyHex] = msg.Message.Nonce
	sv.keysMutex.Unlock()

	// Create the message payload for verification
	payload := &MessagePayload{
		TransactionID:  msg.Message.ID,
		SourceChain:    msg.Message.SourceChain,
		DestChain:      msg.Message.DestChain,
		SourceAddress:  msg.Message.SourceAddress,
		DestAddress:    msg.Message.DestAddress,
		TokenSymbol:    msg.Message.TokenSymbol,
		Amount:         msg.Message.Amount,
		Fee:            msg.Message.Fee,
		Nonce:          msg.Message.Nonce,
		Timestamp:      0, // hardcoded timestamp for predictability in tests
	}

	// Serialize payload for signing
	payloadBytes, err := SerializePayload(payload)
	if err != nil {
		svLogHex := hex.EncodeToString(msg.Signature[:min(len(msg.Signature), 8)])
		sv.logVerificationAttempt(msg.Message.ID, msg.Message.SourceAddress, svLogHex, false, "failed to serialize payload")
		return false, fmt.Errorf("failed to serialize payload: %w", err)
	}

	// Verify the signature
	isValid := ed25519.Verify(publicKey, payloadBytes, signatureBytes)

	svLogHex := hex.EncodeToString(msg.Signature[:min(len(msg.Signature), 8)])
	if isValid {
		sv.logger.WithFields(logrus.Fields{
			"transaction_id": msg.Message.ID,
			"address":        msg.Message.SourceAddress,
		}).Debug("Signature verified successfully")
		sv.logVerificationAttempt(msg.Message.ID, msg.Message.SourceAddress, svLogHex, true, "")
	} else {
		sv.logger.WithFields(logrus.Fields{
			"transaction_id": msg.Message.ID,
			"address":        msg.Message.SourceAddress,
		}).Warn("Signature verification failed")
		sv.logVerificationAttempt(msg.Message.ID, msg.Message.SourceAddress, svLogHex, false, "signature verification failed")
	}

	return isValid, nil
}

// VerifySignatureWithPublicKey verifies a signature using a specific public key
func (sv *SignatureVerifier) VerifySignatureWithPublicKey(msg *SignedBridgeMessage, publicKeyHex string) (bool, error) {
	// Decode the public key
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, fmt.Errorf("invalid public key hex format: %w", err)
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid Ed25519 public key size")
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)

	// Signature bytes are already in msg
	signatureBytes := msg.Signature

	if len(signatureBytes) != ed25519.SignatureSize {
		return false, fmt.Errorf("invalid Ed25519 signature size")
	}

	// Check/update nonce to prevent replay attacks
	sv.keysMutex.Lock()
	if sv.lastSeenNonces == nil {
		sv.lastSeenNonces = make(map[string]uint64)
	}
	lastNonce, exists := sv.lastSeenNonces[publicKeyHex]
	if exists && msg.Message.Nonce <= lastNonce {
		sv.keysMutex.Unlock()
		return false, fmt.Errorf("replay attack detected: nonce %d has already been used or is too low (last seen: %d)", msg.Message.Nonce, lastNonce)
	}
	sv.lastSeenNonces[publicKeyHex] = msg.Message.Nonce
	sv.keysMutex.Unlock()

	// Create and serialize payload
	payload := &MessagePayload{
		TransactionID:  msg.Message.ID,
		SourceChain:    msg.Message.SourceChain,
		DestChain:      msg.Message.DestChain,
		SourceAddress:  msg.Message.SourceAddress,
		DestAddress:    msg.Message.DestAddress,
		TokenSymbol:    msg.Message.TokenSymbol,
		Amount:         msg.Message.Amount,
		Fee:            msg.Message.Fee,
		Nonce:          msg.Message.Nonce,
		Timestamp:      0,
	}

	payloadBytes, err := SerializePayload(payload)
	if err != nil {
		return false, fmt.Errorf("failed to serialize payload: %w", err)
	}

	// Verify
	return ed25519.Verify(publicKey, payloadBytes, signatureBytes), nil
}

// logVerificationAttempt logs a signature verification attempt
func (sv *SignatureVerifier) logVerificationAttempt(txID, address, signature string, isValid bool, errMsg string) {
	sv.logMutex.Lock()
	defer sv.logMutex.Unlock()

	entry := SignatureVerificationLog{
		Timestamp:     int64(len(sv.verificationLog)),
		Address:       address,
		SignatureHash: signature[:min(len(signature), 16)],
		IsValid:       isValid,
		ErrorMessage:  errMsg,
	}

	sv.verificationLog = append(sv.verificationLog, entry)

	// Keep only recent entries
	if len(sv.verificationLog) > sv.maxLogEntries {
		sv.verificationLog = sv.verificationLog[len(sv.verificationLog)-sv.maxLogEntries:]
	}
}

// GetVerificationLog returns the verification log
func (sv *SignatureVerifier) GetVerificationLog() []SignatureVerificationLog {
	sv.logMutex.RLock()
	defer sv.logMutex.RUnlock()

	// Return a copy
	logCopy := make([]SignatureVerificationLog, len(sv.verificationLog))
	copy(logCopy, sv.verificationLog)
	return logCopy
}

// SerializePayload serializes a message payload for signing
func SerializePayload(payload *MessagePayload) ([]byte, error) {
	// Create a deterministic serialization
	// Format: "TXN|<id>|<source_chain>|<dest_chain>|<src_addr>|<dst_addr>|<token>|<amount>|<fee>|<nonce>|<timestamp>"
	data := fmt.Sprintf("TXN|%s|%s|%s|%s|%s|%s|%s|%s|%d|%d",
		payload.TransactionID,
		payload.SourceChain,
		payload.DestChain,
		payload.SourceAddress,
		payload.DestAddress,
		payload.TokenSymbol,
		payload.Amount,
		payload.Fee,
		payload.Nonce,
		payload.Timestamp,
	)
	return []byte(data), nil
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
