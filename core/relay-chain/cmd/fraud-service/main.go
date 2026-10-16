package main

import (
	"fmt"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/fraud"
)

func main() {
	fmt.Println("🛡️  Starting Fraud Detection Service on :9090")
	fs := fraud.NewFraudService()
	fs.Start()
}
