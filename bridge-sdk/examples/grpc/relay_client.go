package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// RelayMessage represents the transaction payload sent to the bridge
type RelayMessage struct {
	EventHash   string    `json:"eventHash"`
	SrcChain    string    `json:"srcChain"`
	DstChain    string    `json:"dstChain"`
	TxHash      string    `json:"txHash"`
	Amount      string    `json:"amount"`
	TokenSymbol string    `json:"tokenSymbol"`
	Sender      string    `json:"sender"`
	Recipient   string    `json:"recipient"`
	Signature   string    `json:"signature"`
	Timestamp   time.Time `json:"timestamp"`
}

func main() {
	fmt.Println("🌉 Starting BlackHole Bridge client demonstration...")

	// 1. HTTP REST Demonstration
	fmt.Println("\n--- HTTP REST Demonstration ---")
	err := demoREST()
	if err != nil {
		fmt.Printf("HTTP REST Demo Error: %v\n", err)
	}

	// 2. gRPC Demonstration Setup
	fmt.Println("\n--- gRPC Demonstration Setup ---")
	demoGRPC()
}

func demoREST() error {
	url := "http://localhost:8084/relay/eth"
	fmt.Printf("Sending relay request to %s...\n", url)

	// Create a sample transaction message
	message := RelayMessage{
		EventHash:   "0x7f83b1c6e9a8f4c2d3d9e8b7a6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7",
		SrcChain:    "ethereum",
		DstChain:    "blackhole",
		TxHash:      "0x8f7c6b5a4d3c2b1a0e9f8d7c6b5a4d3c2b1a0e9f8d7c6b5a4d3c2b1a0e9f8d7c",
		Amount:      "100.50",
		TokenSymbol: "USDC",
		Sender:      "0x742d35Cc6634C0532925a3b8D4C9db96590c6C87",
		Recipient:   "bh1234567890123456789012345678901234567890",
		Signature:   "dGVzdF9zaWduYXR1cmVfZm9yX2RlbW9fcHVycG9zZXNfd2hpY2hfaXNfdmFsaWQ=", // base64 encoded
		Timestamp:   time.Now(),
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request (is bridge running?): %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("HTTP Status Response: %s\n", resp.Status)

	var responseBody map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	if err != nil {
		fmt.Println("Could not parse response JSON (non-JSON response or empty)")
		return nil
	}

	prettyJSON, _ := json.MarshalIndent(responseBody, "", "  ")
	fmt.Printf("Response Payload:\n%s\n", string(prettyJSON))

	return nil
}

func demoGRPC() {
	address := "localhost:9090"
	fmt.Printf("Dialing gRPC server at %s...\n", address)

	// Set up connection with a small timeout so it doesn't block indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		fmt.Printf("gRPC Dial failed (normal if server is offline or stubs aren't compiled): %v\n", err)
		return
	}
	defer conn.Close()

	fmt.Println("Successfully connected to gRPC port!")
	fmt.Println("To call methods, compile the protobuf schemas first using:")
	fmt.Println("  protoc --go_out=. --go-grpc_out=. api-schema.proto")
	fmt.Println("Then instantiate the client:")
	fmt.Println("  client := pb.NewBridgeServiceClient(conn)")
	fmt.Println("  resp, err := client.RelayToChain(ctx, &pb.RelayToChainRequest{...})")
}
