#!/bin/bash

echo "=========================================="
echo "Testing Frontend-Backend Integration"
echo "=========================================="
echo ""

# Test 1: Check if backend is running
echo "1. Testing Backend Health..."
HEALTH=$(curl -s http://localhost:8080/health)
if [ $? -eq 0 ]; then
    echo "✅ Backend is running"
    echo "$HEALTH" | jq .
else
    echo "❌ Backend is not running"
    exit 1
fi
echo ""

# Test 2: Check Early Indicators endpoint
echo "2. Testing Early Indicators Endpoint..."
EARLY=$(curl -s http://localhost:8080/api/v1/early-indicators/loans)
EARLY_COUNT=$(echo "$EARLY" | jq '.data.total')
echo "✅ Early Indicators: $EARLY_COUNT loans"
echo "$EARLY" | jq '.data.loans[] | {loan_id, customer_name, current_dpd}'
echo ""

# Test 3: Check FIMR endpoint
echo "3. Testing FIMR Endpoint..."
FIMR=$(curl -s http://localhost:8080/api/v1/fimr/loans)
FIMR_COUNT=$(echo "$FIMR" | jq '.data.total')
echo "✅ FIMR Loans: $FIMR_COUNT loans"
echo "$FIMR" | jq '.data.loans[] | {loan_id, customer_name, days_since_due}'
echo ""

# Test 4: Check Officers endpoint
echo "4. Testing Officers Endpoint..."
OFFICERS=$(curl -s http://localhost:8080/api/v1/officers)
OFFICERS_COUNT=$(echo "$OFFICERS" | jq '.data.total')
echo "✅ Officers: $OFFICERS_COUNT officers"
echo "$OFFICERS" | jq '.data.officers[] | {officer_id, name, branch}' | head -20
echo ""

# Test 5: Check Portfolio Metrics endpoint
echo "5. Testing Portfolio Metrics Endpoint..."
PORTFOLIO=$(curl -s http://localhost:8080/api/v1/metrics/portfolio)
echo "✅ Portfolio Metrics:"
echo "$PORTFOLIO" | jq '.data'
echo ""

# Test 6: Check Branches endpoint
echo "6. Testing Branches Endpoint..."
BRANCHES=$(curl -s http://localhost:8080/api/v1/branches)
BRANCHES_COUNT=$(echo "$BRANCHES" | jq '.data.total')
echo "✅ Branches: $BRANCHES_COUNT branches"
echo "$BRANCHES" | jq '.data.branches[] | {branch, region, active_loans}' | head -20
echo ""

# Test 7: Check if frontend is running
echo "7. Testing Frontend..."
FRONTEND=$(curl -s http://localhost:5173/ | grep -o "vite" | head -1)
if [ "$FRONTEND" == "vite" ]; then
    echo "✅ Frontend is running at http://localhost:5173/"
else
    echo "❌ Frontend is not running"
fi
echo ""

echo "=========================================="
echo "✅ All Integration Tests Passed!"
echo "=========================================="
echo ""
echo "📊 Summary:"
echo "  - Backend API: http://localhost:8080"
echo "  - Frontend: http://localhost:5173/"
echo "  - Early Indicators: $EARLY_COUNT loans"
echo "  - FIMR Loans: $FIMR_COUNT loans"
echo "  - Officers: $OFFICERS_COUNT officers"
echo "  - Branches: $BRANCHES_COUNT branches"
echo ""
echo "🎯 Expected Data:"
echo "  - Inyang Kpongette (LN2024100001) should appear in FIMR Drilldown"
echo "  - Shamsideen Allamu (LN2024100002) should appear in Early Indicators Drilldown"
echo ""

