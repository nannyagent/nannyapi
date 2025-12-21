#!/bin/bash
# Test account lockout functionality

set -e

echo "=== Testing Account Lockout ==="

# Create a test user
EMAIL="lockout-test-$(date +%s)@example.com"
PASSWORD="TestPassword123!"

echo "1. Creating test user: $EMAIL"
USER_RESPONSE=$(curl -s -X POST http://localhost:8090/api/collections/users/records \
  -H "Content-Type: application/json" \
  -d '{"email":"'"$EMAIL"'","password":"'"$PASSWORD"'","passwordConfirm":"'"$PASSWORD"'"}')
USER_ID=$(echo "$USER_RESPONSE" | jq -r '.id')
echo "Created user: $USER_ID"

# Try 6 failed login attempts (should trigger lockout at 5)
echo ""
echo "2. Attempting 6 failed logins to trigger lockout..."
for i in {1..6}; do
  echo "  Attempt $i..."
  RESPONSE=$(curl -s -X POST http://localhost:8090/api/collections/users/auth-with-password \
    -H "Content-Type: application/json" \
    -d '{"identity":"'"$EMAIL"'","password":"WrongPassword123!"}')
  ERROR=$(echo "$RESPONSE" | jq -r '.message // .error // "unknown"')
  echo "    Response: $ERROR"
  sleep 1
done

# Try correct password (should be locked)
echo ""
echo "3. Trying correct password (should be locked)..."
LOCKED_RESPONSE=$(curl -s -X POST http://localhost:8090/api/collections/users/auth-with-password \
  -H "Content-Type: application/json" \
  -d '{"identity":"'"$EMAIL"'","password":"'"$PASSWORD"'"}')
echo "$LOCKED_RESPONSE" | jq '.'

# Check account_lockout table
echo ""
echo "4. Checking account_lockout records..."
TOKEN=$(curl -s -X POST http://localhost:8090/api/collections/users/auth-with-password \
  -H "Content-Type: application/json" \
  -d '{"identity":"admin@example.com","password":"AdminPassword123!"}' | jq -r '.token // empty')

if [ -n "$TOKEN" ]; then
  curl -s -X GET "http://localhost:8090/api/collections/account_lockout/records?filter=(user_id='$USER_ID')" \
    -H "Authorization: Bearer $TOKEN" | jq '.items'
else
  echo "Note: Need admin user to query account_lockout collection"
fi

echo ""
echo "=== Account Lockout Test Complete ==="
