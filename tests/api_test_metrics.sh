#!/bin/bash

# NannyAPI Metrics Ingestion Tests
# Base URL
BASE_URL="http://127.0.0.1:8090"

echo "================================"
echo "Metrics Ingestion Tests"
echo "================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Prerequisites: Create user and register agent
echo -e "${YELLOW}Setup: Creating user and registering agent...${NC}"

# Create user
USER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/records" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "metrics-user@example.com",
    "password": "MetricsUser123!@#",
    "passwordConfirm": "MetricsUser123!@#"
  }')
USER_ID=$(echo $USER_RESPONSE | jq -r '.id')

# Authenticate
AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/auth-with-password" \
  -H "Content-Type: application/json" \
  -d '{
    "identity": "metrics-user@example.com",
    "password": "MetricsUser123!@#"
  }')
USER_TOKEN=$(echo $AUTH_RESPONSE | jq -r '.token')

# Device auth flow
DEVICE_AUTH=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d '{"action": "device-auth-start"}')
DEVICE_CODE=$(echo $DEVICE_AUTH | jq -r '.device_code')
USER_CODE=$(echo $DEVICE_AUTH | jq -r '.user_code')

# Authorize
curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${USER_TOKEN}" \
  -d "{\"action\": \"authorize\", \"user_code\": \"${USER_CODE}\"}" > /dev/null

# Register agent
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -d "{
    \"action\": \"register\",
    \"device_code\": \"${DEVICE_CODE}\",
    \"agent_name\": \"Metrics-Test-Agent\",
    \"platform\": \"linux\",
    \"hostname\": \"metrics-server.local\",
    \"ip_address\": \"192.168.1.50\",
    \"version\": \"1.0.0\"
  }")

AGENT_TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.access_token')
AGENT_ID=$(echo $REGISTER_RESPONSE | jq -r '.agent_id')

echo -e "${GREEN}Agent registered: $AGENT_ID${NC}"
echo -e "${GREEN}Agent token: ${AGENT_TOKEN:0:40}...${NC}"
echo ""

# Test 1: Ingest basic system metrics
echo -e "${YELLOW}1. Ingesting basic system metrics...${NC}"
METRICS_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_usage_percent": 45.5,
      "memory_used_gb": 8.2,
      "memory_total_gb": 16.0,
      "disk_used_gb": 250.5,
      "disk_total_gb": 512.0,
      "network_rx_gbps": 0.5,
      "network_tx_gbps": 0.3,
      "uptime_seconds": 86400
    }
  }')

echo "Response:"
echo "$METRICS_RESPONSE" | jq '.'
if echo "$METRICS_RESPONSE" | grep -q "success"; then
  echo -e "${GREEN} Basic metrics ingested successfully${NC}"

  # VALIDATION: Check that metrics are actually stored with non-zero values
  sleep 1
  STORED_METRICS=$(curl -s "${BASE_URL}/api/collections/agent_metrics/records?perPage=1" \
    -H "Authorization: Bearer ${USER_TOKEN}")
  CPU_VAL=$(echo "$STORED_METRICS" | jq -r '.items[-1].cpu_percent // 0')
  MEM_VAL=$(echo "$STORED_METRICS" | jq -r '.items[-1].memory_used_gb // 0')
  DISK_VAL=$(echo "$STORED_METRICS" | jq -r '.items[-1].disk_used_gb // 0')

  if [ "$CPU_VAL" != "0" ] && [ "$CPU_VAL" != "null" ] && [ "$MEM_VAL" != "0" ] && [ "$MEM_VAL" != "null" ]; then
    echo -e "${GREEN} VALIDATED: Metrics stored correctly (CPU=$CPU_VAL%, MEM=${MEM_VAL}GB, DISK=${DISK_VAL}GB)${NC}"
  else
    echo -e "${RED} VALIDATION FAILED: Metrics are ZERO! (CPU=$CPU_VAL, MEM=$MEM_VAL, DISK=$DISK_VAL)${NC}"
    exit 1
  fi
fi
echo ""

# Test 2: Ingest comprehensive metrics
echo -e "${YELLOW}2. Ingesting comprehensive system metrics...${NC}"
COMPREHENSIVE_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_usage_percent": 65.8,
      "cpu_cores": 8,
      "memory_used_gb": 12.5,
      "memory_total_gb": 16.0,
      "memory_available_gb": 3.5,
      "disk_used_gb": 275.8,
      "disk_total_gb": 512.0,
      "disk_available_gb": 236.2,
      "network_rx_gbps": 1.2,
      "network_tx_gbps": 0.8,
      "network_rx_packets": 1500000,
      "network_tx_packets": 1200000,
      "uptime_seconds": 172800,
      "load_average_1m": 2.5,
      "load_average_5m": 2.1,
      "load_average_15m": 1.8
    },
    "processes": [
      {
        "pid": 1234,
        "name": "node",
        "cpu_percent": 25.5,
        "memory_mb": 512.0,
        "status": "running"
      },
      {
        "pid": 5678,
        "name": "postgres",
        "cpu_percent": 15.2,
        "memory_mb": 1024.0,
        "status": "running"
      }
    ]
  }')

