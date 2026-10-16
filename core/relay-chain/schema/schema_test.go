package schema

import (
	"encoding/json"
	"testing"
)

func TestSchema_ParseAndValidate_ValidContract(t *testing.T) {
	contract := TxContract{
		SchemaVersion: "v1",
		TraceID:       "trace-schema-1",
		Type:          "token_transfer",
		From:          "0xAlice",
		To:            "0xBob",
		Amount:        1000,
		TokenID:       "BHIV",
		Fee:           10,
		Nonce:         1,
		Timestamp:     1779717000,
		Signature:     "sig123",
	}

	raw, err := json.Marshal(contract)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	parsed, err := ParseAndValidate(raw)
	if err != nil {
		t.Fatalf("Expected valid contract to pass validation, got error: %v", err)
	}
	if parsed.SchemaVersion != "v1" {
		t.Errorf("Expected schema_version v1, got %s", parsed.SchemaVersion)
	}
}

func TestSchema_ParseAndValidate_UnknownFieldsRejected(t *testing.T) {
	payload := `{
		"schema_version": "v1",
		"type": "token_transfer",
		"from": "0xAlice",
		"to": "0xBob",
		"amount": 100,
		"token_id": "BHIV",
		"nonce": 1,
		"unauthorized_field": "injected_data"
	}`

	_, err := ParseAndValidate([]byte(payload))
	if err == nil {
		t.Fatalf("Expected unknown field 'unauthorized_field' to fail validation, but it passed")
	}
}

func TestSchema_ParseAndValidate_MissingRequiredFields(t *testing.T) {
	// Missing schema_version
	p1 := `{"type": "token_transfer", "from": "0xAlice", "to": "0xBob", "amount": 100}`
	if _, err := ParseAndValidate([]byte(p1)); err == nil {
		t.Errorf("Expected missing schema_version to fail validation")
	}

	// Invalid type
	p2 := `{"schema_version": "v1", "type": "invalid_type", "from": "0xAlice", "to": "0xBob", "amount": 100}`
	if _, err := ParseAndValidate([]byte(p2)); err == nil {
		t.Errorf("Expected unknown type to fail validation")
	}

	// Missing amount (0 amount)
	p3 := `{"schema_version": "v1", "type": "token_transfer", "from": "0xAlice", "to": "0xBob", "amount": 0}`
	if _, err := ParseAndValidate([]byte(p3)); err == nil {
		t.Errorf("Expected 0 amount to fail validation")
	}
}
