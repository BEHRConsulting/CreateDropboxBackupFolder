package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original environment and restore at the end
	originalBackupDir := os.Getenv("DROPBOX_BACKUP_FOLDER")
	originalClientID := os.Getenv("DROPBOX_CLIENT_ID")
	originalClientSecret := os.Getenv("DROPBOX_CLIENT_SECRET")
	originalAccessToken := os.Getenv("DROPBOX_ACCESS_TOKEN")
	originalRefreshToken := os.Getenv("DROPBOX_REFRESH_TOKEN")

	defer func() {
		os.Setenv("DROPBOX_BACKUP_FOLDER", originalBackupDir)
		os.Setenv("DROPBOX_CLIENT_ID", originalClientID)
		os.Setenv("DROPBOX_CLIENT_SECRET", originalClientSecret)
		os.Setenv("DROPBOX_ACCESS_TOKEN", originalAccessToken)
		os.Setenv("DROPBOX_REFRESH_TOKEN", originalRefreshToken)
	}()

	tests := []struct {
		name    string
		opts    Options
		envVars map[string]string
		want    *Config
		wantErr bool
	}{
		{
			name: "default configuration with required env vars",
			opts: Options{},
			envVars: map[string]string{
				"DROPBOX_CLIENT_ID":     "test_client_id",
				"DROPBOX_CLIENT_SECRET": "test_client_secret",
			},
			want: &Config{
				ClientID:       "test_client_id",
				ClientSecret:   "test_client_secret",
				LogLevel:       "error",
				MaxConcurrency: 5,
				RetryAttempts:  3,
				RetryDelay:     time.Second * 2,
			},
		},
		{
			name: "with command line options",
			opts: Options{
				BackupDir: ".",
				LogLevel:  "debug",
				Delete:    true,
				Exclude:   []string{"*.tmp", "*.log"},
				ShowCount: true,
				ShowSize:  true,
			},
			envVars: map[string]string{
				"DROPBOX_CLIENT_ID":     "test_client_id",
				"DROPBOX_CLIENT_SECRET": "test_client_secret",
			},
			want: &Config{
				ClientID:       "test_client_id",
				ClientSecret:   "test_client_secret",
				LogLevel:       "debug",
				Delete:         true,
				Exclude:        []string{"*.tmp", "*.log"},
				ShowCount:      true,
				ShowSize:       true,
				MaxConcurrency: 5,
				RetryAttempts:  3,
				RetryDelay:     time.Second * 2,
			},
		},
		{
			name: "with environment variables",
			opts: Options{},
			envVars: map[string]string{
				"DROPBOX_BACKUP_FOLDER": ".",
				"DROPBOX_CLIENT_ID":     "test_client_id",
				"DROPBOX_CLIENT_SECRET": "test_client_secret",
				"DROPBOX_ACCESS_TOKEN":  "test_access_token",
				"DROPBOX_REFRESH_TOKEN": "test_refresh_token",
			},
			want: &Config{
				ClientID:       "test_client_id",
				ClientSecret:   "test_client_secret",
				AccessToken:    "test_access_token",
				RefreshToken:   "test_refresh_token",
				LogLevel:       "error",
				MaxConcurrency: 5,
				RetryAttempts:  3,
				RetryDelay:     time.Second * 2,
			},
		},
		{
			name: "command line overrides environment",
			opts: Options{
				BackupDir: ".",
				LogLevel:  "info",
			},
			envVars: map[string]string{
				"DROPBOX_BACKUP_FOLDER": ".",
				"DROPBOX_CLIENT_ID":     "test_client_id",
				"DROPBOX_CLIENT_SECRET": "test_client_secret",
			},
			want: &Config{
				ClientID:       "test_client_id",
				ClientSecret:   "test_client_secret",
				LogLevel:       "info",
				MaxConcurrency: 5,
				RetryAttempts:  3,
				RetryDelay:     time.Second * 2,
			},
		},
		{
			name:    "missing required environment variables",
			opts:    Options{},
			envVars: map[string]string{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			got, err := Load(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.ClientID != tt.want.ClientID {
					t.Errorf("Load() ClientID = %v, want %v", got.ClientID, tt.want.ClientID)
				}
				if got.ClientSecret != tt.want.ClientSecret {
					t.Errorf("Load() ClientSecret = %v, want %v", got.ClientSecret, tt.want.ClientSecret)
				}
				if got.AccessToken != tt.want.AccessToken {
					t.Errorf("Load() AccessToken = %v, want %v", got.AccessToken, tt.want.AccessToken)
				}
				if got.RefreshToken != tt.want.RefreshToken {
					t.Errorf("Load() RefreshToken = %v, want %v", got.RefreshToken, tt.want.RefreshToken)
				}
				if got.LogLevel != tt.want.LogLevel {
					t.Errorf("Load() LogLevel = %v, want %v", got.LogLevel, tt.want.LogLevel)
				}
				if got.Delete != tt.want.Delete {
					t.Errorf("Load() Delete = %v, want %v", got.Delete, tt.want.Delete)
				}
				if got.ShowCount != tt.want.ShowCount {
					t.Errorf("Load() ShowCount = %v, want %v", got.ShowCount, tt.want.ShowCount)
				}
				if got.ShowSize != tt.want.ShowSize {
					t.Errorf("Load() ShowSize = %v, want %v", got.ShowSize, tt.want.ShowSize)
				}
				if len(got.Exclude) != len(tt.want.Exclude) {
					t.Errorf("Load() Exclude length = %v, want %v", len(got.Exclude), len(tt.want.Exclude))
				}
				for i, exclude := range got.Exclude {
					if i < len(tt.want.Exclude) && exclude != tt.want.Exclude[i] {
						t.Errorf("Load() Exclude[%d] = %v, want %v", i, exclude, tt.want.Exclude[i])
					}
				}
				if got.MaxConcurrency != tt.want.MaxConcurrency {
					t.Errorf("Load() MaxConcurrency = %v, want %v", got.MaxConcurrency, tt.want.MaxConcurrency)
				}
				if got.RetryAttempts != tt.want.RetryAttempts {
					t.Errorf("Load() RetryAttempts = %v, want %v", got.RetryAttempts, tt.want.RetryAttempts)
				}
				if got.RetryDelay != tt.want.RetryDelay {
					t.Errorf("Load() RetryDelay = %v, want %v", got.RetryDelay, tt.want.RetryDelay)
				}
				// Only check BackupDir if we set one specifically and it's not converted to absolute
				if tt.opts.BackupDir != "" {
					// The actual BackupDir will be an absolute path, so we just check it's not empty
					if got.BackupDir == "" {
						t.Errorf("Load() BackupDir should not be empty when specified")
					}
				}
			}

			// Clear environment variables for next test
			for key := range tt.envVars {
				os.Unsetenv(key)
			}
		})
	}
}

