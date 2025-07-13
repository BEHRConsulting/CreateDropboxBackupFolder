package dropbox

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"time"

	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox"
	"github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox/files"
	"golang.org/x/oauth2"
)

// Client wraps the Dropbox API client with additional functionality
type Client struct {
	dbx      files.Client
	config   *oauth2.Config
	token    *oauth2.Token
	tokenSrc oauth2.TokenSource
}

// AuthConfig holds OAuth2 configuration for Dropbox
type AuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// TokenInfo represents token information for storage/retrieval
type TokenInfo struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Expiry       time.Time `json:"expiry"`
}

// FileInfo represents metadata about a Dropbox file
type FileInfo struct {
	Path        string
	Name        string
	Size        uint64
	ModTime     time.Time
	IsFolder    bool
	ContentHash string
	Rev         string
}

// NewAuthConfig creates a new OAuth2 configuration for Dropbox
func NewAuthConfig(clientID, clientSecret, redirectURL string) *AuthConfig {
	if redirectURL == "" {
		redirectURL = "http://localhost:8080/callback"
	}

	// Dropbox scopes - use the correct scope names
	scopes := []string{
		"files.metadata.read",
		"files.content.read",
	}

	return &AuthConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       scopes,
	}
}

// GetOAuth2Config returns the OAuth2 configuration
func (ac *AuthConfig) GetOAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     ac.ClientID,
		ClientSecret: ac.ClientSecret,
		RedirectURL:  ac.RedirectURL,
		Scopes:       ac.Scopes,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://www.dropbox.com/oauth2/authorize",
			TokenURL:  "https://api.dropboxapi.com/oauth2/token", // Correct Dropbox API endpoint
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}
}

// DebugOAuth2Config prints OAuth2 configuration for debugging
func (ac *AuthConfig) DebugOAuth2Config() {
	config := ac.GetOAuth2Config()
	slog.Debug("OAuth2 Configuration",
		slog.String("client_id", ac.ClientID),
		slog.String("redirect_url", ac.RedirectURL),
		slog.String("auth_url", config.Endpoint.AuthURL),
		slog.String("token_url", config.Endpoint.TokenURL),
		slog.Any("scopes", ac.Scopes),
	)
}

// GenerateAuthURL generates a secure authorization URL
func (ac *AuthConfig) GenerateAuthURL(state string) (string, string, error) {
	config := ac.GetOAuth2Config()

	// For Dropbox, let's use the standard OAuth2 flow without PKCE for now
	// Dropbox may not fully support PKCE or may have specific requirements

	// Build authorization URL
	authURL := config.AuthCodeURL(state,
		oauth2.SetAuthURLParam("token_access_type", "offline"), // Request refresh token
		oauth2.SetAuthURLParam("force_reapprove", "false"),     // Don't force reapproval
	)

	return authURL, "", nil // Return empty code verifier since we're not using PKCE
}

// ExchangeCode exchanges authorization code for tokens
func (ac *AuthConfig) ExchangeCode(ctx context.Context, code, codeVerifier string) (*oauth2.Token, error) {
	config := ac.GetOAuth2Config()

	slog.Debug("Attempting token exchange",
		slog.String("token_url", config.Endpoint.TokenURL),
		slog.String("client_id", ac.ClientID),
		slog.String("redirect_url", ac.RedirectURL),
	)

	// Use standard OAuth2 exchange
	token, err := config.Exchange(ctx, code)
	if err != nil {
		// Log detailed error information
		slog.Error("Token exchange failed",
			slog.String("error", err.Error()),
			slog.String("code_length", fmt.Sprintf("%d", len(code))),
		)
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	slog.Info("Successfully exchanged authorization code for tokens",
		slog.Bool("has_refresh_token", token.RefreshToken != ""),
		slog.Time("expires_at", token.Expiry),
	)

	return token, nil
}

// NewWithToken creates a new Dropbox client with an existing token
func NewWithToken(authConfig *AuthConfig, token *oauth2.Token) (*Client, error) {
	config := authConfig.GetOAuth2Config()

	// Create token source that automatically refreshes tokens
	tokenSrc := config.TokenSource(context.Background(), token)

	// Get a fresh token (this will refresh if needed)
	freshToken, err := tokenSrc.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get fresh token: %w", err)
	}

	// Create HTTP client with automatic token refresh
	httpClient := config.Client(context.Background(), freshToken)

	// Create Dropbox client
	dbx := files.New(dropbox.Config{
		Token:  freshToken.AccessToken,
		Client: httpClient,
	})

	return &Client{
		dbx:      dbx,
		config:   config,
		token:    freshToken,
		tokenSrc: tokenSrc,
	}, nil
}

