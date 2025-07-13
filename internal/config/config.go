package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Config holds the application configuration
type Config struct {
	// Dropbox OAuth2 settings
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	// Backup settings
	BackupDir string   `json:"backup_dir"`
	Delete    bool     `json:"delete"`
	Exclude   []string `json:"exclude"`

	// Application settings
	LogLevel  string `json:"log_level"`
	ShowCount bool   `json:"show_count"`
	ShowSize  bool   `json:"show_size"`

	// Runtime settings
	MaxConcurrency int           `json:"max_concurrency"`
	RetryAttempts  int           `json:"retry_attempts"`
	RetryDelay     time.Duration `json:"retry_delay"`
}

// Options represents command-line options for configuration
type Options struct {
	ConfigFile string
	BackupDir  string
	LogLevel   string
	Delete     bool
	Exclude    []string
	ShowCount  bool
	ShowSize   bool
}

// Load creates a new configuration from options and environment variables
func Load(opts Options) (*Config, error) {
	cfg := &Config{
		LogLevel:       "error",
		MaxConcurrency: 5,
		RetryAttempts:  3,
		RetryDelay:     time.Second * 2,
	}

	// Load from environment variables
	if err := cfg.loadFromEnv(); err != nil {
		return nil, fmt.Errorf("failed to load from environment: %w", err)
	}

	// Override with command-line options
	if opts.LogLevel != "" {
		cfg.LogLevel = opts.LogLevel
	}
	if opts.Delete {
		cfg.Delete = opts.Delete
	}
	if len(opts.Exclude) > 0 {
		cfg.Exclude = opts.Exclude
	}
	cfg.ShowCount = opts.ShowCount
	cfg.ShowSize = opts.ShowSize

	// Set backup directory
	if err := cfg.setBackupDir(opts.BackupDir); err != nil {
		return nil, fmt.Errorf("failed to set backup directory: %w", err)
	}

	// Validate configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadFromEnv() error {
	// Dropbox OAuth2 credentials
	c.ClientID = os.Getenv("DROPBOX_CLIENT_ID")
	c.ClientSecret = os.Getenv("DROPBOX_CLIENT_SECRET")
	c.AccessToken = os.Getenv("DROPBOX_ACCESS_TOKEN")
	c.RefreshToken = os.Getenv("DROPBOX_REFRESH_TOKEN")

	return nil
}

func (c *Config) setBackupDir(backupDir string) error {
	// Priority: command-line flag > environment variable > default
	if backupDir != "" {
		c.BackupDir = backupDir
	} else if envDir := os.Getenv("DROPBOX_BACKUP_FOLDER"); envDir != "" {
		c.BackupDir = envDir
	} else {
		// Create default backup folder with timestamp
		timestamp := time.Now().Format("2006-01-02-15-04-05")
		c.BackupDir = fmt.Sprintf("./dropbox_backup_%s", timestamp)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(c.BackupDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for backup directory: %w", err)
	}
	c.BackupDir = absPath

	// Create directory if it doesn't exist
	if err := os.MkdirAll(c.BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	return nil
}

func (c *Config) validate() error {
	if c.ClientID == "" {
		return fmt.Errorf("DROPBOX_CLIENT_ID environment variable is required")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("DROPBOX_CLIENT_SECRET environment variable is required")
	}
	if c.BackupDir == "" {
		return fmt.Errorf("backup directory is required")
	}

	// Validate log level
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[c.LogLevel] {
		return fmt.Errorf("invalid log level: %s (must be debug, info, warn, or error)", c.LogLevel)
	}

	return nil
}
