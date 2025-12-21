#!/bin/bash

# NannyAPI Complete Test Suite
# This script runs all API tests in sequence

echo "================================"
echo "NannyAPI Complete Test Suite"
echo "================================"
echo ""
echo "Starting comprehensive API tests..."
echo ""

# Check if server is running
if ! curl -s http://127.0.0.1:8090/api/health > /dev/null 2>&1; then
  echo "❌ Server is not running on port 8090"
  echo "Please start the server with: ./bin/nannyapi serve"
  exit 1
fi

echo "✓ Server is running"
echo ""

# Make scripts executable
chmod +x tests/api_test_users.sh
chmod +x tests/api_test_agent_auth.sh
chmod +x tests/api_test_metrics.sh

# Run each test suite
echo "============================================"
echo "Running User Management Tests..."
echo "============================================"
./tests/api_test_users.sh
echo ""
echo ""

echo "============================================"
echo "Running Agent Authorization Tests..."
echo "============================================"
./tests/api_test_agent_auth.sh
echo ""
echo ""

echo "============================================"
echo "Running Metrics Ingestion Tests..."
echo "============================================"
./tests/api_test_metrics.sh
echo ""
echo ""

echo "================================"
echo "All Tests Complete!"
echo "================================"
echo ""
echo "Test suites executed:"
echo "  1. User Management (create, update, delete, password security)"
echo "  2. Agent Authorization (device auth flow, register, revoke)"
echo "  3. Metrics Ingestion (system metrics, validation)"
echo ""
echo "You can run individual test suites:"
echo "  ./tests/api_test_users.sh"
echo "  ./tests/api_test_agent_auth.sh"
echo "  ./tests/api_test_metrics.sh"