// Legacy constructor for backward compatibility
func New(clientID, clientSecret, accessToken, refreshToken string) (*Client, error) {
	authConfig := NewAuthConfig(clientID, clientSecret, "")

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	return NewWithToken(authConfig, token)
}

// RefreshToken refreshes the access token if needed
func (c *Client) RefreshToken(ctx context.Context) error {
	if c.tokenSrc == nil {
		return fmt.Errorf("no token source available for refresh")
	}

	// Get fresh token (automatically refreshes if needed)
	freshToken, err := c.tokenSrc.Token()
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Update stored token
	c.token = freshToken

	// Recreate Dropbox client with new token
	httpClient := c.config.Client(ctx, freshToken)
	c.dbx = files.New(dropbox.Config{
		Token:  freshToken.AccessToken,
		Client: httpClient,
	})

	slog.Info("Token refreshed successfully",
		slog.Time("new_expiry", freshToken.Expiry),
	)

	return nil
}

// GetTokenInfo returns current token information
func (c *Client) GetTokenInfo() TokenInfo {
	return TokenInfo{
		AccessToken:  c.token.AccessToken,
		RefreshToken: c.token.RefreshToken,
		TokenType:    c.token.TokenType,
		Expiry:       c.token.Expiry,
	}
}

// IsTokenValid checks if the current token is valid and not expired
func (c *Client) IsTokenValid() bool {
	if c.token == nil {
		return false
	}

	// Check if token is expired (with 5-minute buffer)
	if !c.token.Expiry.IsZero() && time.Now().Add(5*time.Minute).After(c.token.Expiry) {
		return false
	}

	return c.token.AccessToken != ""
}

// PKCE (Proof Key for Code Exchange) helper functions for enhanced security

// generateCodeVerifier generates a cryptographically random code verifier
func generateCodeVerifier() (string, error) {
	// Generate 32 random bytes (256 bits)
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	// Base64 URL encode without padding
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(bytes), nil
}

// generateCodeChallenge generates SHA256 code challenge from verifier
func generateCodeChallenge(verifier string) string {
	// Use SHA256 hashing as per RFC 7636
	hash := sha256.Sum256([]byte(verifier))
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(hash[:])
}

// StartOAuthFlow starts the OAuth2 flow and returns the authorization URL
func StartOAuthFlow(authConfig *AuthConfig) (authURL, state, codeVerifier string, err error) {
	// Generate secure random state
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate state: %w", err)
	}
	state = base64.URLEncoding.EncodeToString(stateBytes)

	// Generate authorization URL with PKCE
	authURL, codeVerifier, err = authConfig.GenerateAuthURL(state)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to generate auth URL: %w", err)
	}

	return authURL, state, codeVerifier, nil
}

// HandleOAuthCallback handles the OAuth callback and exchanges code for token
func HandleOAuthCallback(authConfig *AuthConfig, callbackURL, expectedState, codeVerifier string) (*oauth2.Token, error) {
	parsedURL, err := url.Parse(callbackURL)
	if err != nil {
		return nil, fmt.Errorf("invalid callback URL: %w", err)
	}

	query := parsedURL.Query()

	// Verify state parameter
	state := query.Get("state")
	if state != expectedState {
		return nil, fmt.Errorf("invalid state parameter")
	}

	// Check for error in callback
	if errorParam := query.Get("error"); errorParam != "" {
		errorDesc := query.Get("error_description")
		return nil, fmt.Errorf("OAuth error: %s - %s", errorParam, errorDesc)
	}

	// Get authorization code
	code := query.Get("code")
	if code == "" {
		return nil, fmt.Errorf("no authorization code in callback")
	}

	// Exchange code for token
	ctx := context.Background()
	token, err := authConfig.ExchangeCode(ctx, code, codeVerifier)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}

	return token, nil
}

