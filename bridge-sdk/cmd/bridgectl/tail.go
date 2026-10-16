package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Event represents a parsed event from the bridge log
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Chain     string                 `json:"chain"`
	TxHash    string                 `json:"tx_hash"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

func runTail(url string, follow bool) {
	fmt.Printf("📋 Tailing bridge events from: %s\n", url)
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println("--------------------------------------------------------------------------------")

	seenEvents := make(map[string]bool)

	for {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error contacting bridge API: %v\n", err)
			if !follow {
				os.Exit(1)
			}
			time.Sleep(2 * time.Second)
			continue
		}

		var events []Event
		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if readErr == nil {
			// Try 1: Direct array decode
			if err := json.Unmarshal(bodyBytes, &events); err != nil {
				// Try 2: Decode using wrapper structs to support multiple formats
				var wrapper struct {
					Data struct {
						Events []Event `json:"events"`
					} `json:"data"`
					Events []Event `json:"events"`
				}
				if err2 := json.Unmarshal(bodyBytes, &wrapper); err2 == nil {
					if len(wrapper.Data.Events) > 0 {
						events = wrapper.Data.Events
					} else {
						events = wrapper.Events
					}
				}
			}
		}

		// Print new events in chronological order
		for i := len(events) - 1; i >= 0; i-- {
			e := events[i]
			if !seenEvents[e.ID] {
				seenEvents[e.ID] = true
				fmt.Printf("[%s] %s | Chain: %s | Tx: %s\n",
					e.Timestamp.Format("15:04:05"),
					e.Type,
					e.Chain,
					e.TxHash,
				)
				if len(e.Data) > 0 {
					dataBytes, _ := json.Marshal(e.Data)
					fmt.Printf("   Data: %s\n", string(dataBytes))
				}
			}
		}

		if !follow {
			break
		}

		time.Sleep(2 * time.Second)
	}
}
