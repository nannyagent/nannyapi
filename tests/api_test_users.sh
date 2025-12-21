#!/bin/bash

# NannyAPI User Management Tests
# Base URL
BASE_URL="http://127.0.0.1:8090"

echo "================================"
echo "User Management API Tests"
echo "================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 1. Create a new user
echo -e "${YELLOW}1. Creating a new user...${NC}"
CREATE_USER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/records" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "testuser@example.com",
    "password": "SecurePass123!@#",
    "passwordConfirm": "SecurePass123!@#",
    "emailVisibility": true
  }')

echo "Response: $CREATE_USER_RESPONSE"
USER_ID=$(echo $CREATE_USER_RESPONSE | grep -o '"id":"[^"]*' | cut -d'"' -f4)
echo -e "${GREEN}User ID: $USER_ID${NC}"
echo ""

# 2. Authenticate user
echo -e "${YELLOW}2. Authenticating user...${NC}"
AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/auth-with-password" \
  -H "Content-Type: application/json" \
  -d '{
    "identity": "testuser@example.com",
    "password": "SecurePass123!@#"
  }')

echo "Response: $AUTH_RESPONSE"
AUTH_TOKEN=$(echo $AUTH_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)
echo -e "${GREEN}Auth Token: ${AUTH_TOKEN:0:50}...${NC}"
echo ""

# 3. Get user profile
echo -e "${YELLOW}3. Getting user profile...${NC}"
curl -s -X GET "${BASE_URL}/api/collections/users/records/${USER_ID}" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" | jq '.'
echo ""

# 4. Update user password (with proper authentication)
echo -e "${YELLOW}4. Updating user password...${NC}"
UPDATE_RESPONSE=$(curl -s -X PATCH "${BASE_URL}/api/collections/users/records/${USER_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{
    "password": "NewSecurePass456!@#",
    "passwordConfirm": "NewSecurePass456!@#",
    "oldPassword": "SecurePass123!@#"
  }')

echo "Response: $UPDATE_RESPONSE"
echo ""

# 5. Try to reuse the same password (should fail due to password reuse prevention)
echo -e "${YELLOW}5. Testing password reuse prevention (should fail)...${NC}"
REUSE_RESPONSE=$(curl -s -X PATCH "${BASE_URL}/api/collections/users/records/${USER_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{
    "password": "NewSecurePass456!@#",
    "passwordConfirm": "NewSecurePass456!@#",
    "oldPassword": "NewSecurePass456!@#"
  }')

echo "Response: $REUSE_RESPONSE"
if echo "$REUSE_RESPONSE" | grep -q "recently used"; then
  echo -e "${GREEN}✓ Password reuse prevention is working!${NC}"
else
  echo -e "${RED}✗ Password reuse prevention not working${NC}"
fi
echo ""

# 6. Test password change frequency limit
echo -e "${YELLOW}6. Testing password change frequency limit...${NC}"
echo "Attempting multiple password changes..."

# Re-authenticate with new password
AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/auth-with-password" \
  -H "Content-Type: application/json" \
  -d '{
    "identity": "testuser@example.com",
    "password": "NewSecurePass456!@#"
  }')
AUTH_TOKEN=$(echo $AUTH_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

for i in {2..6}; do
  echo "  Change $i..."
  curl -s -X PATCH "${BASE_URL}/api/collections/users/records/${USER_ID}" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -d "{
      \"password\": \"TestPass${i}!@#Abc\",
      \"passwordConfirm\": \"TestPass${i}!@#Abc\",
      \"oldPassword\": \"NewSecurePass456!@#\"
    }" > /dev/null 2>&1
  
  if [ $i -eq 6 ]; then
    # The 6th change should fail
    FREQ_RESPONSE=$(curl -s -X PATCH "${BASE_URL}/api/collections/users/records/${USER_ID}" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer ${AUTH_TOKEN}" \
      -d '{
        "password": "TooManyChanges!@#123",
        "passwordConfirm": "TooManyChanges!@#123",
        "oldPassword": "TestPass5!@#Abc"
      }')
    
    if echo "$FREQ_RESPONSE" | grep -q "too many"; then
      echo -e "${GREEN}✓ Password frequency limit is working!${NC}"
    else
      echo -e "${RED}✗ Password frequency limit not working${NC}"
    fi
  fi
done
echo ""

# 7. Test invalid password format
echo -e "${YELLOW}7. Testing invalid password (too short, should fail)...${NC}"
INVALID_RESPONSE=$(curl -s -X PATCH "${BASE_URL}/api/collections/users/records/${USER_ID}" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AUTH_TOKEN}" \
  -d '{
    "password": "short",
    "passwordConfirm": "short",
    "oldPassword": "TestPass5!@#Abc"
  }')

echo "Response: $INVALID_RESPONSE"
if echo "$INVALID_RESPONSE" | grep -qi "validation\|at least 8"; then
  echo -e "${GREEN}✓ Password validation is working!${NC}"
else
  echo -e "${RED}✗ Password validation not working${NC}"
fi
echo ""

# 8. Delete user (admin operation - requires admin token)
echo -e "${YELLOW}8. Deleting user (requires admin privileges)...${NC}"
echo "Note: This requires admin authentication"
echo "Command: curl -X DELETE \"${BASE_URL}/api/collections/users/records/${USER_ID}\" -H \"Authorization: Bearer ADMIN_TOKEN\""
echo ""

echo "================================"
echo "User Management Tests Complete"
echo "================================"
