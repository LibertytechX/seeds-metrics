#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

API_URL="http://localhost:8080"

echo "🧪 Testing Analytics API"
echo "========================"
echo ""

# Test 1: Health Check
echo -e "${YELLOW}Test 1: Health Check${NC}"
response=$(curl -s -w "\n%{http_code}" ${API_URL}/health)
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" == "200" ]; then
    echo -e "${GREEN}✅ Health check passed${NC}"
    echo "$body" | jq '.'
else
    echo -e "${RED}❌ Health check failed (HTTP $http_code)${NC}"
    echo "$body"
fi
echo ""

# Test 2: Create Loan
echo -e "${YELLOW}Test 2: Create Loan${NC}"
loan_payload='{
  "loan_id": "LN2024TEST001",
  "customer_id": "CUST20240001",
  "customer_name": "Test Customer",
  "customer_phone": "+234-803-123-4567",
  "officer_id": "OFF2024001",
  "officer_name": "Test Officer",
  "officer_phone": "+234-803-987-6543",
  "region": "South West",
  "branch": "Lagos Main",
  "state": "Lagos",
  "loan_amount": 500000.00,
  "disbursement_date": "2024-10-15",
  "maturity_date": "2025-04-15",
  "loan_term_days": 180,
  "interest_rate": 0.1500,
  "fee_amount": 25000.00,
  "channel": "Direct",
  "status": "Active"
}'

response=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/etl/loans \
  -H "Content-Type: application/json" \
  -d "$loan_payload")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" == "201" ]; then
    echo -e "${GREEN}✅ Loan created successfully${NC}"
    echo "$body" | jq '.'
else
    echo -e "${RED}❌ Loan creation failed (HTTP $http_code)${NC}"
    echo "$body"
fi
echo ""

# Test 3: Create Repayment
echo -e "${YELLOW}Test 3: Create Repayment${NC}"
repayment_payload='{
  "repayment_id": "REP2024TEST001",
  "loan_id": "LN2024TEST001",
  "payment_date": "2024-11-01",
  "payment_amount": 100000.00,
  "principal_paid": 80000.00,
  "interest_paid": 15000.00,
  "fees_paid": 5000.00,
  "penalty_paid": 0.00,
  "payment_method": "Bank Transfer",
  "payment_reference": "TXN20241101TEST",
  "payment_channel": "Mobile App",
  "dpd_at_payment": 0,
  "is_backdated": false,
  "is_reversed": false,
  "waiver_amount": 0.00
}'

response=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/etl/repayments \
  -H "Content-Type: application/json" \
  -d "$repayment_payload")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" == "201" ]; then
    echo -e "${GREEN}✅ Repayment created successfully${NC}"
    echo "$body" | jq '.'
else
    echo -e "${RED}❌ Repayment creation failed (HTTP $http_code)${NC}"
    echo "$body"
fi
echo ""

# Test 4: Batch Sync
echo -e "${YELLOW}Test 4: Batch Sync${NC}"
batch_payload='{
  "sync_timestamp": "2024-10-18T14:30:00Z",
  "sync_type": "incremental",
  "data": {
    "loans": [
      {
        "loan_id": "LN2024TEST002",
        "customer_id": "CUST20240002",
        "customer_name": "Batch Test Customer",
        "officer_id": "OFF2024001",
        "officer_name": "Test Officer",
        "region": "South West",
        "branch": "Lagos Main",
        "loan_amount": 750000.00,
        "disbursement_date": "2024-10-16",
        "maturity_date": "2025-04-16",
        "loan_term_days": 180,
        "interest_rate": 0.1500,
        "fee_amount": 37500.00,
        "channel": "Agent",
        "status": "Active"
      }
    ],
    "repayments": [
      {
        "repayment_id": "REP2024TEST002",
        "loan_id": "LN2024TEST002",
        "payment_date": "2024-11-02",
        "payment_amount": 125000.00,
        "principal_paid": 100000.00,
        "interest_paid": 18750.00,
        "fees_paid": 6250.00,
        "penalty_paid": 0.00,
        "payment_method": "Card Payment",
        "dpd_at_payment": 0,
        "is_backdated": false,
        "is_reversed": false,
        "waiver_amount": 0.00
      }
    ]
  },
  "metadata": {
    "total_loans": 1,
    "total_repayments": 1,
    "source_system": "test_script",
    "etl_version": "1.0.0"
  }
}'

response=$(curl -s -w "\n%{http_code}" -X POST ${API_URL}/api/v1/etl/sync \
  -H "Content-Type: application/json" \
  -d "$batch_payload")
http_code=$(echo "$response" | tail -n1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" == "200" ]; then
    echo -e "${GREEN}✅ Batch sync completed successfully${NC}"
    echo "$body" | jq '.'
else
    echo -e "${RED}❌ Batch sync failed (HTTP $http_code)${NC}"
    echo "$body"
fi
echo ""

echo "========================"
echo -e "${GREEN}🎉 API Testing Complete${NC}"

