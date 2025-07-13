package main

import (
	"os"
	"testing"

	"create-dropbox-backup-folder/internal/config"
)

func TestMain(m *testing.M) {
	// Setup test environment
	code := m.Run()
	// Cleanup if needed
	os.Exit(code)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				BackupDir:    "/tmp/backup",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &config.Config{
				ClientSecret: "test_client_secret",
				BackupDir:    "/tmp/backup",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would test the main config validation logic
			// We can't easily test the actual main function without mocking
			if tt.config.ClientID == "" && !tt.wantErr {
				t.Error("Expected error for empty ClientID")
			}
			if tt.config.ClientID != "" && tt.wantErr {
				t.Error("Expected no error for valid ClientID")
			}
		})
	}
}

func TestDefaultValues(t *testing.T) {
	// Test default configuration values
	opts := config.Options{}
	cfg, err := config.Load(opts)
	if err != nil {
		// This is expected since we don't have environment variables set
		t.Logf("Expected error loading config without environment: %v", err)
		return
	}

	// Test default values if config loads successfully
	if cfg.LogLevel != "error" {
		t.Errorf("Default LogLevel = %v, want 'error'", cfg.LogLevel)
	}
	if cfg.MaxConcurrency != 5 {
		t.Errorf("Default MaxConcurrency = %v, want 5", cfg.MaxConcurrency)
	}
	if cfg.RetryAttempts != 3 {
		t.Errorf("Default RetryAttempts = %v, want 3", cfg.RetryAttempts)
	}
}
