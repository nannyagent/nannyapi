#!/bin/bash

# NannyAPI Agent Authorization Flow Tests
# Base URL
BASE_URL="http://127.0.0.1:8090"

echo "================================"
echo "Agent Authorization Flow Tests"
echo "================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Step 1: Create a user account first
echo -e "${YELLOW}1. Creating user account for agent authorization...${NC}"
CREATE_USER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/records" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "agent-owner@example.com",
    "password": "AgentOwner123!@#",
    "passwordConfirm": "AgentOwner123!@#",
    "emailVisibility": true
  }')

USER_ID=$(echo $CREATE_USER_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo -e "${GREEN}User ID: $USER_ID${NC}"
echo ""

# Step 2: Start Device Auth Flow
echo -e "${YELLOW}2. Starting device authorization flow...${NC}"
DEVICE_AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d '{
    "action": "device-auth-start"
  }')

echo "Response:"
echo "$DEVICE_AUTH_RESPONSE" | jq '.'
DEVICE_CODE=$(echo $DEVICE_AUTH_RESPONSE | jq -r '.device_code')
USER_CODE=$(echo $DEVICE_AUTH_RESPONSE | jq -r '.user_code')
VERIFICATION_URL=$(echo $DEVICE_AUTH_RESPONSE | jq -r '.verification_url')

echo -e "${BLUE}Device Code: $DEVICE_CODE${NC}"
echo -e "${BLUE}User Code: $USER_CODE${NC}"
echo -e "${BLUE}Verification URL: $VERIFICATION_URL${NC}"
echo ""

# Step 3: User authenticates
echo -e "${YELLOW}3. User authenticates...${NC}"
AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/auth-with-password" \
  -H "Content-Type: application/json" \
  -d '{
    "identity": "agent-owner@example.com",
    "password": "AgentOwner123!@#"
  }')

USER_TOKEN=$(echo $AUTH_RESPONSE | jq -r '.token')
echo -e "${GREEN}User Token: ${USER_TOKEN:0:50}...${NC}"
echo ""

# Step 4: User authorizes the device (simulating web flow)
echo -e "${YELLOW}4. User authorizes the device with user code...${NC}"
AUTHORIZE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${USER_TOKEN}" \
  -d "{
    \"action\": \"authorize\",
    \"user_code\": \"${USER_CODE}\"
  }")

echo "Response:"
echo "$AUTHORIZE_RESPONSE" | jq '.'
echo ""

# Step 5: Agent polls and registers (using device code)
echo -e "${YELLOW}5. Agent registers using device code...${NC}"
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d "{
    \"action\": \"register\",
    \"device_code\": \"${DEVICE_CODE}\",
    \"agent_name\": \"Test-MacBook-Pro\",
    \"platform\": \"darwin\",
    \"hostname\": \"test-mbp.local\",
    \"ip_address\": \"192.168.1.100\",
    \"version\": \"1.0.0\"
  }")

echo "Response:"
echo "$REGISTER_RESPONSE" | jq '.'
AGENT_TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.access_token')
AGENT_REFRESH_TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.refresh_token')
AGENT_ID=$(echo $REGISTER_RESPONSE | jq -r '.agent_id')

echo -e "${GREEN}Agent Token: ${AGENT_TOKEN:0:50}...${NC}"
echo -e "${GREEN}Agent ID: ${AGENT_ID}${NC}"
echo ""

# Step 6: Test agent health check
echo -e "${YELLOW}6. Testing agent health check...${NC}"
HEALTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "health"
  }')

echo "Response:"
echo "$HEALTH_RESPONSE" | jq '.'
echo ""

# Step 7: Refresh agent token
echo -e "${YELLOW}7. Refreshing agent token...${NC}"
REFRESH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d "{
    \"action\": \"refresh\",
    \"token\": \"${AGENT_TOKEN}\"
  }")

echo "Response:"
echo "$REFRESH_RESPONSE" | jq '.'
NEW_TOKEN=$(echo $REFRESH_RESPONSE | jq -r '.token')
echo -e "${GREEN}New Token: ${NEW_TOKEN:0:50}...${NC}"
echo ""

# Step 8: List user's agents
echo -e "${YELLOW}8. Listing user's agents...${NC}"
LIST_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${USER_TOKEN}" \
  -d '{
    "action": "list"
  }')

echo "Response:"
echo "$LIST_RESPONSE" | jq '.'
echo ""

# Step 9: Revoke agent
echo -e "${YELLOW}9. Revoking agent...${NC}"
REVOKE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${USER_TOKEN}" \
  -d "{
    \"action\": \"revoke\",
    \"agent_id\": \"${AGENT_ID}\"
  }")

echo "Response:"
echo "$REVOKE_RESPONSE" | jq '.'
echo ""

# Step 10: Try to use revoked agent (should fail)
echo -e "${YELLOW}10. Testing revoked agent access (should fail)...${NC}"
REVOKED_HEALTH=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "health"
  }')

echo "Response:"
echo "$REVOKED_HEALTH" | jq '.'
if echo "$REVOKED_HEALTH" | grep -qi "revoked\|unauthorized"; then
  echo -e "${GREEN} Agent revocation is working!${NC}"
else
  echo -e "${RED} Agent revocation not working${NC}"
fi
echo ""

echo "================================"
echo "Agent Authorization Tests Complete"
echo "================================"
echo ""
echo -e "${BLUE}Summary:${NC}"
echo "- Device Code: $DEVICE_CODE"
echo "- User Code: $USER_CODE"
echo "- Agent ID: $AGENT_ID"
echo "- Agent Token: ${AGENT_TOKEN:0:30}..."
