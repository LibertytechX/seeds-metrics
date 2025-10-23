#!/bin/bash

# FIMR Demo Test Script
# This script creates realistic loan and repayment data to demonstrate FIMR tracking

set -e

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"

echo "🧪 FIMR Demo Test Data Generator"
echo "=================================="
echo ""

# Calculate dates
TODAY=$(date +%Y-%m-%d)
TWO_MONTHS_AGO=$(date -v-60d +%Y-%m-%d 2>/dev/null || date -d "60 days ago" +%Y-%m-%d)
FIVE_WEEKS_AGO=$(date -v-35d +%Y-%m-%d 2>/dev/null || date -d "35 days ago" +%Y-%m-%d)
THREE_MONTHS_FUTURE=$(date -v+90d +%Y-%m-%d 2>/dev/null || date -d "90 days" +%Y-%m-%d)

echo "📅 Date Calculations:"
echo "   Today: $TODAY"
echo "   Two months ago: $TWO_MONTHS_AGO"
echo "   Five weeks ago: $FIVE_WEEKS_AGO"
echo "   Three months future: $THREE_MONTHS_FUTURE"
echo ""

# Check if backend is running
echo "🔍 Checking backend health..."
if ! curl -s "$BASE_URL/health" > /dev/null; then
    echo "❌ Backend is not running! Please start it with: cd backend && docker-compose up -d"
    exit 1
fi
echo "✅ Backend is healthy"
echo ""

# Create customers first (required for foreign key constraints)
echo "👥 Creating Customers..."
echo "========================"

# Customer 1
docker exec -i analytics-postgres psql -U analytics_user -d analytics_db <<EOF
INSERT INTO customers (customer_id, customer_name, customer_phone, state, kyc_status, created_at)
VALUES ('CUST2024100001', 'Inyang Kpongette', '+234-801-234-5678', 'Lagos', 'Verified', CURRENT_TIMESTAMP)
ON CONFLICT (customer_id) DO NOTHING;
EOF
echo "✅ Customer 1 created: Inyang Kpongette"

# Customer 2
docker exec -i analytics-postgres psql -U analytics_user -d analytics_db <<EOF
INSERT INTO customers (customer_id, customer_name, customer_phone, state, kyc_status, created_at)
VALUES ('CUST2024100002', 'Shamsideen Allamu', '+234-803-456-7890', 'Lagos', 'Verified', CURRENT_TIMESTAMP)
ON CONFLICT (customer_id) DO NOTHING;
EOF
echo "✅ Customer 2 created: Shamsideen Allamu"
echo ""

# Create Loan 1 - Inyang Kpongette (FIMR case)
echo "📝 Creating Loan 1 - Inyang Kpongette (FIMR Case)"
echo "=================================================="

LOAN1_ID="LN2024100001"
CUSTOMER1_ID="CUST2024100001"
OFFICER1_ID="OFF2024012"  # Using existing officer

LOAN1_AMOUNT=1000000
LOAN1_INTEREST_RATE=0.10
LOAN1_DURATION_DAYS=90

# Calculate total interest and fees
LOAN1_TOTAL_INTEREST=$(echo "$LOAN1_AMOUNT * $LOAN1_INTEREST_RATE * 3" | bc)  # 3 months
LOAN1_TOTAL_FEES=$(echo "$LOAN1_AMOUNT * 0.05" | bc)  # 5% processing fee
LOAN1_TOTAL_AMOUNT=$(echo "$LOAN1_AMOUNT + $LOAN1_TOTAL_INTEREST + $LOAN1_TOTAL_FEES" | bc)

# Daily repayment amount (principal + interest + fees / 90 days)
LOAN1_DAILY_PRINCIPAL=$(echo "scale=2; $LOAN1_AMOUNT / $LOAN1_DURATION_DAYS" | bc)
LOAN1_DAILY_INTEREST=$(echo "scale=2; $LOAN1_TOTAL_INTEREST / $LOAN1_DURATION_DAYS" | bc)
LOAN1_DAILY_FEES=$(echo "scale=2; $LOAN1_TOTAL_FEES / $LOAN1_DURATION_DAYS" | bc)
LOAN1_DAILY_TOTAL=$(echo "scale=2; $LOAN1_DAILY_PRINCIPAL + $LOAN1_DAILY_INTEREST + $LOAN1_DAILY_FEES" | bc)

