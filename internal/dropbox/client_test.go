package dropbox

import (
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestNewAuthConfig(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		redirectURL  string
		wantURL      string
	}{
		{
			name:         "default redirect URL",
			clientID:     "test_client_id",
			clientSecret: "test_client_secret",
			redirectURL:  "",
			wantURL:      "http://localhost:8080/callback",
		},
		{
			name:         "custom redirect URL",
			clientID:     "test_client_id",
			clientSecret: "test_client_secret",
			redirectURL:  "https://example.com/auth",
			wantURL:      "https://example.com/auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := NewAuthConfig(tt.clientID, tt.clientSecret, tt.redirectURL)

			if config.ClientID != tt.clientID {
				t.Errorf("NewAuthConfig() ClientID = %v, want %v", config.ClientID, tt.clientID)
			}
			if config.ClientSecret != tt.clientSecret {
				t.Errorf("NewAuthConfig() ClientSecret = %v, want %v", config.ClientSecret, tt.clientSecret)
			}
			if config.RedirectURL != tt.wantURL {
				t.Errorf("NewAuthConfig() RedirectURL = %v, want %v", config.RedirectURL, tt.wantURL)
			}

			// Check that required scopes are set
			expectedScopes := []string{"files.metadata.read", "files.content.read"}
			if len(config.Scopes) != len(expectedScopes) {
				t.Errorf("NewAuthConfig() Scopes length = %v, want %v", len(config.Scopes), len(expectedScopes))
			}
			for i, scope := range config.Scopes {
				if i < len(expectedScopes) && scope != expectedScopes[i] {
					t.Errorf("NewAuthConfig() Scopes[%d] = %v, want %v", i, scope, expectedScopes[i])
				}
			}
		})
	}
}

func TestGenerateAuthURL(t *testing.T) {
	config := NewAuthConfig("test_client", "test_secret", "")

	state := "test_state"

	url, codeChallenge, err := config.GenerateAuthURL(state)
	if err != nil {
		t.Errorf("GenerateAuthURL() error = %v", err)
		return
	}

	if url == "" {
		t.Error("GenerateAuthURL() returned empty URL")
	}

	// Code challenge should be empty for this implementation
	if codeChallenge != "" {
		t.Errorf("GenerateAuthURL() returned non-empty code challenge: %s", codeChallenge)
	}

	// Check that URL contains expected parameters
	expectedParams := []string{
		"client_id=test_client",
		"response_type=code",
		"state=" + state,
	}

	for _, param := range expectedParams {
		if !contains(url, param) {
			t.Errorf("GenerateAuthURL() URL does not contain expected parameter: %s", param)
		}
	}
}

func TestTokenInfo(t *testing.T) {
	tests := []struct {
		name      string
		tokenInfo TokenInfo
		wantValid bool
	}{
		{
			name: "valid token",
			tokenInfo: TokenInfo{
				AccessToken:  "valid_token",
				RefreshToken: "refresh_token",
				TokenType:    "Bearer",
				Expiry:       time.Now().Add(time.Hour),
			},
			wantValid: true,
		},
		{
			name: "expired token",
			tokenInfo: TokenInfo{
				AccessToken:  "expired_token",
				RefreshToken: "refresh_token",
				TokenType:    "Bearer",
				Expiry:       time.Now().Add(-time.Hour),
			},
			wantValid: false,
		},
		{
			name: "token without expiry",
			tokenInfo: TokenInfo{
				AccessToken:  "no_expiry_token",
				RefreshToken: "refresh_token",
				TokenType:    "Bearer",
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &oauth2.Token{
				AccessToken:  tt.tokenInfo.AccessToken,
				RefreshToken: tt.tokenInfo.RefreshToken,
				TokenType:    tt.tokenInfo.TokenType,
				Expiry:       tt.tokenInfo.Expiry,
			}

			valid := token.Valid()
			if valid != tt.wantValid {
				t.Errorf("Token.Valid() = %v, want %v", valid, tt.wantValid)
			}
		})
	}
}

func TestFileInfo(t *testing.T) {
	fileInfo := FileInfo{
		Path:        "/test/file.txt",
		Name:        "file.txt",
		Size:        1024,
		ModTime:     time.Now(),
		IsFolder:    false,
		ContentHash: "abc123",
		Rev:         "rev123",
	}

	if fileInfo.Path != "/test/file.txt" {
		t.Errorf("FileInfo.Path = %v, want %v", fileInfo.Path, "/test/file.txt")
	}
	if fileInfo.Name != "file.txt" {
		t.Errorf("FileInfo.Name = %v, want %v", fileInfo.Name, "file.txt")
	}
	if fileInfo.Size != 1024 {
		t.Errorf("FileInfo.Size = %v, want %v", fileInfo.Size, 1024)
	}
	if fileInfo.IsFolder != false {
		t.Errorf("FileInfo.IsFolder = %v, want %v", fileInfo.IsFolder, false)
	}
	if fileInfo.ContentHash != "abc123" {
		t.Errorf("FileInfo.ContentHash = %v, want %v", fileInfo.ContentHash, "abc123")
	}
	if fileInfo.Rev != "rev123" {
		t.Errorf("FileInfo.Rev = %v, want %v", fileInfo.Rev, "rev123")
	}
}

func TestClientCreation(t *testing.T) {
	tests := []struct {
		name         string
		clientID     string
		clientSecret string
		accessToken  string
		refreshToken string
		wantErr      bool
	}{
		{
			name:         "valid parameters",
			clientID:     "test_client_id",
			clientSecret: "test_client_secret",
			accessToken:  "test_access_token",
			refreshToken: "test_refresh_token",
			wantErr:      false, // Actually succeeds in creating the client object
		},
		{
			name:         "empty parameters",
			clientID:     "",
			clientSecret: "",
			accessToken:  "",
			refreshToken: "",
			wantErr:      true, // Fails due to empty tokens
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.clientID, tt.clientSecret, tt.accessToken, tt.refreshToken)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsAt(s, substr))))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
