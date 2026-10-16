package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// loginToPravesh logs into the live Pravesh deployment and retrieves the blackhole_token cookie
func loginToPravesh(t *testing.T) string {
	apiBase := os.Getenv("AUTH_API_URL")
	if apiBase == "" {
		apiBase = "https://pravesh.blackholeinfiverse.com/api"
	}
	loginURL := apiBase + "/login"

	loginPayload, _ := json.Marshal(map[string]string{
		"email":    "admin@bhiv.com",
		"password": "password123",
	})

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(loginURL, "application/json", bytes.NewReader(loginPayload))
	if err != nil {
		t.Skipf("Pravesh remote service unreachable, skipping live test: %v", err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Pravesh login failed with status %d", resp.StatusCode)
	}

	var token string
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "blackhole_token" {
			token = cookie.Value
			break
		}
	}

	if token == "" {
		t.Fatalf("Expected blackhole_token cookie in Pravesh login response, got none")
	}

	return token
}

func TestPraveshRemoteVerification(t *testing.T) {
	token := loginToPravesh(t)
	if token == "" {
		return
	}

	claims, err := verifyRemotePravesh(token)
	if err != nil {
		t.Fatalf("verifyRemotePravesh failed: %v", err)
	}

	if claims.Email != "admin@bhiv.com" {
		t.Errorf("Expected email 'admin@bhiv.com', got '%s'", claims.Email)
	}

	appFound := false
	for _, app := range claims.AllowedApps {
		if app == "vajra" {
			appFound = true
			break
		}
	}

	if !appFound {
		t.Errorf("Expected 'vajra' in allowedApps, got: %v", claims.AllowedApps)
	}
}

func TestRequireAuthMiddlewareWithPraveshToken(t *testing.T) {
	token := loginToPravesh(t)
	if token == "" {
		return
	}

	// Create test handler wrapped in requireAuth
	called := false
	var capturedUsername string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if u, ok := r.Context().Value("username").(string); ok {
			capturedUsername = u
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"authorized"}`))
	})

	protected := requireAuth(testHandler)

	// 1. Request with valid blackhole_token cookie
	req := httptest.NewRequest("GET", "/api/protected/resource", nil)
	req.AddCookie(&http.Cookie{
		Name:  "blackhole_token",
		Value: token,
	})
	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got status %d, body: %s", rec.Code, rec.Body.String())
	}

	if !called {
		t.Errorf("Expected handler to be called")
	}

	if capturedUsername != "admin@bhiv.com" {
		t.Errorf("Expected context username 'admin@bhiv.com', got '%s'", capturedUsername)
	}

	// 2. Request without token should be rejected (401)
	reqNoAuth := httptest.NewRequest("GET", "/api/protected/resource", nil)
	recNoAuth := httptest.NewRecorder()

	protected.ServeHTTP(recNoAuth, reqNoAuth)

	if recNoAuth.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for unauthenticated request, got %d", recNoAuth.Code)
	}
}
