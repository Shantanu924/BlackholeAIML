package sigverify

import (
	"encoding/hex"
	"os"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
)

func TestSigVerify_SignAndVerifySuccess(t *testing.T) {
	// Generate a secp256k1 keypair
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	pubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())
	privKeyHex := hex.EncodeToString(privKey.Serialize())

	req := VerifyRequest{
		TraceID: "trace-test-1",
		Type:    "token_transfer",
		From:    pubKeyHex,
		To:      "0xRecipientAddress",
		Amount:  1000,
		TokenID: "BHIV",
		Fee:     10,
		Nonce:   1,
	}

	// Sign payload
	sigHex, err := Sign(req, privKeyHex)
	if err != nil {
		t.Fatalf("Failed to sign request: %v", err)
	}

	req.Signature = sigHex

	// Verify signature
	res := Verify(req)
	if !res.Valid {
		t.Fatalf("Expected signature to be valid, got rejected: %s", res.RejectionReason)
	}
	if res.SignerAddress != pubKeyHex {
		t.Errorf("Expected signer address %s, got %s", pubKeyHex, res.SignerAddress)
	}
}

func TestSigVerify_TamperedPayloadRejection(t *testing.T) {
	privKey, _ := btcec.NewPrivateKey()
	pubKeyHex := hex.EncodeToString(privKey.PubKey().SerializeCompressed())
	privKeyHex := hex.EncodeToString(privKey.Serialize())

	req := VerifyRequest{
		TraceID: "trace-test-2",
		Type:    "token_transfer",
		From:    pubKeyHex,
		To:      "0xRecipientAddress",
		Amount:  1000,
		TokenID: "BHIV",
		Fee:     10,
		Nonce:   2,
	}

	sigHex, err := Sign(req, privKeyHex)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}
	req.Signature = sigHex

	// Tamper with amount after signing
	req.Amount = 99999

	res := Verify(req)
	if res.Valid {
		t.Fatalf("Expected tampered payload to fail signature verification, but it passed")
	}
}

func TestSigVerify_MissingSignatureStrictMode(t *testing.T) {
	os.Setenv("SIGVERIFY_STRICT_MODE", "true")
	defer os.Unsetenv("SIGVERIFY_STRICT_MODE")

	req := VerifyRequest{
		TraceID: "trace-test-3",
		Type:    "token_transfer",
		From:    "0xSomeAddress",
		To:      "0xRecipient",
		Amount:  100,
		Nonce:   3,
	}

	res := Verify(req)
	if res.Valid {
		t.Fatalf("Expected missing signature in strict mode to fail verification")
	}
}
