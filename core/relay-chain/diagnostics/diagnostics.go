// Package diagnostics provides production-grade node health and subsystem diagnostics
// for the VAJRA blockchain runtime.
//
// Author / Contributor: Ishan Shirode
// Apprenticeship Lead: Prakash Kumar
package diagnostics

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/chain"
)

// SubsystemHealth represents the health status of an individual subsystem.
type SubsystemHealth struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "HEALTHY" | "DEGRADED" | "UNHEALTHY"
	Details string `json:"details"`
}

// SystemDiagnostics represents a complete diagnostic snapshot of a running VAJRA node.
type SystemDiagnostics struct {
	NodeID          string            `json:"node_id"`
	Version         string            `json:"version"`
	Timestamp       int64             `json:"timestamp"`
	UptimeSeconds   int64             `json:"uptime_seconds"`
	GoVersion       string            `json:"go_version"`
	NumGoroutine    int               `json:"num_goroutines"`
	MemoryAllocMB   float64           `json:"memory_alloc_mb"`
	BlockchainState BlockchainSummary `json:"blockchain_state"`
	Subsystems      []SubsystemHealth `json:"subsystems"`
	IntegrityHash   string            `json:"integrity_hash"`
	OverallStatus   string            `json:"overall_status"`
}

// BlockchainSummary provides high-level metrics of the active ledger.
type BlockchainSummary struct {
	BlockHeight        uint64 `json:"block_height"`
	LatestBlockHash    string `json:"latest_block_hash"`
	PendingTxCount     int    `json:"pending_tx_count"`
	TotalSupply        uint64 `json:"total_supply"`
	ActiveValidators   int    `json:"active_validators"`
	TotalStake         uint64 `json:"total_stake"`
	LatestBlockMerkle  string `json:"latest_block_merkle"`
	StateRoot          string `json:"state_root"`
	P2PConnectedPeers  int    `json:"p2p_connected_peers"`
}

var startTime = time.Now()

// RunDiagnostics executes a non-intrusive, read-only diagnostic check on the live blockchain instance.
func RunDiagnostics(bc *chain.Blockchain, nodeID string) SystemDiagnostics {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	uptime := int64(time.Since(startTime).Seconds())
	if nodeID == "" {
		nodeID = os.Getenv("NODE_ID")
		if nodeID == "" {
			nodeID = "vajra-node-primary"
		}
	}

	subsystems := make([]SubsystemHealth, 0)
	overallStatus := "HEALTHY"

	// 1. Check Blockchain Subsystem
	bcStatus := "HEALTHY"
	bcDetails := "Ledger is synchronized and responding"
	var blockHeight uint64
	var latestHash, latestMerkle, stateRoot string
	var pendingCount, peerCount int
	var totalSupply, totalStake uint64

	if bc == nil {
		bcStatus = "UNHEALTHY"
		bcDetails = "Blockchain instance is nil"
		overallStatus = "UNHEALTHY"
	} else {
		blockHeight = uint64(len(bc.Blocks))
		pendingCount = len(bc.PendingTxs)
		totalSupply = bc.TotalSupply
		if len(bc.Blocks) > 0 {
			latestBlock := bc.Blocks[len(bc.Blocks)-1]
			latestHash = latestBlock.Hash
			latestMerkle = latestBlock.Header.MerkleRoot
			stateRoot = latestBlock.Header.StateRoot
		}

		if bc.P2PNode != nil && bc.P2PNode.Host != nil {
			peerCount = len(bc.P2PNode.Host.Network().Peers())
		}

		if bc.StakeLedger != nil {
			for _, v := range bc.StakeLedger.Validators {
				if v.Active {
					totalStake += v.Stake
				}
			}
		}
	}
	subsystems = append(subsystems, SubsystemHealth{
		Name:    "core_ledger",
		Status:  bcStatus,
		Details: bcDetails,
	})

	// 2. Check Akashic Lineage File
	akashicStatus := "HEALTHY"
	akashicDetails := "Akashic lineage store accessible"
	if _, err := os.Stat("akashic_lineage.jsonl"); err != nil {
		akashicStatus = "DEGRADED"
		akashicDetails = fmt.Sprintf("Akashic lineage file check: %v", err)
	}
	subsystems = append(subsystems, SubsystemHealth{
		Name:    "akashic_lineage",
		Status:  akashicStatus,
		Details: akashicDetails,
	})

	// 3. Check TruthStore Log File
	truthStatus := "HEALTHY"
	truthDetails := "Truth store log accessible"
	if _, err := os.Stat("tantra_truth.jsonl"); err != nil {
		truthStatus = "DEGRADED"
		truthDetails = fmt.Sprintf("Truth store file check: %v", err)
	}
	subsystems = append(subsystems, SubsystemHealth{
		Name:    "truth_store",
		Status:  truthStatus,
		Details: truthDetails,
	})

	// 4. Check Nonce Ledger
	nonceStatus := "HEALTHY"
	nonceDetails := "Nonce ledger accessible"
	if _, err := os.Stat("nonce_ledger.jsonl"); err != nil {
		nonceStatus = "DEGRADED"
		nonceDetails = fmt.Sprintf("Nonce ledger check: %v", err)
	}
	subsystems = append(subsystems, SubsystemHealth{
		Name:    "nonce_ledger",
		Status:  nonceStatus,
		Details: nonceDetails,
	})

	activeValCount := 0
	if bc != nil && bc.StakeLedger != nil {
		activeValCount = len(bc.StakeLedger.Validators)
	}

	summary := BlockchainSummary{
		BlockHeight:       blockHeight,
		LatestBlockHash:   latestHash,
		PendingTxCount:    pendingCount,
		TotalSupply:       totalSupply,
		ActiveValidators:  activeValCount,
		TotalStake:        totalStake,
		LatestBlockMerkle: latestMerkle,
		StateRoot:         stateRoot,
		P2PConnectedPeers: peerCount,
	}

	diag := SystemDiagnostics{
		NodeID:          nodeID,
		Version:         "v2.4.0-mainnet-converged",
		Timestamp:       time.Now().Unix(),
		UptimeSeconds:   uptime,
		GoVersion:       runtime.Version(),
		NumGoroutine:    runtime.NumGoroutine(),
		MemoryAllocMB:   float64(mem.Alloc) / (1024 * 1024),
		BlockchainState: summary,
		Subsystems:      subsystems,
		OverallStatus:   overallStatus,
	}

	// Compute tamper-evident integrity hash
	h := sha256.New()
	fmt.Fprintf(h, "%s:%d:%s:%d:%d", diag.NodeID, diag.Timestamp, diag.BlockchainState.LatestBlockHash, diag.BlockchainState.BlockHeight, diag.NumGoroutine)
	diag.IntegrityHash = hex.EncodeToString(h.Sum(nil))

	return diag
}