echo "   Loan ID: $LOAN1_ID"
echo "   Customer: Inyang Kpongette ($CUSTOMER1_ID)"
echo "   Officer: $OFFICER1_ID"
echo "   Amount: ₦$(printf "%'d" $LOAN1_AMOUNT)"
echo "   Interest Rate: 10% per month"
echo "   Duration: 90 days"
echo "   Total Interest: ₦$(printf "%.2f" $LOAN1_TOTAL_INTEREST)"
echo "   Total Fees: ₦$(printf "%.2f" $LOAN1_TOTAL_FEES)"
echo "   Daily Repayment: ₦$(printf "%.2f" $LOAN1_DAILY_TOTAL)"
echo ""

# Create loan 1
cat > /tmp/loan1.json <<EOF
{
  "loan_id": "$LOAN1_ID",
  "customer_id": "$CUSTOMER1_ID",
  "customer_name": "Inyang Kpongette",
  "customer_phone": "+234-801-234-5678",
  "officer_id": "$OFFICER1_ID",
  "officer_name": "Sarah Johnson",
  "officer_phone": "+234-803-987-6543",
  "region": "South West",
  "branch": "Lagos Main",
  "state": "Lagos",
  "loan_amount": $LOAN1_AMOUNT,
  "disbursement_date": "$TWO_MONTHS_AGO",
  "maturity_date": "$THREE_MONTHS_FUTURE",
  "loan_term_days": $LOAN1_DURATION_DAYS,
  "interest_rate": $LOAN1_INTEREST_RATE,
  "fee_amount": $LOAN1_TOTAL_FEES,
  "channel": "Direct",
  "channel_partner": null,
  "status": "Active",
  "closed_date": null
}
EOF

echo "📤 Posting Loan 1..."
curl -s -X POST "$API_URL/etl/loans" \
  -H "Content-Type: application/json" \
  -d @/tmp/loan1.json | jq -r '.status' > /dev/null && echo "✅ Loan 1 created" || echo "❌ Failed to create Loan 1"

# Create loan schedule for Loan 1 (daily repayments for 90 days)
echo "📅 Creating loan schedule for Loan 1..."
for day in {1..90}; do
    DAYS_OFFSET=$((60-day))
    if [ $DAYS_OFFSET -ge 0 ]; then
        DUE_DATE=$(date -v-${DAYS_OFFSET}d +%Y-%m-%d 2>/dev/null || date -d "${DAYS_OFFSET} days ago" +%Y-%m-%d)
    else
        DAYS_FORWARD=$((-DAYS_OFFSET))
        DUE_DATE=$(date -v+${DAYS_FORWARD}d +%Y-%m-%d 2>/dev/null || date -d "${DAYS_FORWARD} days" +%Y-%m-%d)
    fi
    PAYMENT_STATUS="Pending"

    # Mark as paid if day 11-20 (we have payments for these days)
    if [ $day -ge 11 ] && [ $day -le 20 ]; then
        PAYMENT_STATUS="Paid"
    fi

    docker exec -i analytics-postgres psql -U analytics_user -d analytics_db <<EOF > /dev/null
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status, created_at)
VALUES ('$LOAN1_ID', $day, '$DUE_DATE', $LOAN1_DAILY_PRINCIPAL, $LOAN1_DAILY_INTEREST, $LOAN1_DAILY_FEES, $LOAN1_DAILY_TOTAL, '$PAYMENT_STATUS', CURRENT_TIMESTAMP)
ON CONFLICT (loan_id, installment_number) DO NOTHING;
EOF
done
echo "✅ Loan schedule created (90 daily installments)"
echo ""

