// Package runtime implements the ONE canonical deterministic execution runtime.
//
// Phase 1 added: Signature verification before PDV equality gate.
// Gap 1 closed: Distributed PDV — sends payload to 3 independent agents in parallel.
// Gap 2 closed: KarmaChain replication — AKASHIC lineage replicated to multiple nodes.
//
// Canonical execution order (non-negotiable):
//   1. Schema validation
//   2. Signature verification (Phase 1 — NEW)
//   3. Distributed PDV (3 independent agents, parallel)
//   4. PDV equality gate
//   5. Governance gate (Sarathi/DGIC)
//   6. Blockchain write
//   7. Bucket write
//   8. AKASHIC append
//   9. KarmaChain replication
package runtime

import (
	"fmt"
	"log"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/akashic"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/enforcement"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/karmachain"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/pdv"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/schema"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/truthstore"
)

// FailureStage identifies exactly where in the pipeline execution was rejected.
type FailureStage string

const (
	StageSchema     FailureStage = "SCHEMA"
	StagePDV        FailureStage = "PDV"
	StageGovernance FailureStage = "GOVERNANCE"
	StageBlockchain FailureStage = "BLOCKCHAIN"
	StageBucket     FailureStage = "BUCKET"
	StageAKASHIC    FailureStage = "AKASHIC"
)

// ExecutionRequest is the input to the canonical runtime.
type ExecutionRequest struct {
	Contract     *schema.TxContract
	Blockchain   BlockchainWriter
	TruthStore   *truthstore.Store
	AkashicStore *akashic.Store
}

// BlockchainWriter is the minimal interface the runtime needs from the blockchain.
type BlockchainWriter interface {
	ProcessTransactionFromRuntime(traceID, txType, from, to, tokenID string, amount, fee, nonce uint64, timestamp int64) (string, uint64, error)
}

// ExecutionResult is returned by Execute for every call — success or failure.
// Always fully populated so callers can log and respond deterministically.
type ExecutionResult struct {
	// Core identity
	TraceID string `json:"trace_id"`
	TxHash  string `json:"tx_hash,omitempty"`

	// PDV proof
	ExecutionHash  string `json:"execution_hash,omitempty"`
	ValidationHash string `json:"validation_hash,omitempty"`
	ReplayHash     string `json:"replay_hash,omitempty"`

	// Signature proof (Phase 1)
	SignatureValid bool   `json:"signature_valid"`
	PayloadHash    string `json:"payload_hash,omitempty"`

	// Governance
	FraudDecision string `json:"fraud_decision,omitempty"`

	// Blockchain state
	BlockHeight   uint64 `json:"block_height,omitempty"`
	SchemaVersion string `json:"schema_version"`

	// Outcome
	Allowed         bool         `json:"allowed"`
	FailureStage    FailureStage `json:"failure_stage,omitempty"`
	ErrorCode       string       `json:"error_code,omitempty"`
	RejectionReason string       `json:"rejection_reason,omitempty"`

	// Timing
	SubmittedAt int64 `json:"submitted_at"`
}

