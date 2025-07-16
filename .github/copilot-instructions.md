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

