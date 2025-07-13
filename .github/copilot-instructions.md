# Copilot Instructions

<!-- Use this file to provide workspace-specific custom instructions to Copilot. For more details, visit https://code.visualstudio.com/docs/copilot/copilot-customization#_use-a-githubcopilotinstructionsmd-file -->

## Project Overview

This is a Golang application called `create-dropbox-backup-folder` that creates backups of Dropbox accounts to local directories.

## Key Requirements

- Use Go 1.21+ features and best practices
- Implement OAuth2 authentication for Dropbox API
- Support incremental backups (only download newer files)
- Provide CLI flags for deletion, exclusion patterns, and logging levels
- Handle rate limiting and error recovery gracefully
- Use structured logging with configurable levels
- Follow Go project layout standards

## Architecture Guidelines

- Use the `/internal` directory for private application code
- Separate concerns into modules: dropbox client, backup engine, configuration
- Use interfaces for testability and modularity
- Implement proper error handling with wrapped errors
- Use context for cancellation and timeouts

## Dependencies

- Use `github.com/dropbox/dropbox-sdk-go-unofficial/v6/dropbox` for Dropbox API
- Use `github.com/spf13/cobra` for CLI framework
- Use `github.com/sirupsen/logrus` or `slog` for structured logging
- Use `golang.org/x/oauth2` for OAuth2 implementation

## Code Style

- Follow Go conventions and use `gofmt`
- Write comprehensive unit tests
- Use descriptive variable and function names
- Add godoc comments for exported functions
- Handle errors explicitly, don't ignore them

## Original Prompt

- Given to Claude Sonnet 4

```
Application name is create-dropbox-backup-folder

This project is a Golang application that copies all files from a Dropbox account into a local folder. The app should authenticate with Dropbox, list all files and folders recursively, and download them to a specified local directory, preserving the folder structure.

If a file exists in the specified local directory, it should only be overwritten if the file in Dropbox is newer than the local file.

Use production authentication methods and libraries for Dropbox, such as OAuth2, to ensure secure access to the user's Dropbox account.

If the parameter --delete is provided and a file does not exist in the local directory, delete the file from the local directory.

If the parameter --exclude is provided, the app should skip downloading files that match the specified patterns (e.g., `*.tmp`, `*.log`) or directories (e.g., `temp/`, `cache/`) or @filename for a file of exclusions.

if the parameter --count is provided, display total number of files and directories processed.

if the parameter --size is provided, display the total size of files processed.

If the environment variable `DROPBOX_BACKUP_FOLDER` is not set, the app should create a default backup folder named `./dropbox_backup_yyyy-mm-dd-hh-mm-ss`, where the timestamp reflects the current date and time.

If the environment variable `DROPBOX_BACKUP_FOLDER` is set, the app should use that path as the backup folder.

If the parameter --loglevel is provided, the app should set the logging level accordingly. The default logging level should be "error", but it can be set to "debug" or "info" based on the user's preference.

Create unit tests.

The app should handle errors gracefully, providing clear messages if something goes wrong, such as authentication failures, network issues, or file system errors.

The app should be efficient in terms of API calls to Dropbox, minimizing the number of requests made to avoid hitting rate limits.

The code should be well-structured and modular, making it easy to maintain and extend in the future.

The app should include comments and documentation to explain the functionality and usage.
```
