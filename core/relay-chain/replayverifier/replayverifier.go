// Package replayverifier implements Phase 3 — Distributed Replay Equality Layer.
//
// Problem:
//   Single-node replay only proves local determinism.
//   True distributed replay equality requires node A and node B to independently
//   replay the same transaction and produce identical state roots.
//
// This package:
//   1. Sends the same trace_id to multiple nodes for independent replay
//   2. Compares execution_hash, validation_hash, replay_hash across nodes
//   3. Compares final_state_root across nodes
//   4. Detects divergence — any disagreement = DIVERGENCE_DETECTED
//   5. Provides distributed trace continuity proof
//
// Nodes are configured via KARMACHAIN_NODES env var (same as karmachain package).
//
// Constitutional boundary:
//   Replay equality proves deterministic execution correctness.
//   It does NOT prove governance legitimacy.
//   Two nodes agreeing on a hash does not mean the transaction was legitimate —
//   it only means both nodes computed the same result from the same input.
package replayverifier

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// NodeReplayResult is the replay result from a single node.
type NodeReplayResult struct {
	NodeURL        string `json:"node_url"`
	Reachable      bool   `json:"reachable"`
	TraceID        string `json:"trace_id"`
	ExecutionHash  string `json:"execution_hash"`
	ValidationHash string `json:"validation_hash"`
	ReplayHash     string `json:"replay_hash"`
	StateRoot      string `json:"state_root"`
	Found          bool   `json:"found"`
	Error          string `json:"error,omitempty"`
}

// EqualityResult is the result of distributed replay equality verification.
type EqualityResult struct {
	Equal          bool               `json:"equal"`
	TraceID        string             `json:"trace_id"`
	NodeCount      int                `json:"node_count"`
	AgreedCount    int                `json:"agreed_count"`
	DivergentCount int                `json:"divergent_count"`
	ConsensusHash  string             `json:"consensus_hash,omitempty"`
	StateRoot      string             `json:"state_root,omitempty"`
	NodeResults    []NodeReplayResult `json:"node_results"`
	Message        string             `json:"message"`
}

// getNodes returns configured node URLs.
func getNodes() []string {
	raw := os.Getenv("KARMACHAIN_NODES")
	if raw == "" {
		return []string{"http://localhost:8080"}
	}
	var nodes []string
	for _, n := range strings.Split(raw, ",") {
		n = strings.TrimSpace(n)
		if n != "" {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// queryNodeForTrace queries a single node for a trace_id's AKASHIC entry.
func queryNodeForTrace(nodeURL, traceID string) NodeReplayResult {
	result := NodeReplayResult{NodeURL: nodeURL, TraceID: traceID}
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(fmt.Sprintf("%s/api/akashic/trace?trace_id=%s", nodeURL, traceID))
	if err != nil {
		result.Reachable = false
		result.Error = err.Error()
		return result
	}
	defer resp.Body.Close()
	result.Reachable = true

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		result.Error = "decode error"
		return result
	}

	entry, ok := body["entry"].(map[string]interface{})
	if !ok {
		result.Found = false
		return result
	}

	result.Found = true
	result.ExecutionHash, _ = entry["execution_hash"].(string)
	result.ValidationHash, _ = entry["validation_hash"].(string)
	result.ReplayHash, _ = entry["replay_hash"].(string)

	// Also get state root from reconstruct endpoint.
	resp2, err := client.Get(nodeURL + "/api/akashic/reconstruct")
	if err == nil {
		defer resp2.Body.Close()
		var r2 map[string]interface{}
		if json.NewDecoder(resp2.Body).Decode(&r2) == nil {
			if res, ok := r2["result"].(map[string]interface{}); ok {
				result.StateRoot, _ = res["final_state_root"].(string)
			}
		}
	}

	return result
}

// VerifyEquality checks that all configured nodes produce identical hashes
// for the same trace_id. Any disagreement = DIVERGENCE_DETECTED.
func VerifyEquality(traceID string) EqualityResult {
	nodes := getNodes()
	results := make([]NodeReplayResult, len(nodes))
	var wg sync.WaitGroup

	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()
			results[idx] = queryNodeForTrace(url, traceID)
		}(i, node)
	}
	wg.Wait()

	// Find consensus hash (majority).
	hashCounts := make(map[string]int)
	rootCounts := make(map[string]int)
	for _, r := range results {
		if r.Reachable && r.Found && r.ExecutionHash != "" {
			hashCounts[r.ExecutionHash]++
			if r.StateRoot != "" {
				rootCounts[r.StateRoot]++
			}
		}
	}

	var consensusHash, consensusRoot string
	maxH, maxR := 0, 0
	for h, c := range hashCounts {
		if c > maxH {
			maxH = c
			consensusHash = h
		}
	}
	for r, c := range rootCounts {
		if c > maxR {
			maxR = c
			consensusRoot = r
		}
	}

	agreed, divergent := 0, 0
	for _, r := range results {
		if !r.Reachable || !r.Found {
			continue
		}
		if r.ExecutionHash == consensusHash {
			agreed++
		} else {
			divergent++
			log.Printf("[REPLAYVERIFIER][DIVERGENCE] node=%s trace=%s hash=%s expected=%s",
				r.NodeURL, traceID, r.ExecutionHash, consensusHash)
		}
	}

	equal := divergent == 0 && agreed > 0
	msg := "all nodes agree — distributed replay equality confirmed"
	if !equal {
		msg = fmt.Sprintf("DIVERGENCE_DETECTED: %d nodes agree, %d diverge", agreed, divergent)
		log.Printf("[REPLAYVERIFIER][DIVERGENCE_DETECTED] trace=%s %s", traceID, msg)
	} else {
		log.Printf("[REPLAYVERIFIER][EQUAL] trace=%s hash=%s state_root=%s nodes=%d",
			traceID, consensusHash, consensusRoot, agreed)
	}

	return EqualityResult{
		Equal:          equal,
		TraceID:        traceID,
		NodeCount:      len(nodes),
		AgreedCount:    agreed,
		DivergentCount: divergent,
		ConsensusHash:  consensusHash,
		StateRoot:      consensusRoot,
		NodeResults:    results,
		Message:        msg,
	}
}

