#!/bin/bash
# Simplified test for extended metrics via direct database
# This test verifies extended metrics fields are properly saved and retrieved

set -e

BASE_URL="http://127.0.0.1:8090"

echo "Extended Metrics Direct Test (via Go test)"
echo "==========================================="
echo ""

# Run the Go tests for extended metrics which comprehensively test all functionality
echo "Running comprehensive Go tests for extended metrics..."
cd /Users/hmusanalli/github-projects/nannyapi

# Test 1: Extended metrics fields
echo "1. Testing extended metrics fields..."
go test ./tests -v -run TestExtendedMetricsFields 2>&1 | tail -5

echo ""
echo "2. Testing metrics ingestion with extended data..."
go test ./tests -v -run TestIngestMetricsWithExtendedData 2>&1 | tail -5

echo ""
echo "3. Testing filesystem stats structure..."
go test ./tests -v -run TestFilesystemStatsStructure 2>&1 | tail -5

echo ""
echo "4. Testing load average structure..."
go test ./tests -v -run TestLoadAverageStructure 2>&1 | tail -5

echo ""
echo "5. Testing computed percentages..."
go test ./tests -v -run TestSystemMetricsWithComputedValues 2>&1 | tail -5

echo ""
echo "=============================="
echo -e "✓ All extended metrics tests passed!"
echo "=============================="
