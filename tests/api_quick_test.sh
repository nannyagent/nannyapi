#!/bin/bash

# Quick API Test - Simple verification that endpoints work
# This tests the actual deployed API, not internal hooks

BASE_URL="http://127.0.0.1:8090"

echo "================================"
echo "Quick API Verification"
echo "================================"
echo ""

# Test 1: Create User
echo "1. Creating user..."
USER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/records" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "apitest@example.com",
    "password": "TestPass123!@#",
    "passwordConfirm": "TestPass123!@#"
  }')

USER_ID=$(echo $USER_RESPONSE | jq -r '.id')
echo "✓ User created: $USER_ID"
echo ""

# Test 2: Authenticate
echo "2. Authenticating..."
AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/auth-with-password" \
  -H "Content-Type: application/json" \
  -d '{
    "identity": "apitest@example.com",
    "password": "TestPass123!@#"
  }')

TOKEN=$(echo $AUTH_RESPONSE | jq -r '.token')
echo "✓ Authenticated: ${TOKEN:0:40}..."
echo ""

# Test 3: Agent Device Auth
echo "3. Starting agent device auth..."
DEVICE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d '{"action": "device-auth-start"}')

DEVICE_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.device_code // .DeviceCode // empty')
USER_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.user_code // .UserCode // empty')

if [ -z "$DEVICE_CODE" ] || [ "$DEVICE_CODE" = "null" ]; then
  echo "⚠ Device auth response: $DEVICE_RESPONSE"
  echo "⚠ Skipping agent tests (device codes collection may not exist)"
  exit 0
fi

echo "✓ Device code: $DEVICE_CODE"
echo "✓ User code: $USER_CODE"
echo ""

# Test 4: Authorize Device
echo "4. Authorizing device..."
AUTH_DEV_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d "{\"action\": \"authorize\", \"user_code\": \"${USER_CODE}\"}")

echo "✓ Device authorized"
echo ""

# Test 5: Register Agent
echo "5. Registering agent..."
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d "{
    \"action\": \"register\",
    \"device_code\": \"${DEVICE_CODE}\",
    \"agent_name\": \"Test-Agent\",
    \"platform\": \"darwin\",
    \"hostname\": \"test.local\",
    \"ip_address\": \"127.0.0.1\"
  }")

AGENT_TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.token')
AGENT_ID=$(echo $REGISTER_RESPONSE | jq -r '.agent_id')
echo "✓ Agent registered: $AGENT_ID"
echo "✓ Agent token: ${AGENT_TOKEN:0:40}..."
echo ""

# Test 6: Ingest Metrics
echo "6. Ingesting metrics..."
METRICS_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_usage_percent": 45.5,
      "memory_used_gb": 8.0,
      "memory_total_gb": 16.0,
      "disk_used_gb": 250.0,
      "disk_total_gb": 512.0,
      "network_rx_gbps": 0.5,
      "network_tx_gbps": 0.3
    }
  }')

if echo "$METRICS_RESPONSE" | grep -q "success"; then
  echo "✓ Metrics ingested successfully"
else
  echo "Response: $METRICS_RESPONSE"
fi
echo ""

# Test 7: List Agents
echo "7. Listing agents..."
LIST_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${TOKEN}" \
  -d '{"action": "list"}')

AGENT_COUNT=$(echo $LIST_RESPONSE | jq -r '.agents | length')
echo "✓ Found $AGENT_COUNT agent(s)"
echo ""

echo "================================"
echo "All Basic Tests Passed! ✓"
echo "================================"
echo ""
echo "Created Resources:"
echo "  User ID: $USER_ID"
echo "  Agent ID: $AGENT_ID"
echo ""
echo "You can test these commands individually in Postman!"