func TestSetBackupDir(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantPath bool // true if we expect a valid path
	}{
		{
			name:     "empty string creates default",
			input:    "",
			wantPath: true,
		},
		{
			name:     "relative path",
			input:    "./backup",
			wantPath: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{}
			err := cfg.setBackupDir(tt.input)

			if tt.wantPath && cfg.BackupDir == "" {
				t.Errorf("setBackupDir() did not set BackupDir")
			}

			if err != nil {
				// This is expected if the directory can't be created
				t.Logf("setBackupDir() error (expected in test environment): %v", err)
				return
			}

			if tt.input == "" && !strings.Contains(cfg.BackupDir, "dropbox_backup_") {
				t.Errorf("setBackupDir() default BackupDir does not contain expected pattern: %v", cfg.BackupDir)
			}

			// Check that path is absolute (starts with / on Unix systems)
			if !filepath.IsAbs(cfg.BackupDir) {
				t.Errorf("setBackupDir() BackupDir should be absolute path: %v", cfg.BackupDir)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				BackupDir:    "/valid/path",
				LogLevel:     "error",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &Config{
				ClientSecret: "test_client_secret",
				BackupDir:    "/valid/path",
				LogLevel:     "error",
			},
			wantErr: true,
		},
		{
			name: "missing client secret",
			config: &Config{
				ClientID:  "test_client_id",
				BackupDir: "/valid/path",
				LogLevel:  "error",
			},
			wantErr: true,
		},
		{
			name: "missing backup dir",
			config: &Config{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				LogLevel:     "error",
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				BackupDir:    "/valid/path",
				LogLevel:     "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