# Create repayments for Loan 1 - MISSED first 10 days, then paid days 11-20
echo "💰 Creating Repayments for Loan 1 (Days 11-20, MISSED Days 1-10)"
echo "================================================================"

for day in {11..20}; do
    PAYMENT_DATE=$(date -v-$((60-day))d +%Y-%m-%d 2>/dev/null || date -d "$((60-day)) days ago" +%Y-%m-%d)
    REPAYMENT_ID="REP2024$(printf "%06d" $((100000 + day)))"

    cat > /tmp/repayment_loan1_day${day}.json <<EOF
{
  "repayment_id": "$REPAYMENT_ID",
  "loan_id": "$LOAN1_ID",
  "payment_date": "$PAYMENT_DATE",
  "payment_amount": $LOAN1_DAILY_TOTAL,
  "principal_paid": $LOAN1_DAILY_PRINCIPAL,
  "interest_paid": $LOAN1_DAILY_INTEREST,
  "fees_paid": $LOAN1_DAILY_FEES,
  "penalty_paid": 0,
  "payment_method": "Bank Transfer",
  "is_backdated": false,
  "is_reversed": false,
  "waiver_amount": 0
}
EOF

    curl -s -X POST "$API_URL/etl/repayments" \
      -H "Content-Type: application/json" \
      -d @/tmp/repayment_loan1_day${day}.json > /dev/null

    echo "   ✅ Day $day payment posted (₦$(printf "%.2f" $LOAN1_DAILY_TOTAL)) - Date: $PAYMENT_DATE"
done
echo ""

# Create Loan 2 - Shamsideen Allamu (Good payer, NO FIMR)
echo "📝 Creating Loan 2 - Shamsideen Allamu (Good Payer, NO FIMR)"
echo "============================================================="

LOAN2_ID="LN2024100002"
CUSTOMER2_ID="CUST2024100002"
OFFICER2_ID="OFF2024012"  # Same officer for comparison

LOAN2_AMOUNT=2000000
LOAN2_INTEREST_RATE=0.10
LOAN2_DURATION_DAYS=90

# Calculate total interest and fees
LOAN2_TOTAL_INTEREST=$(echo "$LOAN2_AMOUNT * $LOAN2_INTEREST_RATE * 3" | bc)
LOAN2_TOTAL_FEES=$(echo "$LOAN2_AMOUNT * 0.05" | bc)
LOAN2_TOTAL_AMOUNT=$(echo "$LOAN2_AMOUNT + $LOAN2_TOTAL_INTEREST + $LOAN2_TOTAL_FEES" | bc)

# Daily repayment amount
LOAN2_DAILY_PRINCIPAL=$(echo "scale=2; $LOAN2_AMOUNT / $LOAN2_DURATION_DAYS" | bc)
LOAN2_DAILY_INTEREST=$(echo "scale=2; $LOAN2_TOTAL_INTEREST / $LOAN2_DURATION_DAYS" | bc)
LOAN2_DAILY_FEES=$(echo "scale=2; $LOAN2_TOTAL_FEES / $LOAN2_DURATION_DAYS" | bc)
LOAN2_DAILY_TOTAL=$(echo "scale=2; $LOAN2_DAILY_PRINCIPAL + $LOAN2_DAILY_INTEREST + $LOAN2_DAILY_FEES" | bc)

echo "   Loan ID: $LOAN2_ID"
echo "   Customer: Shamsideen Allamu ($CUSTOMER2_ID)"
echo "   Officer: $OFFICER2_ID"
echo "   Amount: ₦$(printf "%'d" $LOAN2_AMOUNT)"
echo "   Interest Rate: 10% per month"
echo "   Duration: 90 days"
echo "   Total Interest: ₦$(printf "%.2f" $LOAN2_TOTAL_INTEREST)"
echo "   Total Fees: ₦$(printf "%.2f" $LOAN2_TOTAL_FEES)"
echo "   Daily Repayment: ₦$(printf "%.2f" $LOAN2_DAILY_TOTAL)"
echo ""

