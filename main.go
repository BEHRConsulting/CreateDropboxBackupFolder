package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"create-dropbox-backup-folder/internal/backup"
	"create-dropbox-backup-folder/internal/config"
	"create-dropbox-backup-folder/internal/dropbox"

	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "create-dropbox-backup-folder",
	Short: "A tool to backup Dropbox files to a local directory",
	Long: `create-dropbox-backup-folder is a command-line tool that authenticates with
Dropbox using OAuth2 and downloads all files and folders to a specified
local directory, preserving the folder structure.

The tool supports incremental backups, exclusion patterns, and configurable
logging levels. It handles authentication securely and efficiently manages
API calls to avoid rate limits.`,
	RunE: runBackup,
}

var (
	flagDelete     bool
	flagExclude    []string
	flagLogLevel   string
	flagBackupDir  string
	flagConfigFile string
	flagCount      bool
	flagSize       bool
)

func init() {
	rootCmd.Flags().BoolVar(&flagDelete, "delete", false, "Delete local files that don't exist in Dropbox")
	rootCmd.Flags().StringSliceVar(&flagExclude, "exclude", []string{}, "Exclude patterns (e.g., '*.tmp', 'temp/', '@filename')")
	rootCmd.Flags().StringVar(&flagLogLevel, "loglevel", "error", "Log level (debug, info, warn, error)")
	rootCmd.Flags().StringVar(&flagBackupDir, "backup-dir", "", "Custom backup directory (overrides DROPBOX_BACKUP_FOLDER)")
	rootCmd.Flags().StringVar(&flagConfigFile, "config", "", "Path to configuration file")
	rootCmd.Flags().BoolVar(&flagCount, "count", false, "Display total number of files and directories processed")
	rootCmd.Flags().BoolVar(&flagSize, "size", false, "Display total size of files processed")

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("create-dropbox-backup-folder %s\nCommit: %s\nBuilt: %s\n", version, commit, date)
		},
	})

	// Add auth command for interactive authentication
	rootCmd.AddCommand(&cobra.Command{
		Use:   "auth",
		Short: "Authenticate with Dropbox using OAuth2",
		Long: `Start an interactive OAuth2 authentication flow with Dropbox.
This will open your web browser and guide you through the authentication process.
After successful authentication, save the tokens to your .env file.`,
		RunE: runAuth,
	})
}

func runBackup(cmd *cobra.Command, args []string) error {
	// Parse and validate configuration
	cfg, err := config.Load(config.Options{
		ConfigFile: flagConfigFile,
		BackupDir:  flagBackupDir,
		LogLevel:   flagLogLevel,
		Delete:     flagDelete,
		Exclude:    flagExclude,
		ShowCount:  flagCount,
		ShowSize:   flagSize,
	})
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Setup logging
	setupLogging(cfg.LogLevel)

	slog.Info("Starting Dropbox backup",
		slog.String("backup_dir", cfg.BackupDir),
		slog.String("log_level", cfg.LogLevel),
		slog.Bool("delete_enabled", cfg.Delete),
		slog.Int("exclude_patterns", len(cfg.Exclude)),
	)

	// Create backup engine
	backupEngine, err := backup.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create backup engine: %w", err)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run backup
	if err := backupEngine.Run(ctx); err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	slog.Info("Backup completed successfully")
	return nil
}

func setupLogging(level string) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelError
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func runAuth(cmd *cobra.Command, args []string) error {
	// Setup basic logging
	setupLogging("info")

	// Check for required environment variables
	clientID := os.Getenv("DROPBOX_CLIENT_ID")
	clientSecret := os.Getenv("DROPBOX_CLIENT_SECRET")

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf(`missing required environment variables:
Please set DROPBOX_CLIENT_ID and DROPBOX_CLIENT_SECRET in your .env file.

Get these credentials from: https://www.dropbox.com/developers/apps

Example .env file:
DROPBOX_CLIENT_ID="your_app_key_here"
DROPBOX_CLIENT_SECRET="your_app_secret_here"`)
	}

	fmt.Println("🔐 Starting Dropbox OAuth2 authentication...")
	fmt.Println("📱 This will open your web browser for authentication.")
	fmt.Println("")

	// Import the dropbox package
	// Note: We need to add the import at the top of the file
	token, err := authenticateInteractively(clientID, clientSecret)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Println("")
	fmt.Println("✅ Authentication successful!")
	fmt.Println("")
	fmt.Println("🔑 Add these tokens to your .env file:")
	fmt.Println("")
	fmt.Printf("DROPBOX_ACCESS_TOKEN=\"%s\"\n", token.AccessToken)
	if token.RefreshToken != "" {
		fmt.Printf("DROPBOX_REFRESH_TOKEN=\"%s\"\n", token.RefreshToken)
	}
	fmt.Println("")
	fmt.Println("💡 You can now run the backup command:")
	fmt.Println("   ./create-dropbox-backup-folder --loglevel info")

	return nil
}

// authenticateInteractively handles the interactive OAuth flow
func authenticateInteractively(clientID, clientSecret string) (*oauth2.Token, error) {
	// Use the interactive authentication from our dropbox package
	return dropbox.AuthenticateWithStoredToken(clientID, clientSecret, "", "")
}