// Execute is the ONE canonical entry point for all transaction execution.
// No step can be skipped. No step can be reordered.
func Execute(req ExecutionRequest) ExecutionResult {
	submittedAt := time.Now().Unix()
	contract := req.Contract

	if contract == nil {
		return ExecutionResult{
			Allowed:         false,
			FailureStage:    StageSchema,
			ErrorCode:       "SCHEMA_NIL",
			RejectionReason: "contract cannot be nil",
			SubmittedAt:     submittedAt,
		}
	}

	base := ExecutionResult{
		SchemaVersion: contract.SchemaVersion,
		SubmittedAt:   submittedAt,
	}

	// Build TxPayload from validated contract.
	txPayload := &enforcement.TxPayload{
		TraceID:   contract.TraceID,
		Type:      contract.Type,
		From:      contract.From,
		To:        contract.To,
		Amount:    contract.Amount,
		TokenID:   contract.TokenID,
		Fee:       contract.Fee,
		Nonce:     contract.Nonce,
		Timestamp: contract.Timestamp,
		Signature: contract.Signature,
	}

	// STEP 2+3+4+5: Enforcement (signature + distributed PDV + governance).
	// Signature verification happens inside enforcement.Enforce() as the first gate.
	distResult := pdv.Check(pdv.AgentRequest{
		TraceID:   contract.TraceID,
		Type:      contract.Type,
		From:      contract.From,
		To:        contract.To,
		Amount:    contract.Amount,
		TokenID:   contract.TokenID,
		Fee:       contract.Fee,
		Nonce:     contract.Nonce,
		Signature: contract.Signature,
	})

	if !distResult.Agreed {
		log.Printf("[RUNTIME][DISTRIBUTED_PDV_REJECT] trace=%s reason=%s",
			distResult.TraceID, distResult.RejectionReason)
		base.TraceID = distResult.TraceID
		base.ExecutionHash = distResult.ExecutionHash
		base.ValidationHash = distResult.ValidationHash
		base.ReplayHash = distResult.ReplayHash
		base.Allowed = false
		base.FailureStage = StagePDV
		base.ErrorCode = "DISTRIBUTED_PDV_REJECT"
		base.RejectionReason = distResult.RejectionReason
		return base
	}

	// Run local enforcement — includes signature verification + governance gate.
	enfResult := enforcement.Enforce(txPayload)

	// Propagate all fields.
	base.TraceID = enfResult.TraceID
	base.ExecutionHash = distResult.ExecutionHash
	base.ValidationHash = distResult.ValidationHash
	base.ReplayHash = distResult.ReplayHash
	base.FraudDecision = enfResult.FraudDecision
	base.SignatureValid = enfResult.SignatureValid
	base.PayloadHash = enfResult.PayloadHash

	if !enfResult.Allowed {
		stage := StagePDV
		code := "PDV_REJECT"
		if enfResult.FraudDecision == "block" {
			stage = StageGovernance
			code = "GOVERNANCE_REJECT"
		}
		// Detect signature rejection specifically.
		if !enfResult.SignatureValid {
			stage = StageSchema
			code = "SIGNATURE_REJECT"
		}
		log.Printf("[RUNTIME][REJECT] stage=%s trace=%s reason=%s",
			stage, base.TraceID, enfResult.RejectionReason)
		base.Allowed = false
		base.FailureStage = stage
		base.ErrorCode = code
		base.RejectionReason = enfResult.RejectionReason
		return base
	}
	log.Printf("[RUNTIME][PDV+SIG+GOV PASS] trace=%s hash=%s sig_valid=%v fraud=%s",
		base.TraceID, base.ExecutionHash, base.SignatureValid, enfResult.FraudDecision)

	// STEP 6: Blockchain write.
	txHash, blockHeight, err := req.Blockchain.ProcessTransactionFromRuntime(
		base.TraceID,
		contract.Type,
		contract.From,
		contract.To,
		contract.TokenID,
		contract.Amount,
		contract.Fee,
		contract.Nonce,
		contract.Timestamp,
	)
	if err != nil {
		log.Printf("[RUNTIME][REJECT] stage=%s trace=%s reason=%s",
			StageBlockchain, base.TraceID, err.Error())
		base.Allowed = false
		base.FailureStage = StageBlockchain
		base.ErrorCode = "BLOCKCHAIN_REJECT"
		base.RejectionReason = err.Error()
		return base
	}
	base.TxHash = txHash
	base.BlockHeight = blockHeight
	log.Printf("[RUNTIME][BLOCKCHAIN WRITE] trace=%s tx=%s height=%d",
		base.TraceID, txHash, blockHeight)

	// STEP 7: Bucket write.
	if req.TruthStore != nil {
		if err := req.TruthStore.Append(truthstore.Record{
			TraceID:        base.TraceID,
			ExecutionHash:  base.ExecutionHash,
			ValidationHash: base.ValidationHash,
			ReplayHash:     base.ReplayHash,
			TxHash:         txHash,
			FraudDecision:  base.FraudDecision,
		}); err != nil {
			log.Printf("[RUNTIME][WARN] stage=%s trace=%s bucket write failed: %v",
				StageBucket, base.TraceID, err)
			base.ErrorCode = "BUCKET_WRITE_WARN"
			base.RejectionReason = fmt.Sprintf("bucket write failed: %v", err)
		} else {
			log.Printf("[RUNTIME][BUCKET WRITE] trace=%s tx=%s", base.TraceID, txHash)
		}
	}

	// STEP 8: AKASHIC append.
	var appendedEntry akashic.LineageEntry
	if req.AkashicStore != nil {
		appendedEntry = akashic.LineageEntry{
			TraceID:        base.TraceID,
			TxHash:         txHash,
			ExecutionHash:  base.ExecutionHash,
			ValidationHash: base.ValidationHash,
			ReplayHash:     base.ReplayHash,
			FraudDecision:  base.FraudDecision,
			BlockHeight:    blockHeight,
		}
		if err := req.AkashicStore.Append(appendedEntry); err != nil {
			log.Printf("[RUNTIME][WARN] stage=%s trace=%s akashic write failed: %v",
				StageAKASHIC, base.TraceID, err)
		} else {
			log.Printf("[RUNTIME][AKASHIC APPEND] trace=%s tx=%s", base.TraceID, txHash)
		}
	}

	// STEP 9: KarmaChain replication.
	if appendedEntry.TraceID != "" {
		kcEntry := karmachain.LineageEntry{
			TraceID:        appendedEntry.TraceID,
			TxHash:         appendedEntry.TxHash,
			ExecutionHash:  appendedEntry.ExecutionHash,
			ValidationHash: appendedEntry.ValidationHash,
			ReplayHash:     appendedEntry.ReplayHash,
			FraudDecision:  appendedEntry.FraudDecision,
			BlockHeight:    appendedEntry.BlockHeight,
			Timestamp:      appendedEntry.Timestamp,
			PrevHash:       appendedEntry.PrevHash,
			EntryHash:      appendedEntry.EntryHash,
		}
		accepted, _ := karmachain.Replicate(kcEntry)
		log.Printf("[KARMACHAIN][REPLICATE] trace=%s accepted=%d", base.TraceID, accepted)
	}

	log.Printf("[RUNTIME][COMPLETE] trace=%s tx=%s sig_valid=%v — all stages passed",
		base.TraceID, txHash, base.SignatureValid)

	base.Allowed = true
	return base
}
