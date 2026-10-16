package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBridgeDashboard_Identity(t *testing.T) {
	handler := setupMux("")
	req := httptest.NewRequest("GET", "/identity", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var id IdentityCard
	if err := json.Unmarshal(w.Body.Bytes(), &id); err != nil {
		t.Fatalf("failed to decode identity JSON: %v", err)
	}

	if id.Service != "ui-bridge-dashboard" {
		t.Errorf("expected service ui-bridge-dashboard, got %s", id.Service)
	}
	if id.Status != "online" {
		t.Errorf("expected status online, got %s", id.Status)
	}
}

func TestBridgeDashboard_Health(t *testing.T) {
	handler := setupMux("")
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var health HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &health); err != nil {
		t.Fatalf("failed to parse health JSON: %v", err)
	}

	if health.Status != "HEALTHY" {
		t.Errorf("expected status HEALTHY, got %s", health.Status)
	}
	if health.Checks["error_boundary"] != "ARMED" {
		t.Errorf("expected error_boundary ARMED, got %s", health.Checks["error_boundary"])
	}
}

func TestBridgeDashboard_Metrics(t *testing.T) {
	handler := setupMux("")
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var metrics MetricsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &metrics); err != nil {
		t.Fatalf("failed to parse metrics JSON: %v", err)
	}

	if metrics.Service != "ui-bridge-dashboard" {
		t.Errorf("expected service ui-bridge-dashboard, got %s", metrics.Service)
	}
}

func TestBridgeDashboard_Alerts(t *testing.T) {
	handler := setupMux("")
	req := httptest.NewRequest("GET", "/alerts", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var alerts AlertsResponse
	if err := json.Unmarshal(w.Body.Bytes(), &alerts); err != nil {
		t.Fatalf("failed to parse alerts JSON: %v", err)
	}

	if alerts.Service != "ui-bridge-dashboard" {
		t.Errorf("expected service ui-bridge-dashboard, got %s", alerts.Service)
	}
}

func TestBridgeDashboard_CORS(t *testing.T) {
	handler := setupMux("")
	req := httptest.NewRequest("OPTIONS", "/identity", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 for OPTIONS, got %d", w.Code)
	}
	if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin '*', got '%s'", origin)
	}
}

func TestBridgeDashboard_ErrorBoundary_PanicRecovery(t *testing.T) {
	handler := setupMux("")
	req := httptest.NewRequest("GET", "/debug/panic", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500 from error boundary, got %d", w.Code)
	}

	var errEnv ErrorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &errEnv); err != nil {
		t.Fatalf("failed to parse panic error envelope: %v", err)
	}

	if errEnv.Code != "INTERNAL_SERVER_PANIC_RECOVERED" {
		t.Errorf("expected code INTERNAL_SERVER_PANIC_RECOVERED, got %s", errEnv.Code)
	}
}
