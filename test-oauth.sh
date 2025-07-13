#!/bin/bash

# Test OAuth2 configuration
echo "🔧 Testing OAuth2 Configuration..."

# Set test credentials if not already set
if [ -z "$DROPBOX_CLIENT_ID" ]; then
    echo "❌ DROPBOX_CLIENT_ID not set"
    echo "Please set your Dropbox app credentials:"
    echo "export DROPBOX_CLIENT_ID='your_app_key'"
    echo "export DROPBOX_CLIENT_SECRET='your_app_secret'"
    exit 1
fi

if [ -z "$DROPBOX_CLIENT_SECRET" ]; then
    echo "❌ DROPBOX_CLIENT_SECRET not set"
    exit 1
fi

echo "✅ Credentials set"
echo "Client ID: ${DROPBOX_CLIENT_ID:0:10}..."

# Test with debug logging
echo "🚀 Running auth command with debug logging..."
./create-dropbox-backup-folder auth --loglevel debug