# Create loan 2
cat > /tmp/loan2.json <<EOF
{
  "loan_id": "$LOAN2_ID",
  "customer_id": "$CUSTOMER2_ID",
  "customer_name": "Shamsideen Allamu",
  "customer_phone": "+234-803-456-7890",
  "officer_id": "$OFFICER2_ID",
  "officer_name": "Sarah Johnson",
  "officer_phone": "+234-803-987-6543",
  "region": "South West",
  "branch": "Lagos Main",
  "state": "Lagos",
  "loan_amount": $LOAN2_AMOUNT,
  "disbursement_date": "$FIVE_WEEKS_AGO",
  "maturity_date": "$THREE_MONTHS_FUTURE",
  "loan_term_days": $LOAN2_DURATION_DAYS,
  "interest_rate": $LOAN2_INTEREST_RATE,
  "fee_amount": $LOAN2_TOTAL_FEES,
  "channel": "Direct",
  "channel_partner": null,
  "status": "Active",
  "closed_date": null
}
EOF

echo "📤 Posting Loan 2..."
curl -s -X POST "$API_URL/etl/loans" \
  -H "Content-Type: application/json" \
  -d @/tmp/loan2.json | jq -r '.status' > /dev/null && echo "✅ Loan 2 created" || echo "❌ Failed to create Loan 2"

# Create loan schedule for Loan 2 (daily repayments for 90 days)
echo "📅 Creating loan schedule for Loan 2..."
for day in {1..90}; do
    DAYS_OFFSET=$((35-day))
    if [ $DAYS_OFFSET -ge 0 ]; then
        DUE_DATE=$(date -v-${DAYS_OFFSET}d +%Y-%m-%d 2>/dev/null || date -d "${DAYS_OFFSET} days ago" +%Y-%m-%d)
    else
        DAYS_FORWARD=$((-DAYS_OFFSET))
        DUE_DATE=$(date -v+${DAYS_FORWARD}d +%Y-%m-%d 2>/dev/null || date -d "${DAYS_FORWARD} days" +%Y-%m-%d)
    fi
    PAYMENT_STATUS="Pending"

    # Mark as paid if day 1-20 (we have payments for these days)
    if [ $day -ge 1 ] && [ $day -le 20 ]; then
        PAYMENT_STATUS="Paid"
    fi

    docker exec -i analytics-postgres psql -U analytics_user -d analytics_db <<EOF > /dev/null
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status, created_at)
VALUES ('$LOAN2_ID', $day, '$DUE_DATE', $LOAN2_DAILY_PRINCIPAL, $LOAN2_DAILY_INTEREST, $LOAN2_DAILY_FEES, $LOAN2_DAILY_TOTAL, '$PAYMENT_STATUS', CURRENT_TIMESTAMP)
ON CONFLICT (loan_id, installment_number) DO NOTHING;
EOF
done
echo "✅ Loan schedule created (90 daily installments)"
echo ""

# Create repayments for Loan 2 - Paid on time from day 1 for 20 days
echo "💰 Creating Repayments for Loan 2 (Days 1-20, ALL ON TIME)"
echo "==========================================================="

for day in {1..20}; do
    PAYMENT_DATE=$(date -v-$((35-day))d +%Y-%m-%d 2>/dev/null || date -d "$((35-day)) days ago" +%Y-%m-%d)
    REPAYMENT_ID="REP2024$(printf "%06d" $((200000 + day)))"

    cat > /tmp/repayment_loan2_day${day}.json <<EOF
{
  "repayment_id": "$REPAYMENT_ID",
  "loan_id": "$LOAN2_ID",
  "payment_date": "$PAYMENT_DATE",
  "payment_amount": $LOAN2_DAILY_TOTAL,
  "principal_paid": $LOAN2_DAILY_PRINCIPAL,
  "interest_paid": $LOAN2_DAILY_INTEREST,
  "fees_paid": $LOAN2_DAILY_FEES,
  "penalty_paid": 0,
  "payment_method": "Cash",
  "is_backdated": false,
  "is_reversed": false,
  "waiver_amount": 0
}
EOF

    curl -s -X POST "$API_URL/etl/repayments" \
      -H "Content-Type: application/json" \
      -d @/tmp/repayment_loan2_day${day}.json > /dev/null

    echo "   ✅ Day $day payment posted (₦$(printf "%.2f" $LOAN2_DAILY_TOTAL)) - Date: $PAYMENT_DATE"
