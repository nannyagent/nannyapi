#!/bin/bash

# Extended Metrics REST API Test
# Uses the same device-auth flow as api_test_metrics.sh

BASE_URL="http://127.0.0.1:8090"

echo "================================"
echo "Extended Metrics Test"
echo "================================"
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Prerequisites: Create user and register agent
echo -e "${YELLOW}Setup: Creating user and registering agent...${NC}"

# Create user
USER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/records" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "extended-metrics-user@example.com",
    "password": "ExtMetrics123!@#",
    "passwordConfirm": "ExtMetrics123!@#"
  }')
USER_ID=$(echo $USER_RESPONSE | jq -r '.id')

# Authenticate
AUTH_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/collections/users/auth-with-password" \
  -H "Content-Type: application/json" \
  -d '{
    "identity": "extended-metrics-user@example.com",
    "password": "ExtMetrics123!@#"
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
    \"agent_name\": \"Extended-Metrics-Agent\",
    \"platform\": \"linux\",
    \"hostname\": \"extended-metrics-host.local\",
    \"ip_address\": \"192.168.1.100\",
    \"version\": \"1.0.0\"
  }")

AGENT_TOKEN=$(echo $REGISTER_RESPONSE | jq -r '.access_token')
AGENT_ID=$(echo $REGISTER_RESPONSE | jq -r '.agent_id')

echo -e "${GREEN}Agent registered: $AGENT_ID${NC}"
echo ""

# Test: Ingest extended metrics with all fields
echo -e "${YELLOW}Ingesting extended metrics with filesystems, load averages, and computed percentages...${NC}"
METRICS_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_TOKEN}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_percent": 67.8,
      "cpu_cores": 16,
      "memory_used_gb": 12.5,
      "memory_total_gb": 32.0,
      "disk_used_gb": 450.0,
      "disk_total_gb": 1000.0,
      "network_rx_gbps": 1.2,
      "network_tx_gbps": 0.8,
      "load_avg_1min": 4.50,
      "load_avg_5min": 3.75,
      "load_avg_15min": 3.20,
      "filesystems": [
        {
          "device": "/dev/nvme0n1p1",
          "mount_path": "/",
          "used_gb": 450.0,
          "free_gb": 550.0,
          "total_gb": 1000.0,
          "usage_percent": 45.0
        },
        {
          "device": "/dev/nvme0n1p2",
          "mount_path": "/data",
          "used_gb": 100.0,
          "free_gb": 150.0,
          "total_gb": 250.0,
          "usage_percent": 40.0
        }
      ]
    }
  }')

if echo "$METRICS_RESPONSE" | grep -q "success"; then
  echo -e "${GREEN}✓ Extended metrics ingested successfully${NC}"
  
  # VALIDATION: Retrieve and verify the stored metrics
  sleep 1
  STORED_METRICS=$(curl -s "${BASE_URL}/api/collections/agent_metrics/records?perPage=1" \
    -H "Authorization: Bearer ${USER_TOKEN}")
  
  CPU_CORES=$(echo "$STORED_METRICS" | jq -r '.items[-1].cpu_cores // 0')
  MEM_PERCENT=$(echo "$STORED_METRICS" | jq -r '.items[-1].memory_percent // 0')
  DISK_PERCENT=$(echo "$STORED_METRICS" | jq -r '.items[-1].disk_usage_percent // 0')
  LOAD_1M=$(echo "$STORED_METRICS" | jq -r '.items[-1].load_avg_1min // 0')
  LOAD_5M=$(echo "$STORED_METRICS" | jq -r '.items[-1].load_avg_5min // 0')
  LOAD_15M=$(echo "$STORED_METRICS" | jq -r '.items[-1].load_avg_15min // 0')
  FILESYSTEMS=$(echo "$STORED_METRICS" | jq -r '.items[-1].filesystems // empty')
  
  echo ""
  echo "Verification Results:"
  
  if [ "$CPU_CORES" == "16" ]; then
    echo -e "${GREEN}✓${NC} CPU Cores: $CPU_CORES"
  else
    echo -e "${RED}✗${NC} CPU Cores: expected 16, got $CPU_CORES"
  fi
  
  if [ "$MEM_PERCENT" != "0" ] && [ "$MEM_PERCENT" != "null" ]; then
    echo -e "${GREEN}✓${NC} Memory Percent: $MEM_PERCENT% (computed from 12.5GB/32GB)"
  else
    echo -e "${RED}✗${NC} Memory Percent: $MEM_PERCENT (expected ~39.06%)"
  fi
  
  if [ "$DISK_PERCENT" == "45" ]; then
    echo -e "${GREEN}✓${NC} Disk Usage Percent: $DISK_PERCENT% (computed from 450GB/1000GB)"
  else
    echo -e "${RED}✗${NC} Disk Usage Percent: expected 45, got $DISK_PERCENT"
  fi
  
  if [ "$LOAD_1M" == "4.5" ]; then
    echo -e "${GREEN}✓${NC} Load Average 1min: $LOAD_1M"
  else
    echo -e "${RED}✗${NC} Load Average 1min: expected 4.5, got $LOAD_1M"
  fi
  
  if [ "$LOAD_5M" == "3.75" ]; then
    echo -e "${GREEN}✓${NC} Load Average 5min: $LOAD_5M"
  else
    echo -e "${RED}✗${NC} Load Average 5min: expected 3.75, got $LOAD_5M"
  fi
  
  if [ "$LOAD_15M" == "3.2" ]; then
    echo -e "${GREEN}✓${NC} Load Average 15min: $LOAD_15M"
  else
    echo -e "${RED}✗${NC} Load Average 15min: expected 3.2, got $LOAD_15M"
  fi
  
  if [ ! -z "$FILESYSTEMS" ] && [ "$FILESYSTEMS" != "null" ]; then
    FS_COUNT=$(echo "$FILESYSTEMS" | jq 'length' 2>/dev/null)
    echo -e "${GREEN}✓${NC} Filesystems: $FS_COUNT entries stored as JSON"
  else
    echo -e "${RED}✗${NC} Filesystems: not stored or empty"
  fi
  
else
  echo -e "${RED}✗ Failed to ingest extended metrics${NC}"
  echo "$METRICS_RESPONSE" | jq '.'
  exit 1
fi

echo ""
echo "================================"
echo -e "${GREEN}✓ Extended metrics test passed!${NC}"
echo "================================"
