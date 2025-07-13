package backup

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"create-dropbox-backup-folder/internal/config"
	"create-dropbox-backup-folder/internal/dropbox"
)

// Engine handles the backup process
type Engine struct {
	config        *config.Config
	dropboxClient *dropbox.Client
	semaphore     chan struct{}
}

// Stats tracks backup statistics
type Stats struct {
	TotalFiles      int
	TotalFolders    int
	DownloadedFiles int
	SkippedFiles    int
	DeletedFiles    int
	TotalBytes      uint64
	StartTime       time.Time
	EndTime         time.Time
}

// New creates a new backup engine
func New(cfg *config.Config) (*Engine, error) {
	// Create Dropbox client with enhanced authentication
	dbxClient, err := dropbox.New(
		cfg.ClientID,
		cfg.ClientSecret,
		cfg.AccessToken,
		cfg.RefreshToken,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Dropbox client: %w", err)
	}

	// Validate token and permissions
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := dbxClient.ValidateTokenScopes(ctx); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	slog.Info("Dropbox authentication successful")

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, cfg.MaxConcurrency)

	return &Engine{
		config:        cfg,
		dropboxClient: dbxClient,
		semaphore:     semaphore,
	}, nil
}

// Run executes the backup process
func (e *Engine) Run(ctx context.Context) error {
	stats := &Stats{
		StartTime: time.Now(),
	}

	slog.Info("Starting backup process",
		slog.String("backup_dir", e.config.BackupDir),
		slog.Int("max_concurrency", e.config.MaxConcurrency),
	)

	// Check and refresh token if needed
	if !e.dropboxClient.IsTokenValid() {
		slog.Info("Token needs refresh, attempting to refresh...")
		if err := e.dropboxClient.RefreshToken(ctx); err != nil {
			return fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	// List all files from Dropbox
	slog.Info("Listing files from Dropbox...")
	dropboxFiles, err := e.dropboxClient.ListAll(ctx)
	if err != nil {
		// Try refreshing token and retry once if listing fails
		slog.Warn("File listing failed, attempting token refresh...")
		if refreshErr := e.dropboxClient.RefreshToken(ctx); refreshErr != nil {
			return fmt.Errorf("failed to list Dropbox files and refresh token: %w", err)
		}

		// Retry listing after token refresh
		dropboxFiles, err = e.dropboxClient.ListAll(ctx)
		if err != nil {
			return fmt.Errorf("failed to list Dropbox files after token refresh: %w", err)
		}
	}

	// Count files and folders separately
	fileCount := 0
	folderCount := 0
	for _, file := range dropboxFiles {
		if file.IsFolder {
			folderCount++
		} else {
			fileCount++
		}
	}

	stats.TotalFiles = fileCount
	stats.TotalFolders = folderCount
	slog.Info("Found items in Dropbox",
		slog.Int("files", fileCount),
		slog.Int("folders", folderCount),
		slog.Int("total", len(dropboxFiles)),
	)

	// Filter files based on exclusion patterns
	filteredFiles := e.filterFiles(dropboxFiles)
	slog.Info("Files after filtering", slog.Int("count", len(filteredFiles)))

	// Download files concurrently
	if err := e.downloadFiles(ctx, filteredFiles, stats); err != nil {
		return fmt.Errorf("failed to download files: %w", err)
	}

	// Handle deletion if enabled
	if e.config.Delete {
		if err := e.deleteOrphanedFiles(ctx, filteredFiles, stats); err != nil {
			return fmt.Errorf("failed to delete orphaned files: %w", err)
		}
	}

	stats.EndTime = time.Now()
	e.logStats(stats)

	return nil
}

func (e *Engine) filterFiles(files []dropbox.FileInfo) []dropbox.FileInfo {
	if len(e.config.Exclude) == 0 {
		return files
	}

	var filtered []dropbox.FileInfo
	for _, file := range files {
		if !e.shouldExclude(file.Path) {
			filtered = append(filtered, file)
		} else {
			slog.Debug("Excluding file", slog.String("path", file.Path))
		}
	}

	return filtered
}

func (e *Engine) shouldExclude(path string) bool {
	for _, pattern := range e.config.Exclude {
		// Handle @filename pattern (exclusion file)
		if strings.HasPrefix(pattern, "@") {
			excludeFile := strings.TrimPrefix(pattern, "@")
			if e.isInExcludeFile(path, excludeFile) {
				return true
			}
			continue
		}

		// Handle directory patterns
		if strings.HasSuffix(pattern, "/") {
			if strings.HasPrefix(path, pattern) || strings.Contains(path, "/"+pattern) {
				return true
			}
			continue
		}

		// Handle file patterns
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}

		// Handle path patterns
		if matched, _ := filepath.Match(pattern, path); matched {
			return true
		}
	}

	return false
}

func (e *Engine) isInExcludeFile(path, excludeFile string) bool {
	// This is a simplified implementation
	// In a real implementation, you would read the exclude file
	// and check if the path matches any patterns in it
	return false
}

func (e *Engine) downloadFiles(ctx context.Context, files []dropbox.FileInfo, stats *Stats) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(files))

	for _, file := range files {
		if file.IsFolder {
			continue // Skip folders, they're created automatically
		}

		wg.Add(1)
		go func(file dropbox.FileInfo) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case e.semaphore <- struct{}{}:
				defer func() { <-e.semaphore }()
			case <-ctx.Done():
				errChan <- ctx.Err()
				return
			}

			if err := e.downloadFile(ctx, file, stats); err != nil {
				errChan <- fmt.Errorf("failed to download %s: %w", file.Path, err)
			}
		}(file)
	}

	// Wait for all downloads to complete
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect any errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) downloadFile(ctx context.Context, file dropbox.FileInfo, stats *Stats) error {
	localPath := filepath.Join(e.config.BackupDir, strings.TrimPrefix(file.Path, "/"))

	// Check if file already exists and is newer
	if e.shouldSkipFile(localPath, file) {
		stats.SkippedFiles++
		slog.Debug("Skipping file (already up to date)", slog.String("path", file.Path))
		return nil
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Download file
	reader, _, err := e.dropboxClient.Download(ctx, file.Path)
	if err != nil {
		return fmt.Errorf("failed to download from Dropbox: %w", err)
	}
	defer reader.Close()

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer localFile.Close()

	// Copy content
	written, err := io.Copy(localFile, reader)
	if err != nil {
		return fmt.Errorf("failed to write file content: %w", err)
	}

	// Set modification time
	if !file.ModTime.IsZero() {
		if err := os.Chtimes(localPath, file.ModTime, file.ModTime); err != nil {
			slog.Warn("Failed to set file modification time",
				slog.String("path", localPath),
				slog.String("error", err.Error()),
			)
		}
	}

	stats.DownloadedFiles++
	stats.TotalBytes += uint64(written)

	slog.Info("Downloaded file",
		slog.String("path", file.Path),
		slog.Int64("size", written),
	)

	return nil
}

func (e *Engine) shouldSkipFile(localPath string, remoteFile dropbox.FileInfo) bool {
	stat, err := os.Stat(localPath)
	if err != nil {
		return false // File doesn't exist, don't skip
	}

	// Compare modification times
	if !remoteFile.ModTime.IsZero() && stat.ModTime().After(remoteFile.ModTime) {
		return true // Local file is newer
	}

	// Compare sizes
	if stat.Size() == int64(remoteFile.Size) && !remoteFile.ModTime.IsZero() && stat.ModTime().Equal(remoteFile.ModTime) {
		return true // Same size and modification time
	}

	return false
}

func (e *Engine) deleteOrphanedFiles(ctx context.Context, dropboxFiles []dropbox.FileInfo, stats *Stats) error {
	// Create a map of Dropbox files for quick lookup
	dropboxFileMap := make(map[string]bool)
	for _, file := range dropboxFiles {
		localPath := filepath.Join(e.config.BackupDir, strings.TrimPrefix(file.Path, "/"))
		dropboxFileMap[localPath] = true
	}

	// Walk through local backup directory
	return filepath.Walk(e.config.BackupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file exists in Dropbox
		if !dropboxFileMap[path] {
			slog.Info("Deleting orphaned file", slog.String("path", path))
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete file %s: %w", path, err)
			}
			stats.DeletedFiles++
		}

		return nil
	})
}

