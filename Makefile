# Makefile for create-dropbox-backup-folder

.PHONY: build run test clean help install setup

# Default target
.DEFAULT_GOAL := help

# Build variables
BINARY_NAME=create-dropbox-backup-folder
VERSION=dev
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

## Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME)

## Run the application with debug logging
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME) --loglevel debug --help

## Run tests
test:
	@echo "Running tests..."
	go test ./...

## Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html

## Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

## Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

## Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

## Setup development environment
setup:
	@echo "Setting up development environment..."
	./setup.sh

## Show version information
version: build
	./$(BINARY_NAME) version

## Show help
help:
	@echo "Available targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -e 's/## /  /'
	@echo ""
	@echo "Usage examples:"
	@echo "  make build         # Build the application"
	@echo "  make run           # Build and run with help"
	@echo "  make test          # Run tests"
	@echo "  make setup         # Setup development environment"
	@echo ""
	@echo "New Statistics Features:"
	@echo "  ./$(BINARY_NAME) --count --size --help"
	@echo "  ./$(BINARY_NAME) --count --loglevel info --backup-dir ./test"
