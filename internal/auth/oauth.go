// Package auth handles OAuth2 authentication with Todoist.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/hy4ri/todoist-tui/internal/config"
)

const (
	// Todoist OAuth2 endpoints
	authorizationURL = "https://todoist.com/oauth/authorize"
	tokenURL         = "https://todoist.com/oauth/access_token"

	// OAuth2 configuration
	redirectURI = "http://localhost:8585/callback"
	scope       = "data:read_write,data:delete"

	// Server configuration
	callbackPort    = 8585
	callbackTimeout = 5 * time.Minute
)

// TokenResponse represents the OAuth2 token response from Todoist.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// GetAccessToken retrieves a valid access token.
// It checks for existing tokens in the config, and if not found or invalid,
// initiates the OAuth2 flow.
func GetAccessToken(cfg *config.Config) (string, error) {
	// First, check for existing valid token
	if cfg.Auth.AccessToken != "" {
		return cfg.Auth.AccessToken, nil
	}

	// Fall back to API token if available
	if cfg.Auth.APIToken != "" {
		return cfg.Auth.APIToken, nil
	}

	// Check for OAuth credentials from environment
	clientID := os.Getenv("TODOIST_CLIENT_ID")
	clientSecret := os.Getenv("TODOIST_CLIENT_SECRET")

	// If not in env, check config
	if clientID == "" {
		clientID = cfg.Auth.ClientID
	}
	if clientSecret == "" {
		clientSecret = cfg.Auth.ClientSecret
	}

	// If we have OAuth credentials, start the flow
	if clientID != "" && clientSecret != "" {
		token, err := performOAuthFlow(clientID, clientSecret)
		if err != nil {
			return "", fmt.Errorf("OAuth flow failed: %w", err)
		}

		// Save the token to config
		cfg.Auth.AccessToken = token
		if err := config.Save(cfg); err != nil {
			// Non-fatal warning
			fmt.Fprintf(os.Stderr, "Warning: failed to save token to config: %v\n", err)
		}

		return token, nil
	}

	// No authentication method available
	return "", fmt.Errorf(
		"no authentication configured. Please either:\n" +
			"  1. Set TODOIST_CLIENT_ID and TODOIST_CLIENT_SECRET environment variables for OAuth, or\n" +
			"  2. Add 'api_token' to ~/.config/todoist-tui/config.yaml\n" +
			"     (Get your API token from https://app.todoist.com/app/settings/integrations/developer)",
	)
}

// performOAuthFlow initiates the OAuth2 authorization flow.
func performOAuthFlow(clientID, clientSecret string) (string, error) {
	// Create a channel to receive the authorization code
	codeChan := make(chan string, 1)
	errChan := make(chan error, 1)

	// Start the callback server
	server := startCallbackServer(codeChan, errChan)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	// Build the authorization URL
	authURL := buildAuthorizationURL(clientID)

	// Open the browser for authorization
	fmt.Println("Opening browser for Todoist authorization...")
	fmt.Printf("If the browser doesn't open, please visit:\n%s\n\n", authURL)

	if err := openBrowser(authURL); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to open browser: %v\n", err)
	}

	fmt.Println("Waiting for authorization...")

	// Wait for the callback with timeout
	select {
	case code := <-codeChan:
		// Exchange the code for a token
		return exchangeCodeForToken(clientID, clientSecret, code)
	case err := <-errChan:
		return "", err
	case <-time.After(callbackTimeout):
		return "", fmt.Errorf("authorization timed out after %v", callbackTimeout)
	}
}

// startCallbackServer starts an HTTP server to receive the OAuth2 callback.
func startCallbackServer(codeChan chan<- string, errChan chan<- error) *http.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		// Check for error
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			errChan <- fmt.Errorf("authorization denied: %s", errMsg)
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, `<html><body><h1>Authorization Failed</h1><p>%s</p><p>You can close this window.</p></body></html>`, errMsg)
			return
		}

		// Get the authorization code
		code := r.URL.Query().Get("code")
		if code == "" {
			errChan <- fmt.Errorf("no authorization code received")
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprint(w, `<html><body><h1>Authorization Failed</h1><p>No authorization code received.</p><p>You can close this window.</p></body></html>`)
			return
		}

		// Send success response
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Authorization Successful!</h1><p>You can close this window and return to the terminal.</p></body></html>`)

		// Send the code
		codeChan <- code
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", callbackPort),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	return server
}

// buildAuthorizationURL constructs the OAuth2 authorization URL.
func buildAuthorizationURL(clientID string) string {
	params := url.Values{
		"client_id":     {clientID},
		"scope":         {scope},
		"state":         {generateState()},
		"redirect_uri":  {redirectURI},
		"response_type": {"code"},
	}
	return authorizationURL + "?" + params.Encode()
}

// generateState generates a random state string for CSRF protection.
func generateState() string {
	// Simple state generation - in production, use crypto/rand
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// exchangeCodeForToken exchanges the authorization code for an access token.
func exchangeCodeForToken(clientID, clientSecret, code string) (string, error) {
	data := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"code":          {code},
		"redirect_uri":  {redirectURI},
	}

	resp, err := http.Post(tokenURL, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to exchange code for token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("received empty access token")
	}

	return tokenResp.AccessToken, nil
}

// openBrowser opens the default browser to the given URL.
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: // Linux and others
		cmd = exec.Command("xdg-open", url)
	}

	return cmd.Start()
}
