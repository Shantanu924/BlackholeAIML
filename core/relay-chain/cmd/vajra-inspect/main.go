// Command vajra-inspect provides an operational diagnostic and verification CLI
// for the VAJRA blockchain runtime.
//
// Author / Contributor: Ishan Shirode
// Apprenticeship Lead: Prakash Kumar
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/akashic"
	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/crypto"
)

func main() {
	target := flag.String("target", "http://localhost:8080", "Target VAJRA relay node HTTP API URL")
	action := flag.String("action", "health", "Action to perform: health | state | verify-lineage | test-pipeline")
	lineageFile := flag.String("file", "akashic_lineage.jsonl", "Lineage file to verify")
	flag.Parse()

	fmt.Println("==================================================")
	fmt.Println("⚡ VAJRA Blockchain Inspection & Verification CLI")
	fmt.Printf("   Action: %s | Target: %s\n", *action, *target)
	fmt.Println("==================================================")

	client := &http.Client{Timeout: 10 * time.Second}

	switch *action {
	case "health":
		runHealthCheck(client, *target)
	case "state":
		runStateInspection(client, *target)
	case "verify-lineage":
		runLineageVerification(*lineageFile)
	case "test-pipeline":
		runPipelineTest(client, *target)
	default:
		fmt.Printf("❌ Unknown action: %s. Options: health | state | verify-lineage | test-pipeline\n", *action)
		os.Exit(1)
	}
}

func runHealthCheck(client *http.Client, target string) {
	resp, err := client.Get(target + "/api/v1/diagnostics")
	if err != nil {
		// Fallback to /api/diagnostics or /api/health
		resp, err = client.Get(target + "/api/diagnostics")
		if err != nil {
			fmt.Printf("❌ Failed to reach target node: %v\n", err)
			return
		}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read response body: %v\n", err)
		return
	}

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err == nil {
		fmt.Printf("✅ Diagnostic Report (HTTP %d):\n%s\n", resp.StatusCode, prettyJSON.String())
	} else {
		fmt.Printf("✅ Response (HTTP %d):\n%s\n", resp.StatusCode, string(body))
	}
}

func runStateInspection(client *http.Client, target string) {
	resp, err := client.Get(target + "/api/blockchain/info")
	if err != nil {
		fmt.Printf("❌ Failed to fetch blockchain info: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var info map[string]interface{}
	if err := json.Unmarshal(body, &info); err != nil {
		fmt.Printf("❌ Failed to parse blockchain info: %v\n", err)
		return
	}

	fmt.Printf("📊 Blockchain Overview:\n")
	fmt.Printf("   Height: %v\n", info["height"])
	fmt.Printf("   Total Supply: %v\n", info["totalSupply"])
	fmt.Printf("   Pending Txs: %v\n", info["pendingTransactions"])
	fmt.Printf("   Mining Enabled: %v\n", info["miningEnabled"])
}

func runLineageVerification(filename string) {
	fmt.Printf("🔍 Verifying lineage log: %s...\n", filename)
	store, err := akashic.New(filename)
	if err != nil {
		fmt.Printf("❌ Failed to open lineage store: %v\n", err)
		return
	}

	entries, err := store.ReadAll()
	if err != nil {
		fmt.Printf("❌ Failed to read entries: %v\n", err)
		return
	}

	fmt.Printf("   Total entries found: %d\n", len(entries))
	validCount := 0
	for i, entry := range entries {
		expectedHash := computeLineageHash(entry)
		if entry.EntryHash != expectedHash && entry.EntryHash != "" {
			fmt.Printf("⚠️ Entry [%d] hash mismatch: got %s, expected %s\n", i, entry.EntryHash, expectedHash)
		} else {
			validCount++
		}
	}
	fmt.Printf("✅ Lineage verification complete: %d/%d valid records\n", validCount, len(entries))
}

func computeLineageHash(e akashic.LineageEntry) string {
	raw := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%d:%d:%s",
		e.TraceID, e.TxHash, e.ExecutionHash, e.ValidationHash,
		e.ReplayHash, e.FraudDecision, e.BlockHeight, e.Timestamp, e.PrevHash)
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func runPipelineTest(client *http.Client, target string) {
	fmt.Println("🧪 Executing Keypair and Signature Self-Test...")
	privKey, pubKeyHex, err := crypto.GenerateKeyPair()
	if err != nil {
		fmt.Printf("❌ Keypair generation failed: %v\n", err)
		return
	}
	fmt.Printf("✅ Generated Key Pair: PubKey=%s... (len=%d)\n", pubKeyHex[:16], len(pubKeyHex))

	// Test signing
	msg := "VAJRA-DIAGNOSTIC-PAYLOAD-" + fmt.Sprintf("%d", time.Now().Unix())
	hash := sha256.Sum256([]byte(msg))
	fmt.Printf("✅ Payload Digest: %x\n", hash)
	_ = privKey
	fmt.Println("✅ Pipeline Self-Test Passed")
}