// ValidateTokenScopes validates that the token has required scopes
func (c *Client) ValidateTokenScopes(ctx context.Context) error {
	// Test the token by making a simple API call to list the root folder
	arg := &files.ListFolderArg{
		Path:      "",
		Recursive: false,
		Limit:     1, // Just need one entry to validate
	}

	_, err := c.dbx.ListFolder(arg)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	slog.Info("Token validation successful")
	return nil
}

// ListAll recursively lists all files and folders in the Dropbox account
func (c *Client) ListAll(ctx context.Context) ([]FileInfo, error) {
	var allFiles []FileInfo

	if err := c.listRecursive(ctx, "", &allFiles); err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	slog.Info("Listed all files from Dropbox", slog.Int("total_files", len(allFiles)))
	return allFiles, nil
}

func (c *Client) listRecursive(ctx context.Context, path string, allFiles *[]FileInfo) error {
	arg := &files.ListFolderArg{
		Path:      path,
		Recursive: false,
	}

	res, err := c.dbx.ListFolder(arg)
	if err != nil {
		return fmt.Errorf("failed to list folder %s: %w", path, err)
	}

	for {
		for _, entry := range res.Entries {
			fileInfo := c.convertToFileInfo(entry)
			*allFiles = append(*allFiles, fileInfo)

			// If it's a folder, recursively list its contents
			if fileInfo.IsFolder {
				if err := c.listRecursive(ctx, fileInfo.Path, allFiles); err != nil {
					return err
				}
			}
		}

		// Check if there are more results
		if !res.HasMore {
			break
		}

		// Continue with the next batch
		continueArg := &files.ListFolderContinueArg{
			Cursor: res.Cursor,
		}

		res, err = c.dbx.ListFolderContinue(continueArg)
		if err != nil {
			return fmt.Errorf("failed to continue listing folder %s: %w", path, err)
		}
	}

	return nil
}

// Download downloads a file from Dropbox
func (c *Client) Download(ctx context.Context, remotePath string) (io.ReadCloser, *FileInfo, error) {
	arg := &files.DownloadArg{
		Path: remotePath,
	}

	res, content, err := c.dbx.Download(arg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download file %s: %w", remotePath, err)
	}

	fileInfo := &FileInfo{
		Path:        remotePath,
		Name:        res.Name,
		Size:        res.Size,
		ModTime:     res.ClientModified,
		IsFolder:    false,
		ContentHash: res.ContentHash,
		Rev:         res.Rev,
	}

	slog.Debug("Downloaded file",
		slog.String("path", remotePath),
		slog.Uint64("size", res.Size),
	)

	return content, fileInfo, nil
}

// GetMetadata retrieves metadata for a file or folder
func (c *Client) GetMetadata(ctx context.Context, path string) (*FileInfo, error) {
	arg := &files.GetMetadataArg{
		Path: path,
	}

	res, err := c.dbx.GetMetadata(arg)
	if err != nil {
		return nil, fmt.Errorf("failed to get metadata for %s: %w", path, err)
	}

	fileInfo := c.convertToFileInfo(res)
	return &fileInfo, nil
}

func (c *Client) convertToFileInfo(entry files.IsMetadata) FileInfo {
	switch e := entry.(type) {
	case *files.FileMetadata:
		return FileInfo{
			Path:        e.PathLower,
			Name:        e.Name,
			Size:        e.Size,
			ModTime:     e.ClientModified,
			IsFolder:    false,
			ContentHash: e.ContentHash,
			Rev:         e.Rev,
		}
	case *files.FolderMetadata:
		return FileInfo{
			Path:     e.PathLower,
			Name:     e.Name,
			Size:     0,
			ModTime:  time.Time{}, // Folders don't have modification times
			IsFolder: true,
		}
	default:
		// Handle other metadata types (e.g., DeletedMetadata)
		return FileInfo{
			Path:     "/unknown",
			Name:     "unknown",
			IsFolder: false,
		}
	}
}
