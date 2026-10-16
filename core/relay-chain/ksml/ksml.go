// Package ksml implements the KSML (Knowledge Semantic Markup Language) /
// CET (Canonical Execution Template) upstream contract layer.
//
// KSML/CET is the upstream deterministic contract structure provided by
// Raj Prajapati. This package:
//   1. Defines the KSML contract schema
//   2. Validates KSML contracts (version, intent type, fields)
//   3. Converts KSML contracts to schema v1 TxContracts for PDV routing
//   4. Rejects malformed or unknown KSML contracts
//
// Every wallet-originated or upstream-originated transaction enters TANTRA
// through a KSML contract, which is then converted to a canonical schema v1
// TxContract before PDV enforcement.
//
// KSML contract format:
//   {
//     "ksml_version": "v1",
//     "intent_type":  "transfer" | "stake" | "governance" | "bridge",
//     "source":       "wallet" | "relay" | "bridge" | "governance",
//     "trace_id":     "<optional — injected by PDV if absent>",
//     "payload":      { ... intent-specific fields ... },
//     "meta":         { "timestamp": ..., "nonce": ..., "signature": ... }
//   }
package ksml

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
)

const KSMLVersion = "v1"

// ValidIntentTypes is the exhaustive list of allowed KSML intent types.
var ValidIntentTypes = map[string]bool{
	"transfer":   true,
	"stake":      true,
	"unstake":    true,
	"governance": true,
	"bridge":     true,
	"mint":       true,
	"burn":       true,
}

// ValidSources is the exhaustive list of allowed KSML sources.
var ValidSources = map[string]bool{
	"wallet":     true,
	"relay":      true,
	"bridge":     true,
	"governance": true,
	"system":     true,
}

// KSMLContract is the canonical upstream execution contract.
type KSMLContract struct {
	KSMLVersion string                 `json:"ksml_version"`
	IntentType  string                 `json:"intent_type"`
	Source      string                 `json:"source"`
	TraceID     string                 `json:"trace_id,omitempty"`
	Amount      uint64                 `json:"amount,omitempty"`
	Payload     map[string]interface{} `json:"payload"`
	Meta        KSMLMeta               `json:"meta"`
}

// KSMLMeta carries execution metadata.
type KSMLMeta struct {
	Timestamp int64  `json:"timestamp"`
	Nonce     uint64 `json:"nonce"`
	Signature string `json:"signature,omitempty"`
}

// KSMLError describes a KSML contract rejection.
type KSMLError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (e *KSMLError) Error() string {
	return fmt.Sprintf("ksml violation: field=%s reason=%s", e.Field, e.Reason)
}

// ParseAndValidate decodes raw JSON into a KSMLContract and validates it.
// Rejects unknown fields, wrong version, unknown intent types.
func ParseAndValidate(raw []byte) (*KSMLContract, error) {
	// Reject unknown fields.
	var contract KSMLContract
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&contract); err != nil {
		return nil, &KSMLError{Field: "contract", Reason: "malformed KSML or unknown fields: " + err.Error()}
	}

	// Validate ksml_version.
	if contract.KSMLVersion == "" {
		return nil, &KSMLError{Field: "ksml_version", Reason: "required, must be " + KSMLVersion}
	}
	if contract.KSMLVersion != KSMLVersion {
		return nil, &KSMLError{Field: "ksml_version", Reason: fmt.Sprintf("unsupported version %q", contract.KSMLVersion)}
	}

	// Validate intent_type.
	if contract.IntentType == "" {
		return nil, &KSMLError{Field: "intent_type", Reason: "required"}
	}
	if !ValidIntentTypes[contract.IntentType] {
		return nil, &KSMLError{Field: "intent_type", Reason: fmt.Sprintf("unknown intent type %q", contract.IntentType)}
	}

	// Validate source.
	if contract.Source == "" {
		return nil, &KSMLError{Field: "source", Reason: "required"}
	}
	if !ValidSources[contract.Source] {
		return nil, &KSMLError{Field: "source", Reason: fmt.Sprintf("unknown source %q", contract.Source)}
	}

	// Validate payload exists.
	if contract.Payload == nil {
		return nil, &KSMLError{Field: "payload", Reason: "required"}
	}

	// Validate meta.
	if contract.Meta.Timestamp == 0 {
		return nil, &KSMLError{Field: "meta.timestamp", Reason: "required"}
	}

	return &contract, nil
}

// ToTxContract converts a validated KSMLContract to a schema v1 TxContract
// for PDV routing. This is the canonical mapping from KSML to TANTRA execution.
func ToTxContract(k *KSMLContract) (*schema.TxContract, error) {
	p := k.Payload

	// Extract required fields from payload.
	from, _ := p["from"].(string)
	to, _ := p["to"].(string)
	tokenID, _ := p["token_id"].(string)

	var amount uint64
	if k.Amount > 0 {
		amount = k.Amount
	} else {
		switch v := p["amount"].(type) {
		case float64:
			amount = uint64(v)
		case uint64:
			amount = v
		}
	}

	if from == "" {
		return nil, &KSMLError{Field: "payload.from", Reason: "required"}
	}
	if to == "" {
		return nil, &KSMLError{Field: "payload.to", Reason: "required"}
	}
	if amount == 0 {
		return nil, &KSMLError{Field: "payload.amount", Reason: "must be greater than 0"}
	}
	if tokenID == "" {
		tokenID = "BHIV" // default token
	}

	// Map KSML intent_type to schema tx type.
	txType := intentTypeToTxType(k.IntentType)

	return &schema.TxContract{
		SchemaVersion: schema.CurrentVersion,
		TraceID:       k.TraceID,
		Type:          txType,
		From:          from,
		To:            to,
		Amount:        amount,
		TokenID:       tokenID,
		Nonce:         k.Meta.Nonce,
		Timestamp:     k.Meta.Timestamp,
		Signature:     k.Meta.Signature,
	}, nil
}

// intentTypeToTxType maps KSML intent types to schema v1 transaction types.
func intentTypeToTxType(intentType string) string {
	switch intentType {
	case "transfer":
		return "token_transfer"
	case "stake":
		return "stake_deposit"
	case "unstake":
		return "stake_withdraw"
	case "mint":
		return "mint"
	case "burn":
		return "burn"
	default:
		return "token_transfer"
	}
}
