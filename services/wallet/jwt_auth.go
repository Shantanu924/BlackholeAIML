package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	wallet "github.com/Shivam-Patel-G/blackhole-blockchain/services/wallet/wallet"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims represents the claims inside the BHIV SSO token
type JWTClaims struct {
	UserID      string   `json:"user_id"`
	Email       string   `json:"email"`
	TenantID    string   `json:"tenant_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	AllowedApps []string `json:"allowedApps"`
	jwt.RegisteredClaims
}

var praveshHTTPClient = &http.Client{
	Timeout: 5 * time.Second,
}

// verifyRemotePravesh validates the session token against the deployed Pravesh AUTH service
func verifyRemotePravesh(tokenString string) (*JWTClaims, error) {
	apiBase := os.Getenv("AUTH_API_URL")
	if apiBase == "" {
		apiBase = "https://pravesh.blackholeinfiverse.com/api"
	}
	apiBase = strings.TrimRight(apiBase, "/")
	meURL := apiBase + "/me"

	req, err := http.NewRequest("GET", meURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create pravesh auth request: %w", err)
	}

	// Pravesh verifies via blackhole_token cookie
	req.Header.Set("Cookie", fmt.Sprintf("blackhole_token=%s", tokenString))
	req.Header.Set("Accept", "application/json")

	resp, err := praveshHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pravesh service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pravesh returned status %d: %s", resp.StatusCode, string(body))
	}

	var meResp struct {
		User struct {
			UserID      string   `json:"user_id"`
			Email       string   `json:"email"`
			Roles       []string `json:"roles"`
			AllowedApps []string `json:"allowedApps"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&meResp); err != nil {
		return nil, fmt.Errorf("failed to decode pravesh /api/me response: %w", err)
	}

	if meResp.User.Email == "" && meResp.User.UserID == "" {
		return nil, fmt.Errorf("invalid user payload from pravesh")
	}

	claims := &JWTClaims{
		UserID:      meResp.User.UserID,
		Email:       meResp.User.Email,
		Roles:       meResp.User.Roles,
		AllowedApps: meResp.User.AllowedApps,
	}

	return claims, nil
}

// requireAuth middleware to check BHIV SSO authentication
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Local dev testing bypass
		if os.Getenv("DEV_MODE") == "true" {
			ctx := context.WithValue(r.Context(), "userID", "dev_user")
			ctx = context.WithValue(ctx, "username", "dev_user")
			next(w, r.WithContext(ctx))
			return
		}

		// 1. Extract token from cookie (check blackhole_token first, then bhiv_token)
		var tokenString string
		if cookie, err := r.Cookie("blackhole_token"); err == nil && cookie.Value != "" {
			tokenString = cookie.Value
		} else if cookie, err := r.Cookie("bhiv_token"); err == nil && cookie.Value != "" {
			tokenString = cookie.Value
		} else {
			// Check Authorization Bearer header
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			handleUnauthorized(w, r)
			return
		}

		var claims *JWTClaims
		var authErr error

		// 2. Try local JWT parsing if secret is configured
		secret := os.Getenv("JWT_SECRET")
		if secret != "" {
			parsedClaims := &JWTClaims{}
			token, err := jwt.ParseWithClaims(tokenString, parsedClaims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			})
			if err == nil && token.Valid {
				claims = parsedClaims
			} else {
				authErr = err
			}
		}

		// 3. Fallback to remote Pravesh verification if local validation didn't succeed
		if claims == nil {
			remoteClaims, remoteErr := verifyRemotePravesh(tokenString)
			if remoteErr == nil {
				claims = remoteClaims
			} else {
				logError("AUTH_VERIFY", fmt.Errorf("local: %v, remote pravesh: %v", authErr, remoteErr))
				handleUnauthorized(w, r)
				return
			}
		}

		// 4. Check allowedApps for 'vajra'
		appAllowed := false
		for _, app := range claims.AllowedApps {
			if strings.EqualFold(app, "vajra") {
				appAllowed = true
				break
			}
		}

		if !appAllowed {
			logError("AUTH_VERIFY", fmt.Errorf("app 'vajra' not in allowedApps for user %s (apps: %v)", claims.Email, claims.AllowedApps))
			handleUnauthorized(w, r)
			return
		}

		// 5. Synchronize the user to local DB
		err := syncUserToLocalDB(claims)
		if err != nil {
			logError("AUTH_SYNC", fmt.Errorf("failed to sync user: %v", err))
			handleUnauthorized(w, r)
			return
		}

		// 6. Set context so handlers can retrieve both username and claims
		username := claims.Email
		if username == "" {
			username = "user_" + claims.UserID
		}
		ctx := context.WithValue(r.Context(), "user_claims", claims)
		ctx = context.WithValue(ctx, "username", username)
		ctx = context.WithValue(ctx, "userID", claims.UserID)
		next(w, r.WithContext(ctx))
	}
}

// handleUnauthorized manages redirection or API 401s
func handleUnauthorized(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		sendJSONResponse(w, APIResponse{
			Success: false,
			Message: "Authentication required",
		}, http.StatusUnauthorized)
		return
	}
	redirectURL := os.Getenv("AUTH_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "https://pravesh.blackholeinfiverse.com/login"
	}
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}

// syncUserToLocalDB ensures a wallet.User exists for the central SSO user
func syncUserToLocalDB(claims *JWTClaims) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	username := claims.Email
	if username == "" {
		username = "user_" + claims.UserID
	}

	// Fetch or create user in local database
	if wallet.UserCollection == nil && wallet.FallbackDB == nil {
		fmt.Printf("⚠️  [syncUserToLocalDB] Database not available, skipping sync for %s\n", username)
		return nil
	}

	// Try to find the user locally first
	_, err := wallet.GetUserByUsername(ctx, username)
	if err != nil {
		// User doesn't exist, create them
		_, err := wallet.RegisterUser(ctx, username, "sso-managed-"+claims.Subject)
		if err != nil {
			return fmt.Errorf("failed to create local user: %v", err)
		}
	}

	return nil
}

// getUserFromContext retrieves the user from the JWT claims stored in the context
func getUserFromContext(r *http.Request) (*wallet.User, error) {
	username, ok := r.Context().Value("username").(string)
	if !ok || username == "" {
		return nil, fmt.Errorf("user not authenticated")
	}

	user, err := wallet.GetUserByUsername(r.Context(), username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %v", err)
	}

	return user, nil
}

