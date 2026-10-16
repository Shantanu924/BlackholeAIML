package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  bridgectl stats    - View bridge statistics")
		fmt.Println("  bridgectl tail     - Live tail of bridge events")
		os.Exit(1)
	}

	command := os.Args[1]

	// Determine port (default to 8084, fallback to 8080)
	port := "8084"
	if envPort := os.Getenv("PORT"); envPort != "" {
		port = envPort
	}

	switch command {
	case "stats":
		runStats(port)
	case "tail":
		runTail(fmt.Sprintf("http://localhost:%s/log/events", port), true)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: stats, tail")
		os.Exit(1)
	}
}

func runStats(port string) {
	url := fmt.Sprintf("http://localhost:%s/stats", port)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		// Fallback to /bridge/status
		url = fmt.Sprintf("http://localhost:%s/bridge/status", port)
		resp, err = client.Get(url)
	}
	
	if err != nil {
		log.Fatalf("Failed to get stats: %v", err)
	}
	defer resp.Body.Close()

	var stats map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&stats)

	fmt.Printf("Bridge Stats (from %s):\n", url)
	for k, v := range stats {
		fmt.Printf("  %s: %v\n", k, v)
	}
}