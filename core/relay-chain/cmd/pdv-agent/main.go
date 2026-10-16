// PDV Agent — standalone process for distributed PDV validation.
//
// Run three instances on separate ports for true distributed PDV:
//
//   Terminal 1: go run cmd/pdv-agent/main.go -port 9101 -agent ExecutionAgent
//   Terminal 2: go run cmd/pdv-agent/main.go -port 9102 -agent ValidationAgent
//   Terminal 3: go run cmd/pdv-agent/main.go -port 9103 -agent ReplayAgent
//
// Then set env vars before starting the relay node:
//   set PDV_EXECUTION_AGENT_URL=http://localhost:9101/pdv/execute
//   set PDV_VALIDATION_AGENT_URL=http://localhost:9102/pdv/validate
//   set PDV_REPLAY_AGENT_URL=http://localhost:9103/pdv/replay
//
// Each agent runs in its own OS process — no shared memory, no shared goroutines.
// Hash equality across three separate processes proves distributed determinism.
package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/pdv"
)

func main() {
	port := flag.Int("port", 9101, "port to listen on")
	agentName := flag.String("agent", "ExecutionAgent", "agent name: ExecutionAgent | ValidationAgent | ReplayAgent")
	flag.Parse()

	fmt.Printf("[PDV-AGENT] Starting %s on port %d\n", *agentName, *port)
	fmt.Printf("[PDV-AGENT] Separate OS process — no shared memory with relay node\n")
	fmt.Printf("[PDV-AGENT] Hash equality with other agents proves distributed determinism\n")

	// StartAgentServer blocks — runs until killed.
	log.Fatal(pdv.StartAgentServerErr(*port, *agentName))
}
