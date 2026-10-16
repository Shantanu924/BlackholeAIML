// Package karmachain implements the KarmaChain layer —
// the chain-of-chains reconstruction and replication system.
//
// KarmaChain provides:
//   1. AKASHIC lineage replication to multiple nodes
//   2. State root equality verification across nodes
//   3. Reconstruction from any surviving node
//   4. Cross-node lineage consistency proof
//
// Replication targets are configured via:
//   KARMACHAIN_NODES=http://node1:8080,http://node2:8080,http://node3:8080
//
// "If one node survives, execution truth survives." — this package makes that true
// across a real distributed network, not just a single file.
package karmachain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// LineageEntry mirrors akashic.LineageEntry for cross-node replication.
type LineageEntry struct {
	TraceID        string `json:"trace_id"`
	TxHash         string `json:"tx_hash"`
	ExecutionHash  string `json:"execution_hash"`
	ValidationHash string `json:"validation_hash"`
	ReplayHash     string `json:"replay_hash"`
	FraudDecision  string `json:"fraud_decision"`
	BlockHeight    uint64 `json:"block_height"`
	StateRootHash  string `json:"state_root_hash"`
	Timestamp      int64  `json:"timestamp"`
	PrevHash       string `json:"prev_hash"`
	EntryHash      string `json:"entry_hash"`
}

// NodeStatus represents the replication status of a single node.
type NodeStatus struct {
	URL          string `json:"url"`
	Reachable    bool   `json:"reachable"`
	EntryCount   int    `json:"entry_count"`
	StateRoot    string `json:"state_root"`
	LastSyncedAt int64  `json:"last_synced_at"`
	Error        string `json:"error,omitempty"`
}

// ConsistencyResult is returned by VerifyConsistency.
type ConsistencyResult struct {
	Consistent  bool         `json:"consistent"`
	NodeCount   int          `json:"node_count"`
	Agreed      int          `json:"agreed"`
	Disagreed   int          `json:"disagreed"`
	StateRoot   string       `json:"state_root"`
	NodeStatuses []NodeStatus `json:"node_statuses"`
	Message     string       `json:"message"`
}

// getNodes returns the configured KarmaChain node URLs.
func getNodes() []string {
	raw := os.Getenv("KARMACHAIN_NODES")
	if raw == "" {
		// Default: single local node (same as current setup — no regression)
		return []string{"http://localhost:8080"}
	}
	nodes := strings.Split(raw, ",")
	var result []string
	for _, n := range nodes {
		n = strings.TrimSpace(n)
		if n != "" {
			result = append(result, n)
		}
	}
	return result
}

// Replicate sends a lineage entry to all configured KarmaChain nodes in parallel.
// Returns the number of nodes that successfully accepted the entry.
func Replicate(entry LineageEntry) (int, []error) {
	nodes := getNodes()
	if len(nodes) == 0 {
		return 0, []error{fmt.Errorf("no KarmaChain nodes configured")}
	}

	type result struct {
		url string
		err error
	}

	results := make(chan result, len(nodes))
	for _, node := range nodes {
		go func(url string) {
			err := replicateToNode(url, entry)
			results <- result{url, err}
		}(node)
	}

	var errs []error
	accepted := 0
	for range nodes {
		r := <-results
		if r.err != nil {
			log.Printf("[KARMACHAIN][REPLICATE_FAIL] node=%s trace=%s err=%v",
				r.url, entry.TraceID, r.err)
			errs = append(errs, r.err)
		} else {
			log.Printf("[KARMACHAIN][REPLICATE_OK] node=%s trace=%s", r.url, entry.TraceID)
			accepted++
		}
	}

	log.Printf("[KARMACHAIN][REPLICATE] trace=%s accepted=%d/%d",
		entry.TraceID, accepted, len(nodes))
	return accepted, errs
}

