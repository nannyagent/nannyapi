#!/bin/bash
# Complete API test for Postman testing

set -e

echo "=== Complete NannyAPI Test ==="
echo ""

# 1. Create user
echo "1. Creating user..."
EMAIL="test-$(date +%s)@example.com"
USER_RESPONSE=$(curl -s -X POST http://localhost:8090/api/collections/users/records \
  -H "Content-Type: application/json" \
  -d '{"email":"'"$EMAIL"'","password":"TestPassword123!","passwordConfirm":"TestPassword123!"}')
USER_ID=$(echo "$USER_RESPONSE" | jq -r '.id')
echo "User created: $USER_ID"

# 2. Authenticate user
echo ""
echo "2. Authenticating user..."
AUTH_RESPONSE=$(curl -s -X POST http://localhost:8090/api/collections/users/auth-with-password \
  -H "Content-Type: application/json" \
  -d '{"identity":"'"$EMAIL"'","password":"TestPassword123!"}')
USER_TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.token')
echo "Token: ${USER_TOKEN:0:50}..."

# 3. Start device auth flow
echo ""
echo "3. Starting device auth..."
DEVICE_RESPONSE=$(curl -s -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"device-auth-start"}')
USER_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.user_code')
DEVICE_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.device_code')
echo "User code: $USER_CODE"
echo "Device code: $DEVICE_CODE"

# 4. User authorizes device
echo ""
echo "4. User authorizing device..."
AUTH_RESULT=$(curl -s -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"authorize","user_code":"'"$USER_CODE"'"}')
echo "$AUTH_RESULT" | jq '.'

# 5. Agent registers with device code
echo ""
echo "5. Agent registering..."
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"register","device_code":"'"$DEVICE_CODE"'","hostname":"test-host","platform":"linux","version":"1.0.0","primary_ip":"192.168.1.100","all_ips":["192.168.1.100","10.0.0.5"],"kernel_version":"5.4.0-42-generic"}')
AGENT_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.agent_id')
ACCESS_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.access_token')
REFRESH_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.refresh_token')
echo "Agent ID: $AGENT_ID"
echo "Access Token: ${ACCESS_TOKEN:0:30}..."
echo "Refresh Token: ${REFRESH_TOKEN:0:30}..."

# 6. Ingest metrics
echo ""
echo "6. Ingesting metrics..."
METRICS_RESPONSE=$(curl -s -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer $AGENT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "action":"ingest-metrics",
    "metrics":{
      "cpu_usage":75.5,
      "memory_usage":82.3,
      "network_rx_bytes":5242880,
      "network_tx_bytes":2621440,
      "memory_used_gb":12.5,
      "memory_total_gb":16.0,
      "disk_used_gb":450.0,
      "disk_total_gb":500.0
    }
  }')
echo "$METRICS_RESPONSE" | jq '.'

# 7. Refresh token
echo ""
echo "7. Refreshing token..."
REFRESH_RESPONSE=$(curl -s -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"refresh","refresh_token":"'"$REFRESH_TOKEN"'"}')
echo "$REFRESH_RESPONSE" | jq '.'

# 8. Update password
echo ""
echo "8. Updating password..."
UPDATE_RESPONSE=$(curl -s -X PATCH http://localhost:8090/api/collections/users/records/$USER_ID \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"oldPassword":"TestPassword123!","password":"NewPassword123!","passwordConfirm":"NewPassword123!"}')
echo "Password updated"

# 9. Delete user
echo ""
echo "9. Deleting user..."
curl -s -X DELETE http://localhost:8090/api/collections/users/records/$USER_ID \
  -H "Authorization: Bearer $USER_TOKEN"
echo "User deleted"

echo ""
echo "=== All tests completed successfully ==="
