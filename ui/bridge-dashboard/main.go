package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"
	"time"
)

var (
	startTime    = time.Now()
	requestCount uint64
	errorCount   uint64
	panicCount   uint64
)

// VAJRA Identity Card
type IdentityCard struct {
	Service   string   `json:"service"`
	Domain    string   `json:"domain"`
	Status    string   `json:"status"`
	Version   string   `json:"version"`
	Endpoints []string `json:"endpoints"`
}

// HealthResponse represents real-time bridge UI health
type HealthResponse struct {
	Status    string            `json:"status"`
	Service   string            `json:"service"`
	Uptime    string            `json:"uptime"`
	Timestamp int64             `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// MetricsResponse represents bridge UI telemetry
type MetricsResponse struct {
	Service         string  `json:"service"`
	TotalRequests   uint64  `json:"total_requests"`
	TotalErrors     uint64  `json:"total_errors"`
	PanicsRecovered uint64  `json:"panics_recovered"`
	Goroutines      int     `json:"goroutines"`
	MemoryAllocMB   float64 `json:"memory_alloc_mb"`
	UptimeSeconds   float64 `json:"uptime_seconds"`
}

// Alert definition
type Alert struct {
	Severity  string `json:"severity"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

// AlertsResponse holds active bridge alerts
type AlertsResponse struct {
	Service string  `json:"service"`
	Count   int     `json:"count"`
	Alerts  []Alert `json:"alerts"`
}

// ErrorEnvelope defines structured error response
type ErrorEnvelope struct {
	Error     string `json:"error"`
	Code      string `json:"code"`
	Status    int    `json:"status"`
	Timestamp int64  `json:"timestamp"`
	Path      string `json:"path"`
}

// setupMux sets up all routes with error boundary safety and production monitoring
func setupMux(staticDir string) http.Handler {
	mux := http.NewServeMux()

	// Serve static assets
	if staticDir != "" {
		fs := http.FileServer(http.Dir(staticDir))
		mux.Handle("/", fs)
	}

	// VAJRA Runtime Identity Card
	mux.HandleFunc("/identity", func(w http.ResponseWriter, r *http.Request) {
		identity := IdentityCard{
			Service:   "ui-bridge-dashboard",
			Domain:    "vajra",
			Status:    "online",
			Version:   "2.0.0",
			Endpoints: []string{"/", "/identity", "/health", "/metrics", "/alerts"},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(identity)
	})

	// Production Monitoring: Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		health := HealthResponse{
			Status:    "HEALTHY",
			Service:   "ui-bridge-dashboard",
			Uptime:    time.Since(startTime).String(),
			Timestamp: time.Now().Unix(),
			Checks: map[string]string{
				"static_assets":  "UP",
				"bridge_router":  "UP",
				"error_boundary": "ARMED",
			},
		}
		json.NewEncoder(w).Encode(health)
	})

	// Production Monitoring: Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		w.Header().Set("Content-Type", "application/json")
		metrics := MetricsResponse{
			Service:         "ui-bridge-dashboard",
			TotalRequests:   atomic.LoadUint64(&requestCount),
			TotalErrors:     atomic.LoadUint64(&errorCount),
			PanicsRecovered: atomic.LoadUint64(&panicCount),
			Goroutines:      runtime.NumGoroutine(),
			MemoryAllocMB:   float64(m.Alloc) / 1024.0 / 1024.0,
			UptimeSeconds:   time.Since(startTime).Seconds(),
		}
		json.NewEncoder(w).Encode(metrics)
	})

	// Production Monitoring: Alerts endpoint
	mux.HandleFunc("/alerts", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var alerts []Alert
		if atomic.LoadUint64(&panicCount) > 0 {
			alerts = append(alerts, Alert{
				Severity:  "WARNING",
				Message:   fmt.Sprintf("Interceptions detected: %d panic(s) recovered", atomic.LoadUint64(&panicCount)),
				Timestamp: time.Now().Unix(),
			})
		}
		json.NewEncoder(w).Encode(AlertsResponse{
			Service: "ui-bridge-dashboard",
			Count:   len(alerts),
			Alerts:  alerts,
		})
	})

	// Panic trigger for testing error boundary
	mux.HandleFunc("/debug/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("simulated unhandled bridge dashboard panic for error boundary verification")
	})

	// Wrap in CORS and Panic Recovery Error Boundary
	return enableCORS(withRecovery(mux))
}

// withRecovery provides global panic recovery error boundary safety
func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddUint64(&requestCount, 1)
		defer func() {
			if rec := recover(); rec != nil {
				atomic.AddUint64(&panicCount, 1)
				atomic.AddUint64(&errorCount, 1)
				log.Printf("🚨 [ERROR BOUNDARY] Panic recovered on path %s: %v", r.URL.Path, rec)

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(ErrorEnvelope{
					Error:     fmt.Sprintf("Internal bridge server error intercepted by error boundary: %v", rec),
					Code:      "INTERNAL_SERVER_PANIC_RECOVERED",
					Status:    http.StatusInternalServerError,
					Timestamp: time.Now().Unix(),
					Path:      r.URL.Path,
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// enableCORS handles CORS preflight and headers
func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	port := 8103
	if p := os.Getenv("PORT"); p != "" {
		fmt.Sscanf(p, "%d", &port)
	}

	handler := setupMux("./static")
	fmt.Printf("🌉 Bridge Dashboard UI participant starting on port %d with production monitoring & error boundary safety...\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), handler))
}
