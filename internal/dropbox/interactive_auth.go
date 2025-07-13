package dropbox

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"golang.org/x/oauth2"
)

// InteractiveAuth handles the interactive OAuth2 flow
type InteractiveAuth struct {
	authConfig   *AuthConfig
	server       *http.Server
	resultChan   chan AuthResult
	codeVerifier string
	state        string
}

// AuthResult represents the result of an authentication flow
type AuthResult struct {
	Token *oauth2.Token
	Error error
}

// NewInteractiveAuth creates a new interactive authentication handler
func NewInteractiveAuth(clientID, clientSecret string) *InteractiveAuth {
	authConfig := NewAuthConfig(clientID, clientSecret, "http://localhost:8080/callback")

	return &InteractiveAuth{
		authConfig: authConfig,
		resultChan: make(chan AuthResult, 1),
	}
}

// Authenticate starts the interactive OAuth2 flow
func (ia *InteractiveAuth) Authenticate(ctx context.Context) (*oauth2.Token, error) {
	// Debug OAuth2 configuration
	ia.authConfig.DebugOAuth2Config()

	// Start local server for callback
	if err := ia.startCallbackServer(); err != nil {
		return nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	defer ia.stopCallbackServer()

	// Generate authorization URL and store verifier/state
	authURL, state, codeVerifier, err := StartOAuthFlow(ia.authConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to start OAuth flow: %w", err)
	}

	// Store for use in callback
	ia.state = state
	ia.codeVerifier = codeVerifier

	slog.Debug("OAuth2 flow started",
		slog.String("auth_url", authURL),
		slog.String("state", state),
	)

	// Open browser
	fmt.Printf("Opening browser for Dropbox authorization...\n")
	fmt.Printf("If the browser doesn't open automatically, visit: %s\n", authURL)

	if err := openBrowser(authURL); err != nil {
		slog.Warn("Failed to open browser automatically", slog.String("error", err.Error()))
	}

	// Wait for callback or timeout
	select {
	case result := <-ia.resultChan:
		if result.Error != nil {
			return nil, result.Error
		}
		return result.Token, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("authentication timeout or cancelled")
	case <-time.After(5 * time.Minute):
		return nil, fmt.Errorf("authentication timeout after 5 minutes")
	}
}

// startCallbackServer starts the local HTTP server for OAuth callback
func (ia *InteractiveAuth) startCallbackServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", ia.handleCallback)
	mux.HandleFunc("/", ia.handleRoot)

	ia.server = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	go func() {
		if err := ia.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			ia.resultChan <- AuthResult{Error: fmt.Errorf("callback server error: %w", err)}
		}
	}()

	// Give the server a moment to start
	time.Sleep(100 * time.Millisecond)

	return nil
}

// stopCallbackServer stops the local HTTP server
func (ia *InteractiveAuth) stopCallbackServer() {
	if ia.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ia.server.Shutdown(ctx)
	}
}

// handleCallback handles the OAuth2 callback
func (ia *InteractiveAuth) handleCallback(w http.ResponseWriter, r *http.Request) { // Extract parameters from URL
	query := r.URL.Query()
	code := query.Get("code")
	state := query.Get("state")
	errorParam := query.Get("error")

	if errorParam != "" {
		errorDesc := query.Get("error_description")
		err := fmt.Errorf("OAuth error: %s - %s", errorParam, errorDesc)
		ia.resultChan <- AuthResult{Error: err}

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
	<h1>Authentication Failed</h1>
	<p>Error: %s</p>
	<p>Description: %s</p>
	<p>You can close this window and try again.</p>
</body>
</html>`, errorParam, errorDesc)
		return
	}

	// Verify state parameter for CSRF protection
	if state != ia.state {
		err := fmt.Errorf("invalid state parameter")
		ia.resultChan <- AuthResult{Error: err}

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
	<h1>Authentication Failed</h1>
	<p>Invalid state parameter. Possible CSRF attack.</p>
	<p>You can close this window and try again.</p>
</body>
</html>`)
		return
	}

	if code == "" {
		err := fmt.Errorf("no authorization code received")
		ia.resultChan <- AuthResult{Error: err}

		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
	<h1>Authentication Failed</h1>
	<p>No authorization code received.</p>
	<p>You can close this window and try again.</p>
</body>
</html>`)
		return
	}

	// Exchange code for token with proper PKCE verifier
	ctx := context.Background()

	slog.Debug("Exchanging authorization code",
		slog.String("code", code[:10]+"..."), // Log partial code for security
		slog.String("state", state),
	)

	token, err := ia.authConfig.ExchangeCode(ctx, code, ia.codeVerifier)
	if err != nil {
		slog.Error("Failed to exchange authorization code", slog.String("error", err.Error()))
		ia.resultChan <- AuthResult{Error: fmt.Errorf("failed to exchange code: %w", err)}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body>
	<h1>Authentication Failed</h1>
	<p>Failed to exchange authorization code for token.</p>
	<p>Error: %s</p>
	<p>You can close this window and try again.</p>
</body>
</html>`, err.Error())
		return
	}

	// Success
	ia.resultChan <- AuthResult{Token: token}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body>
	<h1>Authentication Successful!</h1>
	<p>You have successfully authenticated with Dropbox.</p>
	<p>You can now close this window and return to the application.</p>
	<script>
		setTimeout(function() {
			window.close();
		}, 3000);
	</script>
</body>
</html>`)
}

// handleRoot handles requests to the root path
func (ia *InteractiveAuth) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Dropbox Backup Tool</title></head>
<body>
	<h1>Dropbox Backup Tool</h1>
	<p>Waiting for OAuth2 callback...</p>
	<p>If you haven't been redirected to Dropbox for authentication, please check the console for the authorization URL.</p>
</body>
</html>`)
}

// openBrowser opens the default browser with the given URL
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// AuthenticateWithStoredToken attempts to use a stored token, falling back to interactive auth
func AuthenticateWithStoredToken(clientID, clientSecret, accessToken, refreshToken string) (*oauth2.Token, error) {
	// If we have tokens, try to use them
	if accessToken != "" {
		token := &oauth2.Token{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
		}

		// Test the token
		authConfig := NewAuthConfig(clientID, clientSecret, "")
		client, err := NewWithToken(authConfig, token)
		if err != nil {
			return nil, fmt.Errorf("failed to create client with stored token: %w", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := client.ValidateTokenScopes(ctx); err == nil {
			slog.Info("Using stored access token")
			return token, nil
		}

		slog.Warn("Stored token is invalid, starting interactive authentication")
	}

	// Fall back to interactive authentication
	interactiveAuth := NewInteractiveAuth(clientID, clientSecret)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	return interactiveAuth.Authenticate(ctx)
}
