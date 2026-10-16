// Nonce Coordinator — standalone cross-node nonce governance service.
//
// Run:
//   go run cmd/noncecoord/main.go -port 9200 -ledger global_nonce_ledger.jsonl
//
// Configure relay nodes to use it:
//   set NONCE_COORDINATOR_URL=http://localhost:9200
//
// All relay nodes call POST /nonce/check before accepting any nonce.
// The coordinator maintains a global seen-set across all nodes.
// Duplicate nonce from ANY node → HTTP 409 + NONCE_REPLAY.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/noncecoord"
)

func main() {
	port := flag.Int("port", 9200, "coordinator port")
	ledger := flag.String("ledger", "global_nonce_ledger.jsonl", "global nonce ledger path")
	flag.Parse()

	fmt.Printf("[NONCECOORD] Starting nonce coordinator on port %d\n", *port)
	fmt.Printf("[NONCECOORD] Global ledger: %s\n", *ledger)
	fmt.Printf("[NONCECOORD] Set NONCE_COORDINATOR_URL=http://localhost:%d on all relay nodes\n", *port)

	log.Fatal(noncecoord.StartServer(*port, *ledger))
}
