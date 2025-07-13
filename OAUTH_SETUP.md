# Dropbox OAuth2 Setup Guide

This guide walks you through setting up OAuth2 authentication for the Dropbox Backup Tool.

## Step 1: Create a Dropbox App

1. Go to the [Dropbox App Console](https://www.dropbox.com/developers/apps)
2. Click "Create app"
3. Choose "Scoped access"
4. Choose "Full Dropbox" access type
5. Give your app a name (e.g., "My Backup Tool")
6. Click "Create app"

## Step 2: Configure App Permissions

In your app settings, go to the "Permissions" tab and enable:

- `files.metadata.read` - Read file and folder metadata
- `files.content.read` - Read file content

## Step 3: Get Your App Credentials

From the "Settings" tab, note down:

- **App key** (this is your `DROPBOX_CLIENT_ID`)
- **App secret** (this is your `DROPBOX_CLIENT_SECRET`)

## Step 4: Generate Access Token

### Option A: Generate Access Token (Simple)

In the "Settings" tab, scroll down to "Generated access token" and click "Generate".

**Important**: This token expires and should only be used for testing.

### Option B: OAuth2 Flow (Recommended for Production)

For production use, implement the full OAuth2 flow:

1. **Authorization URL**: Direct users to:
   ```
   https://www.dropbox.com/oauth2/authorize?client_id=YOUR_CLIENT_ID&response_type=code&redirect_uri=YOUR_REDIRECT_URI
   ```

2. **Exchange Code for Token**: After user authorization, exchange the code for tokens:
   ```bash
   curl -X POST https://api.dropboxapi.com/oauth2/token \
     -d grant_type=authorization_code \
     -d code=AUTHORIZATION_CODE \
     -d client_id=YOUR_CLIENT_ID \
     -d client_secret=YOUR_CLIENT_SECRET
   ```

3. **Refresh Token**: Use the refresh token to get new access tokens when they expire.

## Step 5: Configure Environment Variables

Create a `.env` file (copy from `.env.example`):

```bash
DROPBOX_CLIENT_ID="your_app_key_here"
DROPBOX_CLIENT_SECRET="your_app_secret_here"
DROPBOX_ACCESS_TOKEN="your_access_token_here"
DROPBOX_REFRESH_TOKEN="your_refresh_token_here"  # Optional
```

## Step 6: Test Your Setup

Run the application to test your configuration:

```bash
# Build the application
make build

# Test with minimal backup (just list files)
./create-dropbox-backup-folder --loglevel debug --backup-dir ./test-backup
```

## Troubleshooting

### Common Issues

1. **Invalid Client ID/Secret**
   - Double-check your app key and secret
   - Ensure there are no extra spaces or characters

2. **Invalid Access Token**
   - Regenerate the access token
   - Check token hasn't expired
   - Ensure proper permissions are granted

3. **Permission Denied**
   - Verify app permissions in Dropbox console
   - Check if user has granted necessary scopes

4. **Rate Limited**
   - The tool handles rate limiting automatically
   - Reduce concurrency if needed

### Debug Mode

Always test with debug logging first:

```bash
./create-dropbox-backup-folder --loglevel debug --help
```

## Security Best Practices

1. **Never commit tokens to version control**
   - Use `.env` files (already in `.gitignore`)
   - Use environment variables in production

2. **Rotate tokens regularly**
   - Implement token refresh logic
   - Monitor for unauthorized access

3. **Limit app permissions**
   - Only request necessary scopes
   - Use least privilege principle

4. **Secure storage**
   - Store tokens securely in production
   - Consider using key management services

## Production Deployment

For production use:

1. Implement full OAuth2 flow with refresh tokens
2. Store tokens securely (not in plain text files)
3. Add proper error handling for token expiration
4. Monitor API usage and respect rate limits
5. Log security events and token refreshes

## Resources

- [Dropbox OAuth2 Guide](https://developers.dropbox.com/oauth-guide)
- [Dropbox API Documentation](https://www.dropbox.com/developers/documentation)
- [OAuth2 RFC](https://tools.ietf.org/html/rfc6749)
