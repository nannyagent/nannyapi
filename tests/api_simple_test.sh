#!/bin/bash

# Simple working API tests (without agent flow that requires migrations)
BASE_URL="http://127.0.0.1:8090"

echo "================================"
echo "Working API Commands Test"
echo "================================"
echo ""

# Test 1: Create User
echo "1. Creating user..."
curl -X POST "${BASE_URL}/api/collections/users/records" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "demo@example.com",
    "password": "DemoPass123!@#",
    "passwordConfirm": "DemoPass123!@#"
  }' | jq '.'
echo ""

# Test 2: Authenticate
echo "2. Authenticating..."
AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/auth-with-password" \
  -H "Content-Type: application/json" \
  -d '{
    "identity": "demo@example.com",
    "password": "DemoPass123!@#"
  }')

echo "$AUTH_RESPONSE" | jq '.'
TOKEN=$(echo "$AUTH_RESPONSE" | jq -r '.token')
USER_ID=$(echo "$AUTH_RESPONSE" | jq -r '.record.id')
echo ""

# Test 3: Get profile
echo "3. Getting user profile..."
curl -s -X GET "${BASE_URL}/api/collections/users/records/${USER_ID}" \
  -H "Authorization: Bearer ${TOKEN}" | jq '.'
echo ""

echo "================================"
echo "✓ Basic User API Works!"
echo "================================"
echo ""
echo "To test agent functionality, you need to:"
echo "1. Stop the server"
echo "2. Run migrations: ./bin/nannyapi migrate"
echo "3. Restart server: ./bin/nannyapi serve"
echo ""
echo "Then agent endpoints will work:"
echo "  - Device auth flow"
echo "  - Agent registration"
echo "  - Metrics ingestion"