done
echo ""

# Clean up temp files
rm -f /tmp/loan*.json /tmp/repayment*.json

# Manually trigger the update_loan_metrics function for both loans
echo "🔄 Triggering metric calculations..."
docker exec -i analytics-postgres psql -U analytics_user -d analytics_db <<EOF > /dev/null
SELECT update_loan_metrics('$LOAN1_ID');
SELECT update_loan_metrics('$LOAN2_ID');
EOF
echo "✅ Metrics calculated"
echo ""

echo "🎉 Test data created successfully!"
echo ""
echo "=================================="
echo "📊 VERIFICATION"
echo "=================================="
echo ""

# Wait for triggers to process
echo "⏳ Waiting 2 seconds for database triggers to process..."
sleep 2
echo ""

# Test FIMR loans endpoint
echo "1️⃣  Testing FIMR Loans Endpoint"
echo "   GET $API_URL/fimr/loans"
echo "   --------------------------------"
FIMR_RESULT=$(curl -s "$API_URL/fimr/loans")
FIMR_COUNT=$(echo "$FIMR_RESULT" | jq -r '.data.total')
echo "$FIMR_RESULT" | jq '.data.loans[] | {loan_id, customer_name, officer_id, current_dpd, first_payment_missed}'
echo "   Total FIMR Loans: $FIMR_COUNT"
echo "   Expected: 1 (Inyang Kpongette's loan)"
echo ""

# Test early indicators endpoint
echo "2️⃣  Testing Early Indicators Endpoint"
echo "   GET $API_URL/early-indicators/loans"
echo "   --------------------------------"
EARLY_RESULT=$(curl -s "$API_URL/early-indicators/loans")
EARLY_COUNT=$(echo "$EARLY_RESULT" | jq -r '.data.total')
echo "$EARLY_RESULT" | jq '.data.loans[] | {loan_id, customer_name, current_dpd, principal_outstanding}'
echo "   Total Early Indicator Loans: $EARLY_COUNT"
echo ""

# Test officers endpoint
echo "3️⃣  Testing Officers Metrics"
echo "   GET $API_URL/officers"
echo "   --------------------------------"
curl -s "$API_URL/officers" | jq '.data.officers[] | {officer_id, name, calculatedMetrics: {fimr, ayr, dqi, riskScore}}'
echo ""

# Test portfolio metrics
echo "4️⃣  Testing Portfolio Metrics"
echo "   GET $API_URL/metrics/portfolio"
echo "   --------------------------------"
curl -s "$API_URL/metrics/portfolio" | jq '.data | {totalLoans, totalPortfolio, avgDQI, avgAYR, watchlistCount}'
echo ""

echo "=================================="
echo "✅ FIMR Demo Test Complete!"
echo "=================================="
echo ""
echo "📋 Summary:"
echo "   • Loan 1 (Inyang): ₦1,000,000 - MISSED first 10 days (FIMR)"
echo "   • Loan 2 (Shamsideen): ₦2,000,000 - Paid on time (NO FIMR)"
echo "   • Both loans assigned to officer: $OFFICER1_ID"
echo ""
echo "🔍 Next Steps:"
echo "   • Check FIMR dashboard at: http://localhost:5173/"
echo "   • Query specific loan: curl $API_URL/officers/$OFFICER1_ID | jq ."
echo "   • View database: docker exec -it analytics-postgres psql -U analytics_user -d analytics_db"
echo ""

