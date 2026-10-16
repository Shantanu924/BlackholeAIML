// Package schema implements TANTRA canonical transaction contracts.
//
// Phase 3 requirement: NO free-form transaction execution.
// Every transaction entering the system MUST conform to a versioned schema.
// Unknown fields → reject. Missing required fields → reject.
// Canonical serialization ensures replay-safe hash stability.
package schema

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const CurrentVersion = "v1"

// ValidTypes is the exhaustive list of allowed transaction types.
var ValidTypes = map[string]bool{
	"token_transfer": true,
	"transfer":       true,
	"stake_deposit":  true,
	"stake_withdraw": true,
	"mint":           true,
	"burn":           true,
}

// TxContract is the canonical, versioned transaction schema.
// All fields are explicit. No extra fields allowed.
type TxContract struct {
	SchemaVersion string `json:"schema_version"` // must be "v1"
	TraceID       string `json:"trace_id"`       // optional — injected by PDV if absent
	Type          string `json:"type"`           // must be in ValidTypes
	From          string `json:"from"`
	To            string `json:"to"`
	Amount        uint64 `json:"amount"`
	TokenID       string `json:"token_id"`
	Fee           uint64 `json:"fee"`
	Nonce         uint64 `json:"nonce"`
	Timestamp     int64  `json:"timestamp"` // carried for audit, NOT hashed
	Signature     string `json:"signature"`
}

// ValidationError describes a schema rejection.
type ValidationError struct {
	Field   string `json:"field"`
	Reason  string `json:"reason"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("schema violation: field=%s reason=%s", e.Field, e.Reason)
}

// ParseAndValidate decodes raw JSON into a TxContract, rejects unknown fields,
// and validates all required fields. Returns a structured error on any violation.
func ParseAndValidate(raw []byte) (*TxContract, error) {
	// Step 1 — decode into struct with DisallowUnknownFields.
	// This rejects any field not defined in TxContract.
	var contract TxContract
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&contract); err != nil {
		return nil, &ValidationError{Field: "payload", Reason: "malformed JSON or unknown fields: " + err.Error()}
	}

	// Step 3 — validate schema_version.
	if contract.SchemaVersion == "" {
		return nil, &ValidationError{Field: "schema_version", Reason: "required, must be " + CurrentVersion}
	}
	if contract.SchemaVersion != CurrentVersion {
		return nil, &ValidationError{Field: "schema_version", Reason: fmt.Sprintf("unsupported version %q, expected %q", contract.SchemaVersion, CurrentVersion)}
	}

	// Step 4 — validate type.
	if contract.Type == "" {
		return nil, &ValidationError{Field: "type", Reason: "required"}
	}
	if !ValidTypes[contract.Type] {
		validList := make([]string, 0, len(ValidTypes))
		for k := range ValidTypes {
			validList = append(validList, k)
		}
		sort.Strings(validList)
		return nil, &ValidationError{Field: "type", Reason: fmt.Sprintf("unknown type %q, valid: %s", contract.Type, strings.Join(validList, ", "))}
	}

	// Step 5 — validate required address fields.
	if contract.From == "" {
		return nil, &ValidationError{Field: "from", Reason: "required"}
	}
	if contract.To == "" {
		return nil, &ValidationError{Field: "to", Reason: "required"}
	}

	// Step 6 — validate amount.
	if contract.Amount == 0 {
		return nil, &ValidationError{Field: "amount", Reason: "must be greater than 0"}
	}

	// Step 7 — validate token_id.
	if contract.TokenID == "" {
		return nil, &ValidationError{Field: "token_id", Reason: "required"}
	}

	// Step 8 — validate timestamp present.
	if contract.Timestamp == 0 {
		return nil, &ValidationError{Field: "timestamp", Reason: "required, must be current unix timestamp"}
	}

	return &contract, nil
}

// CanonicalJSON produces deterministic JSON for the contract.
// Fields are serialized in a fixed order — replay-safe.
// Timestamp is intentionally excluded from the canonical zone.
func CanonicalJSON(c *TxContract) ([]byte, error) {
	canonical := struct {
		SchemaVersion string `json:"schema_version"`
		TraceID       string `json:"trace_id"`
		Type          string `json:"type"`
		From          string `json:"from"`
		To            string `json:"to"`
		Amount        uint64 `json:"amount"`
		TokenID       string `json:"token_id"`
		Fee           uint64 `json:"fee"`
		Nonce         uint64 `json:"nonce"`
		Signature     string `json:"signature"`
	}{
		SchemaVersion: c.SchemaVersion,
		TraceID:       c.TraceID,
		Type:          c.Type,
		From:          c.From,
		To:            c.To,
		Amount:        c.Amount,
		TokenID:       c.TokenID,
		Fee:           c.Fee,
		Nonce:         c.Nonce,
		Signature:     c.Signature,
	}
	return json.Marshal(canonical)
}

// CanonicalHash returns SHA-256 of the canonical JSON (timestamp excluded).
func CanonicalHash(c *TxContract) (string, error) {
	data, err := CanonicalJSON(c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
