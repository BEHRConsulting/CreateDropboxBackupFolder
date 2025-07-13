# Create Dropbox Backup Folder

A command-line tool that creates complete backup of Dropbox account to local directory, preserving folder structure and providing incremental sync capabilities.

## AI Generated (mostly) 
- I kept tweeking the prompt to get the functions I needed.
- This was a test using Github's Copilot in agent mode and Claude Sonnet 4 AI model.
- The prompts I used the generate this app are on the bottom of the file [copilot-instructions.md](.github/copilot-instructions.md).

## Workflow
- My laptop does not have the internal storage to download my entire dropbox, so it doesn't get backed up as I wanted it to. I run this app on my external drive to get a full backup.
- After a backup/sync, I zip the folder with a date
- Next time I run the app, the updated files are downloaded and a new zip is made.

## Features

- **🔐 Production-Grade OAuth2**: Secure authentication with PKCE, automatic token refresh, and interactive flow
- **📦 Incremental Backups**: Only downloads files that are newer than local copies
- **🚫 Exclusion Patterns**: Skip files and directories based on patterns or exclusion files
- **🗑️ Deletion Sync**: Optionally delete local files that no longer exist in Dropbox
- **⚡ Concurrent Downloads**: Configurable concurrency for efficient downloading
- **📊 Structured Logging**: Configurable log levels (debug, info, warn, error)
- **🔄 Error Recovery**: Retry mechanisms and graceful error handling
- **🛡️ Security First**: Environment variable configuration, no hardcoded credentials

## Installation

### Prerequisites

- Go 1.21 or later
- Dropbox App credentials (Client ID and Secret)

### Build from Source

```bash
git clone <repository-url>
cd CreateDropboxBackupFolder
go build -o create-dropbox-backup-folder
```

## Configuration

### Environment Variables

Set the following environment variables before running the application:

```bash
export DROPBOX_CLIENT_ID="your_client_id"
export DROPBOX_CLIENT_SECRET="your_client_secret"
export DROPBOX_ACCESS_TOKEN="your_access_token"
export DROPBOX_REFRESH_TOKEN="your_refresh_token"  # Optional
export DROPBOX_BACKUP_FOLDER="/path/to/backup"     # Optional
```

### Dropbox App Setup

1. Go to [Dropbox App Console](https://www.dropbox.com/developers/apps)
2. Create a new app with "Full Dropbox" access
3. Note your App key (Client ID) and App secret (Client Secret)
4. Generate an access token or implement the full OAuth2 flow

## Usage

## Quick Start

### 1. **Setup Authentication**

```bash
# Set your Dropbox app credentials (from https://www.dropbox.com/developers/apps)
export DROPBOX_CLIENT_ID="your_app_key"
export DROPBOX_CLIENT_SECRET="your_app_secret"

# Run interactive OAuth2 authentication
./create-dropbox-backup-folder auth

# This will open your browser and guide you through secure authentication
# Save the returned tokens to your .env file
```

### 2. **Run Your First Backup**

```bash
# Basic backup to timestamped folder
./create-dropbox-backup-folder

# Custom backup with options
./create-dropbox-backup-folder --backup-dir ./my-backup --loglevel info
```

### Advanced Usage

```bash
# Exclude specific patterns
./create-dropbox-backup-folder --exclude "*.tmp" --exclude "*.log" --exclude "temp/"

# Use exclusion file
./create-dropbox-backup-folder --exclude "@.backupignore"

# Show detailed statistics
./create-dropbox-backup-folder --count --size --loglevel info

# Full production backup with all options
./create-dropbox-backup-folder \
  --backup-dir ./my-backup \
  --delete \
  --count \
  --size \
  --loglevel info \
  --exclude "*.tmp" \
  --exclude "cache/"
```

### Authentication Commands

| Command | Description |
|---------|-------------|
| `auth` | Interactive OAuth2 authentication flow |
| `version` | Show version and build information |

### Command-Line Options

| Flag | Description | Default |
|------|-------------|---------|
| `--backup-dir` | Custom backup directory | `./dropbox_backup_YYYY-MM-DD-HH-MM-SS` |
| `--delete` | Delete local files not in Dropbox | `false` |
| `--exclude` | Exclusion patterns (can be used multiple times) | `[]` |
| `--loglevel` | Log level (debug, info, warn, error) | `error` |
| `--config` | Path to configuration file | `""` |
| `--count` | Display total number of files and directories processed | `false` |
| `--size` | Display total size of files processed | `false` |

### Exclusion Patterns

- **File patterns**: `*.tmp`, `*.log`
- **Directory patterns**: `temp/`, `cache/`
- **Exclusion files**: `@.backupignore` (reads patterns from file)

### Statistics Output

The application provides detailed statistics about the backup process:

#### Count Statistics (`--count`)
```
📊 File Count Summary:
   Total files processed: 1,247
   Total folders processed: 156
   Total items: 1,403
   Files downloaded: 23
   Files skipped: 1,224
   Files deleted: 0
```

#### Size Statistics (`--size`)
```
💾 Size Summary:
   Total bytes processed: 2.3 GB
   Average transfer rate: 15.2 MB/s
```

#### Combined Output
Use both `--count` and `--size` flags together to see comprehensive statistics about your backup operation.

## Project Structure

```
.
├── main.go                    # Application entry point
├── internal/
│   ├── backup/
│   │   └── engine.go         # Backup orchestration logic
│   ├── config/
│   │   └── config.go         # Configuration management
│   └── dropbox/
│       └── client.go         # Dropbox API client wrapper
├── .github/
│   └── copilot-instructions.md
├── .vscode/
│   └── tasks.json
├── go.mod
├── go.sum
└── README.md
```

## Development

### Building

```bash
go build -o create-dropbox-backup-folder
```

### Testing

```bash
go test ./...
```

### Running in Development

```bash
go run main.go --loglevel debug --backup-dir ./test-backup
```

## Error Handling

The application includes comprehensive error handling for:

- Network connectivity issues
- API rate limiting
- File system errors
- Authentication failures
- Insufficient disk space

## Rate Limiting

The application respects Dropbox API rate limits by:

- Using configurable concurrency limits
- Implementing exponential backoff retry logic
- Monitoring API response headers

## Security

- **🔐 OAuth2 with PKCE**: Implements Proof Key for Code Exchange for enhanced security
- **🔄 Auto Token Refresh**: Automatic refresh of expired access tokens
- **🌐 HTTPS Only**: All API communications over encrypted channels
- **🔒 Secure Storage**: Tokens stored in environment variables, never in code
- **✅ Token Validation**: Validates permissions before starting backup
- **🛡️ Rate Limiting**: Respects API limits with exponential backoff

See [SECURITY.md](SECURITY.md) for detailed security implementation.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

[Add your license information here]

## Support

For issues and questions:
- Create an issue in the repository
- Check the logs with `--loglevel debug` for troubleshooting
- Ensure your Dropbox app has the correct permissions