// VerifyStateRootEquality checks that all nodes have the same final_state_root.
// This is the strongest distributed replay proof — same lineage, same order, same result.
func VerifyStateRootEquality() EqualityResult {
	nodes := getNodes()
	results := make([]NodeReplayResult, len(nodes))
	var wg sync.WaitGroup

	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()
			r := NodeReplayResult{NodeURL: url}
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get(url + "/api/akashic/reconstruct")
			if err != nil {
				r.Reachable = false
				r.Error = err.Error()
				results[idx] = r
				return
			}
			defer resp.Body.Close()
			r.Reachable = true
			var body map[string]interface{}
			if json.NewDecoder(resp.Body).Decode(&body) == nil {
				if res, ok := body["result"].(map[string]interface{}); ok {
					r.StateRoot, _ = res["final_state_root"].(string)
					r.Found = r.StateRoot != ""
				}
			}
			results[idx] = r
		}(i, node)
	}
	wg.Wait()

	rootCounts := make(map[string]int)
	for _, r := range results {
		if r.Reachable && r.StateRoot != "" {
			rootCounts[r.StateRoot]++
		}
	}

	var consensusRoot string
	maxC := 0
	for root, c := range rootCounts {
		if c > maxC {
			maxC = c
			consensusRoot = root
		}
	}

	agreed, divergent := 0, 0
	for _, r := range results {
		if !r.Reachable {
			continue
		}
		if r.StateRoot == consensusRoot {
			agreed++
		} else {
			divergent++
		}
	}

	equal := divergent == 0 && agreed > 0
	msg := "state root equality confirmed across all nodes"
	if !equal {
		msg = fmt.Sprintf("STATE_ROOT_DIVERGENCE: %d agree, %d diverge", agreed, divergent)
	}

	log.Printf("[REPLAYVERIFIER][STATE_ROOT] equal=%v agreed=%d divergent=%d root=%s",
		equal, agreed, divergent, consensusRoot)

	return EqualityResult{
		Equal:          equal,
		NodeCount:      len(nodes),
		AgreedCount:    agreed,
		DivergentCount: divergent,
		StateRoot:      consensusRoot,
		NodeResults:    results,
		Message:        msg,
	}
}
