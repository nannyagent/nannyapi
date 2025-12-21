#!/bin/bash
set -e

BASE_URL="http://127.0.0.1:8090"

echo "=== Testing User Agent Access ==="
echo ""

# 1. Create user
echo "1. Creating user..."
USER_EMAIL="testuser@example.com"
USER_PASSWORD="TestPass123!@#"

USER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/records" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\",\"passwordConfirm\":\"${USER_PASSWORD}\"}")

USER_ID=$(echo "$USER_RESPONSE" | jq -r '.id')
echo "User created: $USER_ID"

# 2. Login
echo ""
echo "2. Logging in..."
AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/auth-with-password" \
  -H "Content-Type: application/json" \
  -d "{\"identity\":\"${USER_EMAIL}\",\"password\":\"${USER_PASSWORD}\"}")

USER_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.token')
echo "User token: ${USER_TOKEN:0:50}..."

# 3. Start device auth
echo ""
echo "3. Starting device auth..."
DEVICE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d '{"action":"device-auth-start"}')

DEVICE_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.device_code')
USER_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.user_code')
echo "Device code: $DEVICE_CODE"
echo "User code: $USER_CODE"

# 4. Authorize device
echo ""
echo "4. Authorizing device..."
curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${USER_TOKEN}" \
  -d "{\"action\":\"authorize\",\"user_code\":\"${USER_CODE}\"}" | jq .

# 5. Register agent
echo ""
echo "5. Registering agent..."
AGENT_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d "{\"action\":\"register\",\"device_code\":\"${DEVICE_CODE}\",\"hostname\":\"test-agent\",\"platform\":\"linux\",\"version\":\"1.0.0\"}")

AGENT_ID=$(echo "$AGENT_RESPONSE" | jq -r '.agent_id')
AGENT_TOKEN=$(echo "$AGENT_RESPONSE" | jq -r '.access_token')
echo "Agent registered: $AGENT_ID"
echo "Agent token: ${AGENT_TOKEN:0:50}..."

# 6. Try to list agents via API (this should work)
echo ""
echo "6. Listing agents via PocketBase API..."
curl -s "${BASE_URL}/api/collections/agents/records" \
  -H "Authorization: Bearer ${USER_TOKEN}" | jq .

# 7. Try to list agents via custom endpoint
echo ""
echo "7. Listing agents via custom endpoint..."
curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${USER_TOKEN}" \
  -d '{"action":"list"}' | jq .

echo ""
echo "=== Test Complete ==="
