#!/bin/bash

# Test script for Credit Health Overview endpoint with new filters
# This script tests the /api/v1/branches endpoint with various filter combinations

BASE_URL="http://localhost:8080/api/v1"

echo "=========================================="
echo "Testing Credit Health Overview Endpoint"
echo "=========================================="
echo ""

# Test 1: Get all branches (no filters)
echo "Test 1: Get all branches (no filters)"
curl -s "${BASE_URL}/branches" | jq '.' | head -50
echo ""
echo "---"
echo ""

# Test 2: Filter by region
echo "Test 2: Filter by region"
curl -s "${BASE_URL}/branches?region=Lagos" | jq '.' | head -50
echo ""
echo "---"
echo ""

# Test 3: Filter by branch
echo "Test 3: Filter by branch"
curl -s "${BASE_URL}/branches?branch=Lekki" | jq '.' | head -50
echo ""
echo "---"
echo ""

# Test 4: Filter by channel
echo "Test 4: Filter by channel"
curl -s "${BASE_URL}/branches?channel=AGENT" | jq '.' | head -50
echo ""
echo "---"
echo ""

# Test 5: Filter by user_type
echo "Test 5: Filter by user_type"
curl -s "${BASE_URL}/branches?user_type=AGENT" | jq '.' | head -50
echo ""
echo "---"
echo ""

# Test 6: Filter by wave
echo "Test 6: Filter by wave"
curl -s "${BASE_URL}/branches?wave=Wave1" | jq '.' | head -50
echo ""
echo "---"
echo ""

# Test 7: Multiple filters combined
echo "Test 7: Multiple filters combined (region + channel)"
curl -s "${BASE_URL}/branches?region=Lagos&channel=AGENT" | jq '.' | head -50
echo ""
echo "---"
echo ""

# Test 8: Multiple filters with sorting
echo "Test 8: Multiple filters with sorting (region + sort_by + sort_dir)"
curl -s "${BASE_URL}/branches?region=Lagos&sort_by=portfolio_total&sort_dir=desc" | jq '.' | head -50
echo ""
echo "---"
echo ""

# Test 9: All filters combined
echo "Test 9: All filters combined"
curl -s "${BASE_URL}/branches?region=Lagos&branch=Lekki&channel=AGENT&user_type=AGENT&wave=Wave1&sort_by=portfolio_total&sort_dir=desc" | jq '.' | head -50
echo ""
echo "---"
echo ""

echo "=========================================="
echo "Testing Complete"
echo "=========================================="

