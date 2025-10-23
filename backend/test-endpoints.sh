#!/bin/bash

echo "🧪 Testing All Dashboard API Endpoints"
echo "========================================"
echo ""

BASE_URL="http://localhost:8080"

# Test 1: Health Check
echo "1️⃣  Testing Health Check..."
curl -s "$BASE_URL/health" | jq -r '.status' && echo "✅ Health check passed" || echo "❌ Health check failed"
echo ""

# Test 2: Portfolio Metrics
echo "2️⃣  Testing Portfolio Metrics..."
curl -s "$BASE_URL/api/v1/metrics/portfolio" | jq -r '.status' && echo "✅ Portfolio metrics passed" || echo "❌ Portfolio metrics failed"
echo ""

# Test 3: Officers List
echo "3️⃣  Testing Officers List..."
curl -s "$BASE_URL/api/v1/officers" | jq -r '.status' && echo "✅ Officers list passed" || echo "❌ Officers list failed"
echo ""

# Test 4: Officer Detail
echo "4️⃣  Testing Officer Detail..."
curl -s "$BASE_URL/api/v1/officers/OFF2024012" | jq -r '.status' && echo "✅ Officer detail passed" || echo "❌ Officer detail failed"
echo ""

# Test 5: FIMR Loans
echo "5️⃣  Testing FIMR Loans..."
curl -s "$BASE_URL/api/v1/fimr/loans" | jq -r '.status' && echo "✅ FIMR loans passed" || echo "❌ FIMR loans failed"
echo ""

# Test 6: Early Indicator Loans
echo "6️⃣  Testing Early Indicator Loans..."
curl -s "$BASE_URL/api/v1/early-indicators/loans" | jq -r '.status' && echo "✅ Early indicator loans passed" || echo "❌ Early indicator loans failed"
echo ""

# Test 7: Branches
echo "7️⃣  Testing Branches..."
curl -s "$BASE_URL/api/v1/branches" | jq -r '.status' && echo "✅ Branches passed" || echo "❌ Branches failed"
echo ""

# Test 8: Team Members
echo "8️⃣  Testing Team Members..."
curl -s "$BASE_URL/api/v1/team-members" | jq -r '.status' && echo "✅ Team members passed" || echo "❌ Team members failed"
echo ""

echo "========================================"
echo "✅ All endpoint tests completed!"