func (e *Engine) logStats(stats *Stats) {
	duration := stats.EndTime.Sub(stats.StartTime)

	// Always log basic completion info
	slog.Info("Backup completed",
		slog.Int("downloaded_files", stats.DownloadedFiles),
		slog.Int("skipped_files", stats.SkippedFiles),
		slog.Int("deleted_files", stats.DeletedFiles),
		slog.Duration("duration", duration),
	)

	// Display count information if requested
	if e.config.ShowCount {
		fmt.Printf("\n📊 File Count Summary:\n")
		fmt.Printf("   Total files processed: %d\n", stats.TotalFiles)
		fmt.Printf("   Total folders processed: %d\n", stats.TotalFolders)
		fmt.Printf("   Total items: %d\n", stats.TotalFiles+stats.TotalFolders)
		fmt.Printf("   Files downloaded: %d\n", stats.DownloadedFiles)
		fmt.Printf("   Files skipped: %d\n", stats.SkippedFiles)
		if stats.DeletedFiles > 0 {
			fmt.Printf("   Files deleted: %d\n", stats.DeletedFiles)
		}
	}

	// Display size information if requested
	if e.config.ShowSize {
		fmt.Printf("\n💾 Size Summary:\n")
		fmt.Printf("   Total bytes processed: %s\n", formatBytes(stats.TotalBytes))
		if duration > 0 {
			bytesPerSecond := float64(stats.TotalBytes) / duration.Seconds()
			fmt.Printf("   Average transfer rate: %s/s\n", formatBytes(uint64(bytesPerSecond)))
		}
	}

	// Add a separator if either count or size was displayed
	if e.config.ShowCount || e.config.ShowSize {
		fmt.Println()
	}
}

// formatBytes formats byte counts in human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
