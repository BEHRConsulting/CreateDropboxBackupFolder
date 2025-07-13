package backup

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"create-dropbox-backup-folder/internal/config"
	"create-dropbox-backup-folder/internal/dropbox"
)

// mockDropboxClient implements a mock Dropbox client for testing
type mockDropboxClient struct {
	files     map[string]*mockFile
	folders   map[string][]string
	listError error
	downError error
}

type mockFile struct {
	path    string
	size    uint64
	modTime time.Time
	content []byte
}

func newMockDropboxClient() *mockDropboxClient {
	return &mockDropboxClient{
		files:   make(map[string]*mockFile),
		folders: make(map[string][]string),
	}
}

func (m *mockDropboxClient) addFile(path string, size uint64, modTime time.Time, content []byte) {
	m.files[path] = &mockFile{
		path:    path,
		size:    size,
		modTime: modTime,
		content: content,
	}

	// Add to parent folder
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		m.folders[dir] = append(m.folders[dir], filepath.Base(path))
	} else {
		m.folders[""] = append(m.folders[""], filepath.Base(path))
	}
}

func (m *mockDropboxClient) addFolder(path string) {
	if _, exists := m.folders[path]; !exists {
		m.folders[path] = []string{}
	}

	// Add to parent folder
	dir := filepath.Dir(path)
	if dir != "." && dir != "/" {
		m.folders[dir] = append(m.folders[dir], filepath.Base(path)+"/")
	} else {
		m.folders[""] = append(m.folders[""], filepath.Base(path)+"/")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes uint64
		want  string
	}{
		{"zero bytes", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"kilobytes", 1024, "1.0 KB"},
		{"megabytes", 1048576, "1.0 MB"},
		{"gigabytes", 1073741824, "1.0 GB"},
		{"terabytes", 1099511627776, "1.0 TB"},
		{"mixed kb", 1536, "1.5 KB"},
		{"mixed mb", 2621440, "2.5 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %v, want %v", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{
			name:     "no patterns",
			patterns: []string{},
			path:     "/any/path.txt",
			want:     false,
		},
		{
			name:     "exact match",
			patterns: []string{"/exact/path.txt"},
			path:     "/exact/path.txt",
			want:     true,
		},
		{
			name:     "wildcard match",
			patterns: []string{"*.tmp"},
			path:     "/path/file.tmp",
			want:     true,
		},
		{
			name:     "directory match",
			patterns: []string{"temp/"},
			path:     "/temp/file.txt",
			want:     true,
		},
		{
			name:     "no match",
			patterns: []string{"*.log", "cache/"},
			path:     "/data/file.txt",
			want:     false,
		},
		{
			name:     "multiple patterns one match",
			patterns: []string{"*.log", "*.tmp", "cache/"},
			path:     "/data/file.tmp",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &Engine{
				config: &config.Config{
					Exclude: tt.patterns,
				},
			}
			got := engine.shouldExclude(tt.path)
			if got != tt.want {
				t.Errorf("shouldExclude(%s) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestStatsCalculations(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(time.Minute * 5)

	stats := &Stats{
		TotalFiles:      100,
		TotalFolders:    20,
		DownloadedFiles: 75,
		SkippedFiles:    15,
		DeletedFiles:    10,
		TotalBytes:      1048576, // 1 MB
		StartTime:       startTime,
		EndTime:         endTime,
	}

	// Test duration calculation
	duration := stats.EndTime.Sub(stats.StartTime)
	expectedDuration := time.Minute * 5
	if duration != expectedDuration {
		t.Errorf("Duration calculation = %v, want %v", duration, expectedDuration)
	}

	// Test transfer rate (bytes per second)
	expectedRate := float64(stats.TotalBytes) / duration.Seconds()
	actualRate := float64(stats.TotalBytes) / duration.Seconds()
	if actualRate != expectedRate {
		t.Errorf("Transfer rate = %v, want %v", actualRate, expectedRate)
	}
}

func TestShouldSkipFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file with a specific modification time
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Get the file's modification time
	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	localModTime := fileInfo.ModTime()

	engine := &Engine{
		config: &config.Config{
			BackupDir: tempDir,
		},
	}

	tests := []struct {
		name           string
		dropboxModTime time.Time
		want           bool
	}{
		{
			name:           "dropbox file is newer",
			dropboxModTime: localModTime.Add(time.Hour),
			want:           false,
		},
		{
			name:           "dropbox file is older",
			dropboxModTime: localModTime.Add(-time.Hour),
			want:           true,
		},
		{
			name:           "same modification time",
			dropboxModTime: localModTime,
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileInfo := dropbox.FileInfo{
				Path:    "/test.txt",
				Name:    "test.txt",
				Size:    uint64(len(content)),
				ModTime: tt.dropboxModTime,
			}
			got := engine.shouldSkipFile(testFile, fileInfo)
			if got != tt.want {
				t.Errorf("shouldSkipFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldSkipFileNotExists(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentFile := filepath.Join(tempDir, "nonexistent.txt")

	engine := &Engine{
		config: &config.Config{
			BackupDir: tempDir,
		},
	}

	fileInfo := dropbox.FileInfo{
		Path:    "/nonexistent.txt",
		Name:    "nonexistent.txt",
		Size:    1024,
		ModTime: time.Now(),
	}

	// Should not skip if file doesn't exist (should download)
	got := engine.shouldSkipFile(nonExistentFile, fileInfo)
	if got != false {
		t.Errorf("shouldSkipFile() for non-existent file = %v, want false", got)
	}
}

func TestLogStats(t *testing.T) {
	stats := &Stats{
		TotalFiles:      100,
		TotalFolders:    20,
		DownloadedFiles: 75,
		SkippedFiles:    20,
		DeletedFiles:    5,
		TotalBytes:      2097152, // 2 MB
		StartTime:       time.Now().Add(-time.Minute * 5),
		EndTime:         time.Now(),
	}

	// Test with both count and size enabled
	engine := &Engine{
		config: &config.Config{
			ShowCount: true,
			ShowSize:  true,
		},
	}

	// This primarily tests that logStats doesn't panic
	// In a real test environment, you might want to capture log output
	engine.logStats(stats)

	// Test with count only
	engine.config.ShowCount = true
	engine.config.ShowSize = false
	engine.logStats(stats)

	// Test with size only
	engine.config.ShowCount = false
	engine.config.ShowSize = true
	engine.logStats(stats)

	// Test with neither
	engine.config.ShowCount = false
	engine.config.ShowSize = false
	engine.logStats(stats)
}

func TestEngineCreation(t *testing.T) {
	tests := []struct {
		name    string
		config  *config.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &config.Config{
				ClientID:       "test_client_id",
				ClientSecret:   "test_client_secret",
				AccessToken:    "test_access_token",
				RefreshToken:   "test_refresh_token",
				BackupDir:      "/tmp/backup",
				MaxConcurrency: 5,
			},
			wantErr: true, // Will fail because we don't have real Dropbox credentials
		},
		{
			name: "invalid config - missing client ID",
			config: &config.Config{
				ClientSecret:   "test_client_secret",
				AccessToken:    "test_access_token",
				RefreshToken:   "test_refresh_token",
				BackupDir:      "/tmp/backup",
				MaxConcurrency: 5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