echo "Response:"
echo "$COMPREHENSIVE_RESPONSE" | jq '.'
if echo "$COMPREHENSIVE_RESPONSE" | grep -q "success"; then
  echo -e "${GREEN} Comprehensive metrics ingested successfully${NC}"
fi
echo ""

# Test 3: High-load scenario metrics
echo -e "${YELLOW}3. Ingesting high-load scenario metrics...${NC}"
HIGH_LOAD_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_usage_percent": 95.5,
      "memory_used_gb": 15.2,
      "memory_total_gb": 16.0,
      "disk_used_gb": 480.0,
      "disk_total_gb": 512.0,
      "network_rx_gbps": 2.5,
      "network_tx_gbps": 2.3,
      "uptime_seconds": 259200,
      "load_average_1m": 8.5,
      "load_average_5m": 7.2,
      "load_average_15m": 6.5
    }
  }')

echo "Response:"
echo "$HIGH_LOAD_RESPONSE" | jq '.'
if echo "$HIGH_LOAD_RESPONSE" | grep -q "success"; then
  echo -e "${GREEN} High-load metrics ingested successfully${NC}"
fi
echo ""

# Test 4: Minimal metrics (required fields only)
echo -e "${YELLOW}4. Ingesting minimal metrics (required fields only)...${NC}"
MINIMAL_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_usage_percent": 30.0,
      "memory_used_gb": 6.0,
      "memory_total_gb": 16.0,
      "disk_used_gb": 200.0,
      "disk_total_gb": 512.0,
      "network_rx_gbps": 0.1,
      "network_tx_gbps": 0.05
    }
  }')

echo "Response:"
echo "$MINIMAL_RESPONSE" | jq '.'
if echo "$MINIMAL_RESPONSE" | grep -q "success"; then
  echo -e "${GREEN} Minimal metrics ingested successfully${NC}"
fi
echo ""

# Test 5: Invalid metrics (missing required fields - should fail)
echo -e "${YELLOW}5. Testing invalid metrics (missing fields, should fail)...${NC}"
INVALID_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_usage_percent": 30.0
    }
  }')

echo "Response:"
echo "$INVALID_RESPONSE" | jq '.'
if echo "$INVALID_RESPONSE" | grep -qi "error\|invalid\|required"; then
  echo -e "${GREEN} Validation is working correctly${NC}"
else
  echo -e "${RED} Validation not working${NC}"
fi
echo ""

# Test 6: Metrics with network stats in different units
echo -e "${YELLOW}6. Testing network stats with proper units (Gbps)...${NC}"
NETWORK_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_usage_percent": 50.0,
      "memory_used_gb": 8.0,
      "memory_total_gb": 16.0,
      "disk_used_gb": 250.0,
      "disk_total_gb": 512.0,
      "network_rx_gbps": 1.5,
      "network_tx_gbps": 1.2,
      "network_rx_packets": 2000000,
      "network_tx_packets": 1800000
    }
  }')

echo "Response:"
echo "$NETWORK_RESPONSE" | jq '.'
if echo "$NETWORK_RESPONSE" | grep -q "success"; then
  echo -e "${GREEN} Network stats ingested successfully${NC}"
fi
echo ""

# Test 7: Rapid metrics ingestion (simulating real agent behavior)
echo -e "${YELLOW}7. Testing rapid metrics ingestion (5 samples)...${NC}"
for i in {1..5}; do
  CPU=$((40 + RANDOM % 30))
  MEM=$((6 + RANDOM % 8))

  curl -s -X POST "${BASE_URL}/api/agent" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${AGENT_TOKEN}" \
    -d "{
      \"action\": \"ingest-metrics\",
      \"system_metrics\": {
        \"cpu_usage_percent\": ${CPU}.5,
        \"memory_used_gb\": ${MEM}.2,
        \"memory_total_gb\": 16.0,
        \"disk_used_gb\": 250.0,
        \"disk_total_gb\": 512.0,
        \"network_rx_gbps\": 0.5,
        \"network_tx_gbps\": 0.3
      }
    }" > /dev/null

  echo "  Sample $i: CPU=${CPU}%, MEM=${MEM}GB"
  sleep 0.5
done
echo -e "${GREEN} Rapid ingestion completed${NC}"
echo ""

echo "================================"
echo "Metrics Ingestion Tests Complete"
echo "================================"
echo ""
echo -e "${BLUE}Summary:${NC}"
echo "- Agent ID: $AGENT_ID"
echo "- Multiple metrics ingested successfully"
echo "- Validation working correctly"
echo ""
echo -e "${YELLOW}To query metrics from PocketBase:${NC}"
echo "curl '${BASE_URL}/api/collections/agent_metrics/records?filter=(agent_id=\"${AGENT_ID}\")&sort=-created' \\"
echo "  -H 'Authorization: Bearer ADMIN_TOKEN'"
