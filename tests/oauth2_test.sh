#!/bin/bash
# OAuth2 authentication test
# Tests that OAuth2 credentials are configured and PocketBase OAuth2 endpoint is accessible

set -e

BASE_URL="http://127.0.0.1:8090"
GITHUB_CLIENT_ID="${GITHUB_CLIENT_ID:-}"
GITHUB_CLIENT_SECRET="${GITHUB_CLIENT_SECRET:-}"
GOOGLE_CLIENT_ID="${GOOGLE_CLIENT_ID:-}"
GOOGLE_CLIENT_SECRET="${GOOGLE_CLIENT_SECRET:-}"

echo "OAuth2 Configuration Test"
echo "========================"
echo ""

# Test 1: Verify environment variables
echo "Test 1: OAuth2 Credentials"
if [ -z "$GITHUB_CLIENT_ID" ] || [ -z "$GITHUB_CLIENT_SECRET" ]; then
  echo "FAIL: GitHub OAuth2 credentials missing"
  exit 1
else
  echo "PASS: GitHub ClientID=${GITHUB_CLIENT_ID:0:20}..."
fi

if [ -z "$GOOGLE_CLIENT_ID" ] || [ -z "$GOOGLE_CLIENT_SECRET" ]; then
  echo "FAIL: Google OAuth2 credentials missing"
  exit 1
else
  echo "PASS: Google ClientID=${GOOGLE_CLIENT_ID:0:40}..."
fi

echo ""
echo "Test 2: Database Configuration"

# Test 2: Verify OAuth2 is enabled in database
OAUTH_ENABLED=$(sqlite3 pb_data/data.db "SELECT json_extract(options, '$.oauth2.enabled') FROM _collections WHERE name='users'" 2>/dev/null || echo "0")

if [ "$OAUTH_ENABLED" != "1" ]; then
  echo "FAIL: OAuth2 not enabled in database"
  exit 1
fi
echo "PASS: OAuth2 enabled in database"

# Test 3: Verify GitHub provider configured
GITHUB_ID_IN_DB=$(sqlite3 pb_data/data.db "SELECT json_extract(options, '$.oauth2.providers[0].clientId') FROM _collections WHERE name='users'" 2>/dev/null || echo "")
if [ -z "$GITHUB_ID_IN_DB" ]; then
  echo "FAIL: GitHub OAuth2 provider not configured in database"
  exit 1
fi
echo "PASS: GitHub provider in database (${GITHUB_ID_IN_DB:0:20}...)"

# Test 4: Verify Google provider configured
GOOGLE_ID_IN_DB=$(sqlite3 pb_data/data.db "SELECT json_extract(options, '$.oauth2.providers[1].clientId') FROM _collections WHERE name='users'" 2>/dev/null || echo "")
if [ -z "$GOOGLE_ID_IN_DB" ]; then
  echo "FAIL: Google OAuth2 provider not configured in database"
  exit 1
fi
echo "PASS: Google provider in database (${GOOGLE_ID_IN_DB:0:40}...)"

echo ""
echo "Test 3: OAuth2 Redirect Endpoint"

# Test 5: Verify OAuth2 redirect endpoint exists
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/oauth2-redirect")
if [ "$HTTP_CODE" != "307" ]; then
  echo "FAIL: OAuth2 redirect endpoint returned HTTP $HTTP_CODE (expected 307)"
  exit 1
fi
echo "PASS: OAuth2 redirect endpoint accessible (HTTP 307)"

echo ""
echo "========================"
echo "All tests passed!"
echo ""
echo "OAuth2 Flow Instructions:"
echo "1. Frontend should register OAuth app redirect URL as: $BASE_URL/api/oauth2-redirect"
echo "2. Use PocketBase SDK: pb.collection('users').authWithOAuth2({ provider: 'github' })"
echo "3. This opens a popup and handles the full OAuth2 flow automatically"
echo "4. No custom endpoints needed - PocketBase handles it all"
