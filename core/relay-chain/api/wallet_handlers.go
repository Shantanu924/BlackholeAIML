package api

import (
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"

	"github.com/btcsuite/btcd/btcec/v2"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/sigverify"
)

func (s *APIServer) handleWalletKeygen(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "POST required"})
		return
	}
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "keygen failed"})
		return
	}
	pubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())
	privKeyHex := hex.EncodeToString(privKey.Serialize())
	log.Printf("[KEYGEN] new wallet address=%s", pubKeyHex[:16])
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"private_key": privKeyHex,
		"public_key":  pubKeyHex,
		"address":     pubKeyHex,
		"warning":     "Store private_key securely. It will not be shown again.",
	})
}

// handleWalletSign signs a transaction payload with a private key.
// POST /api/wallet/sign
// Body: { private_key, trace_id, type, from, to, amount, token_id, fee, nonce }
// Returns: { signature, payload_hash } — include signature in relay/submit body.
func (s *APIServer) handleWalletSign(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "POST required"})
		return
	}
	var req struct {
		PrivateKey string `json:"private_key"`
		TraceID    string `json:"trace_id"`
		Type       string `json:"type"`
		From       string `json:"from"`
		To         string `json:"to"`
		Amount     uint64 `json:"amount"`
		TokenID    string `json:"token_id"`
		Fee        uint64 `json:"fee"`
		Nonce      uint64 `json:"nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "invalid request"})
		return
	}
	if req.PrivateKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "private_key required"})
		return
	}
	sig, err := sigverify.Sign(sigverify.VerifyRequest{
		TraceID: req.TraceID, Type: req.Type, From: req.From, To: req.To,
		Amount: req.Amount, TokenID: req.TokenID, Fee: req.Fee, Nonce: req.Nonce,
	}, req.PrivateKey)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	payloadHash := sigverify.PayloadHash(sigverify.VerifyRequest{
		TraceID: req.TraceID, Type: req.Type, From: req.From, To: req.To,
		Amount: req.Amount, TokenID: req.TokenID, Fee: req.Fee, Nonce: req.Nonce,
	})
	log.Printf("[SIGN] from=%s nonce=%d", req.From[:min(16, len(req.From))], req.Nonce)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"signature":    sig,
		"payload_hash": payloadHash,
	})
}

// handleWalletPayloadHash returns the canonical payload hash without signing.
// POST /api/wallet/payload-hash — useful for client-side signing.
func (s *APIServer) handleWalletPayloadHash(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "POST required"})
		return
	}
	var req sigverify.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": "invalid request"})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":      true,
		"payload_hash": sigverify.PayloadHash(req),
	})
}

// handleAIFraudStatus returns AI fraud detection status
