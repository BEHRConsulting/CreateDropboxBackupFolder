#!/bin/bash

# Dropbox Backup Setup Script
# This script helps you set up the Dropbox backup tool

set -e

echo "🚀 Setting up Dropbox Backup Tool..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+ and try again."
    echo "   Visit: https://golang.org/doc/install"
    exit 1
fi

echo "✅ Go is installed"

# Build the application
echo "🔨 Building application..."
go build -o create-dropbox-backup-folder

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
else
    echo "❌ Build failed!"
    exit 1
fi

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo "📄 Creating .env file..."
    cp .env.example .env
    echo "✅ Created .env file. Please edit it with your Dropbox credentials."
else
    echo "ℹ️  .env file already exists"
fi

# Make the binary executable
chmod +x create-dropbox-backup-folder

echo ""
echo "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo "1. Edit .env file with your Dropbox app credentials"
echo "2. Get your credentials from: https://www.dropbox.com/developers/apps"
echo "3. Run: ./create-dropbox-backup-folder --help"
echo ""
echo "Example usage:"
echo "  ./create-dropbox-backup-folder --loglevel info --backup-dir ./my-backup"
echo ""
