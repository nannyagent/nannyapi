#!/bin/bash
# Test OAuth2 authentication - REQUIRES BROWSER INTERACTION

BASE_URL="http://127.0.0.1:8090"
FRONTEND_URL="http://localhost:8080"

echo "=== OAuth2 Test Instructions ==="
echo ""
echo "OAuth2 requires browser interaction. To test:"
echo ""
echo "1. GitHub OAuth:"
echo "   ${BASE_URL}/api/collections/users/auth-with-oauth2?provider=github&redirect=${FRONTEND_URL}/oauth-callback"
echo ""
echo "2. Google OAuth:"
echo "   ${BASE_URL}/api/collections/users/auth-with-oauth2?provider=google&redirect=${FRONTEND_URL}/oauth-callback"
echo ""
echo "After authorization, the callback URL will receive:"
echo "  - code: OAuth authorization code"
echo "  - state: CSRF protection token"
echo ""
echo "Frontend should exchange the code for a token using:"
echo "  POST ${BASE_URL}/api/collections/users/auth-with-oauth2"
echo "  Body: {code, codeVerifier, redirectUrl, provider}"
echo ""
echo "=== Verifying OAuth2 Configuration ==="
echo ""

# Check if OAuth credentials are set
if [ -z "$GITHUB_CLIENT_ID" ] || [ -z "$GITHUB_CLIENT_SECRET" ]; then
  echo "❌ GitHub OAuth2 credentials not set in environment"
else
  echo "✅ GitHub OAuth2 credentials configured"
  echo "   Client ID: ${GITHUB_CLIENT_ID:0:20}..."
fi

if [ -z "$GOOGLE_CLIENT_ID" ] || [ -z "$GOOGLE_CLIENT_SECRET" ]; then
  echo "❌ Google OAuth2 credentials not set in environment"
else
  echo "✅ Google OAuth2 credentials configured"
  echo "   Client ID: ${GOOGLE_CLIENT_ID:0:40}..."
fi

echo ""
echo "=== Checking Database Configuration ==="

# Check if OAuth is configured in database
OAUTH_ENABLED=$(sqlite3 pb_data/data.db "SELECT json_extract(options, '$.oauth2.enabled') FROM _collections WHERE name='users'")
GITHUB_CONFIGURED=$(sqlite3 pb_data/data.db "SELECT json_extract(options, '$.oauth2.providers[0].clientId') FROM _collections WHERE name='users'")
GOOGLE_CONFIGURED=$(sqlite3 pb_data/data.db "SELECT json_extract(options, '$.oauth2.providers[1].clientId') FROM _collections WHERE name='users'")

if [ "$OAUTH_ENABLED" = "1" ]; then
  echo "✅ OAuth2 enabled in database"
  [ -n "$GITHUB_CONFIGURED" ] && echo "✅ GitHub provider configured: ${GITHUB_CONFIGURED:0:20}..."
  [ -n "$GOOGLE_CONFIGURED" ] && echo "✅ Google provider configured: ${GOOGLE_CONFIGURED:0:40}..."
else
  echo "❌ OAuth2 not enabled in database"
  echo "   Run: bash scripts/reset-and-start.sh (with .env sourced)"
fi

echo ""
echo "=== Test Complete ==="
echo "OAuth2 requires manual browser testing - cannot be automated with curl"