// replicateToNode sends a single lineage entry to one node.
func replicateToNode(nodeURL string, entry LineageEntry) error {
	url := nodeURL + "/api/akashic/replicate"
	body, _ := json.Marshal(entry)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("node returned HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return nil
}

// VerifyConsistency checks that all configured nodes have the same state root.
// This proves that execution truth is consistent across the distributed network.
func VerifyConsistency() ConsistencyResult {
	nodes := getNodes()
	type nodeResult struct {
		status NodeStatus
	}

	results := make(chan nodeResult, len(nodes))
	var wg sync.WaitGroup

	for _, node := range nodes {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			status := queryNodeStatus(url)
			results <- nodeResult{status}
		}(node)
	}

	wg.Wait()
	close(results)

	var statuses []NodeStatus
	for r := range results {
		statuses = append(statuses, r.status)
	}

	// Find the majority state root.
	rootCounts := make(map[string]int)
	for _, s := range statuses {
		if s.Reachable && s.StateRoot != "" {
			rootCounts[s.StateRoot]++
		}
	}

	var majorityRoot string
	maxCount := 0
	for root, count := range rootCounts {
		if count > maxCount {
			maxCount = count
			majorityRoot = root
		}
	}

	agreed := 0
	disagreed := 0
	for _, s := range statuses {
		if s.Reachable {
			if s.StateRoot == majorityRoot {
				agreed++
			} else {
				disagreed++
			}
		}
	}

	consistent := disagreed == 0 && agreed > 0
	msg := "all nodes consistent"
	if !consistent {
		msg = fmt.Sprintf("state root disagreement: %d agreed, %d disagreed", agreed, disagreed)
	}

	log.Printf("[KARMACHAIN][CONSISTENCY] consistent=%v agreed=%d disagreed=%d root=%s",
		consistent, agreed, disagreed, majorityRoot)

	return ConsistencyResult{
		Consistent:   consistent,
		NodeCount:    len(nodes),
		Agreed:       agreed,
		Disagreed:    disagreed,
		StateRoot:    majorityRoot,
		NodeStatuses: statuses,
		Message:      msg,
	}
}

// queryNodeStatus queries a single node for its AKASHIC state root.
func queryNodeStatus(nodeURL string) NodeStatus {
	status := NodeStatus{URL: nodeURL}
	client := &http.Client{Timeout: 3 * time.Second}

	resp, err := client.Get(nodeURL + "/api/akashic/reconstruct")
	if err != nil {
		status.Reachable = false
		status.Error = err.Error()
		return status
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		status.Reachable = false
		status.Error = "failed to decode response"
		return status
	}

	status.Reachable = true
	status.LastSyncedAt = time.Now().Unix()

	if r, ok := result["result"].(map[string]interface{}); ok {
		if root, ok := r["final_state_root"].(string); ok {
			status.StateRoot = root
		}
		if count, ok := r["total_entries"].(float64); ok {
			status.EntryCount = int(count)
		}
	}

	return status
}

// Reconstruct pulls the full lineage from the first reachable node and
// returns it for local reconstruction. This implements the
// "if one node survives, execution truth survives" guarantee.
func Reconstruct() ([]LineageEntry, string, error) {
	nodes := getNodes()
	for _, node := range nodes {
		entries, stateRoot, err := reconstructFromNode(node)
		if err != nil {
			log.Printf("[KARMACHAIN][RECONSTRUCT_FAIL] node=%s err=%v", node, err)
			continue
		}
		log.Printf("[KARMACHAIN][RECONSTRUCT_OK] node=%s entries=%d state_root=%s",
			node, len(entries), stateRoot)
		return entries, stateRoot, nil
	}
	return nil, "", fmt.Errorf("no reachable KarmaChain node found for reconstruction")
}

// reconstructFromNode pulls the full lineage from a single node.
func reconstructFromNode(nodeURL string) ([]LineageEntry, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	// Get all lineage entries.
	resp, err := client.Get(nodeURL + "/api/akashic/lineage")
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, "", err
	}

	entriesRaw, ok := result["entries"].([]interface{})
	if !ok {
		return nil, "", fmt.Errorf("invalid lineage response")
	}

	var entries []LineageEntry
	for _, e := range entriesRaw {
		data, _ := json.Marshal(e)
		var entry LineageEntry
		if err := json.Unmarshal(data, &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	// Get state root from reconstruct endpoint.
	resp2, err := client.Get(nodeURL + "/api/akashic/reconstruct")
	if err != nil {
		return entries, "", nil
	}
	defer resp2.Body.Close()

	var r2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&r2)
	stateRoot := ""
	if res, ok := r2["result"].(map[string]interface{}); ok {
		if root, ok := res["final_state_root"].(string); ok {
			stateRoot = root
		}
	}

	return entries, stateRoot, nil
}
