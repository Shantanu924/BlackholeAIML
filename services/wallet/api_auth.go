package main

import (
	"context"

	"net/http"
	"time"

	wallet "github.com/Shivam-Patel-G/blackhole-blockchain/services/wallet/wallet"
)



// serveRegister serves the registration page


// Local auth endpoints (login, register, logout) have been removed.
// Authentication is now managed centrally via the BHIV Core Identity system (SSO).
//
// DECENTRALIZED FALLBACK (PHASE 2):
// Because centralized identity creates a single point of failure (e.g. Render 503s),
// we provide a local fallback mechanism that verifies the master wallet credentials
// directly against the local BadgerDB and mints a local emergency session.

import "encoding/json"
import "os"
import "fmt"
import "github.com/golang-jwt/jwt/v5"

// handleFallbackLogin provides a decentralized emergency login
func handleFallbackLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}

	// 1. Authenticate locally bypassing SSO
	user, err := wallet.AuthenticateUser(r.Context(), req.Username, req.Password)
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Invalid local credentials"}, http.StatusUnauthorized)
		return
	}

	// 2. Generate an emergency local JWT
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "fallback-secret" // Ensure it doesn't crash if secret isn't loaded
	}

	claims := &JWTClaims{
		UserID:      user.ID.Hex(),
		Email:       user.Username,
		TenantID:    "local-fallback",
		Roles:       []string{"user"},
		AllowedApps: []string{"vajra"},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Failed to create fallback token"}, http.StatusInternalServerError)
		return
	}

	// 3. Set the cookie as the exact same token so requireAuth accepts it
	http.SetCookie(w, &http.Cookie{
		Name:     "bhiv_token",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // allow localhost
		MaxAge:   86400,
	})

	sendJSONResponse(w, APIResponse{Success: true, Message: "Fallback login successful"}, http.StatusOK)
}


// handleStatus provides API status information
func handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	// Check blockchain connection status
	blockchainConnected := false
	if wallet.DefaultBlockchainClient != nil {
		blockchainConnected = wallet.DefaultBlockchainClient.IsConnected()
	}

	// Check database connection
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var dbStatus string = "connected"
	if wallet.UserCollection != nil {
		err := wallet.UserCollection.Database().Client().Ping(ctx, nil)
		if err != nil {
			dbStatus = "disconnected"
		}
	} else {
		dbStatus = "not_initialized"
	}

	// Check enhanced storage status
	var storageHealth map[string]interface{}
	if storageManager != nil {
		health := storageManager.HealthCheck(ctx)
		storageHealth = map[string]interface{}{
			"postgresql_available": storageManager.IsPostgreSQLAvailable(),
			"redis_available":      storageManager.IsRedisAvailable(),
			"badgerdb_available":   true,
		}

		// Add health check details
		healthDetails := make(map[string]string)
		for system, err := range health {
			if err != nil {
				healthDetails[system] = "error: " + err.Error()
			} else {
				healthDetails[system] = "healthy"
			}
		}
		storageHealth["health_details"] = healthDetails
	} else {
		storageHealth = map[string]interface{}{
			"postgresql_available": false,
			"redis_available":      false,
			"badgerdb_available":   false,
			"status":               "not_initialized",
		}
	}

	// Check external Auth server health
	authServerURL := os.Getenv("AUTH_SERVER_URL")
	if authServerURL == "" {
		authServerURL = "https://bhiv-auth.onrender.com"
	}
	
	authStatus := "unknown"
	authReqCtx, authCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer authCancel()
	
	req, err := http.NewRequestWithContext(authReqCtx, "GET", authServerURL+"/api/health", nil)
	if err == nil {
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			authStatus = "down"
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				authStatus = "healthy"
			} else {
				authStatus = fmt.Sprintf("degraded_HTTP_%d", resp.StatusCode)
			}
		}
	} else {
		authStatus = "error_creating_request"
	}

	status := map[string]interface{}{
		"service":              "blackhole-wallet",
		"version":              "2.0.0",
		"status":               "running",
		"timestamp":            time.Now().Unix(),
		"blockchain_connected": blockchainConnected,
		"database_status":      dbStatus,
		"storage_system":       storageHealth,
		"auth_server_status":   authStatus,
		"active_sessions":      "managed_by_sso",
		"uptime_seconds":       int64(time.Since(startTime).Seconds()),
	}

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "Service status retrieved successfully",
		Data:    status,
	}, http.StatusOK)
}

// getUserFromSession gets user from session (legacy wrapper for context)
func getUserFromSession(r *http.Request) (*wallet.User, error) {
	return getUserFromContext(r)
}

// handleUser provides current user information
func handleUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		sendJSONResponse(w, APIResponse{Success: false, Message: "Method not allowed"}, http.StatusMethodNotAllowed)
		return
	}

	user, err := getUserFromContext(r)
	if err != nil {
		logError("GET_USER", err)
		sendJSONResponse(w, APIResponse{Success: false, Message: "User not found or unauthorized"}, http.StatusUnauthorized)
		return
	}

	userInfo := map[string]interface{}{
		"id":         user.ID.Hex(),
		"username":   user.Username,
		"created_at": user.CreatedAt.Unix(),
	}

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "User information retrieved successfully",
		Data:    userInfo,
	}, http.StatusOK)
}

// handleLogout clears session cookies and returns SSO redirect target
func handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "bhiv_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "blackhole_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	})

	redirectURL := os.Getenv("AUTH_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "https://pravesh.blackholeinfiverse.com/login"
	}

	sendJSONResponse(w, APIResponse{
		Success: true,
		Message: "Logged out successfully",
		Data: map[string]string{
			"redirect": redirectURL,
		},
	}, http.StatusOK)
}

// handleSSORedirect redirects directly to the central SSO service
func handleSSORedirect(w http.ResponseWriter, r *http.Request) {
	redirectURL := os.Getenv("AUTH_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "https://pravesh.blackholeinfiverse.com/login"
	}
	http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
}
