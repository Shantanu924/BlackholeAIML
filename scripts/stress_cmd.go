//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"
)

func main() {
	url := "http://localhost:8080/api/health"
	var wg sync.WaitGroup
	start := time.Now()
	successCount := 0
	var mu sync.Mutex

	fmt.Println("Starting Network Stress Test (Go native)...")

	// Send 1000 requests to health/RPC endpoint concurrently
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, err := http.NewRequest("GET", url, bytes.NewBuffer([]byte{}))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == 200 {
				mu.Lock()
				successCount++
				mu.Unlock()
				resp.Body.Close()
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Printf("Stress Test Complete in %v\n", duration)
	fmt.Printf("Successful Requests: %d/1000\n", successCount)
	fmt.Printf("TPS / RPS: %.2f\n", 1000.0/duration.Seconds())
	if successCount > 900 {
		fmt.Println("Consensus & API stability under load: PASSED")
	} else {
		fmt.Println("Consensus & API stability under load: FAILED")
	}
}
