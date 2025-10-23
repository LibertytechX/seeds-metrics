#!/bin/bash

# Simplified FIMR Demo Test Script
# Creates realistic loan and repayment data to demonstrate FIMR tracking

set -e

BASE_URL="http://localhost:8080"
API_URL="$BASE_URL/api/v1"

echo "🧪 FIMR Demo Test Data Generator (Simplified)"
echo "=============================================="
echo ""

# Check if backend is running
echo "🔍 Checking backend health..."
if ! curl -s "$BASE_URL/health" > /dev/null; then
    echo "❌ Backend is not running! Please start it with: cd backend && docker-compose up -d"
    exit 1
fi
echo "✅ Backend is healthy"
echo ""

# Create all test data using SQL
echo "📝 Creating test data in database..."
PGPASSWORD=19sedimat54 psql -h localhost -p 5432 -U postgres -d seedsmetrics <<'EOSQL'

-- Create officer first (required for foreign key constraint)
INSERT INTO officers (officer_id, officer_name, officer_phone, region, branch, created_at)
VALUES
    ('OFF2024012', 'Sarah Johnson', '+234-803-987-6543', 'South West', 'Lagos Main', CURRENT_TIMESTAMP)
ON CONFLICT (officer_id) DO NOTHING;

-- Create customers
INSERT INTO customers (customer_id, customer_name, customer_phone, state, kyc_status, created_at)
VALUES
    ('CUST2024100001', 'Inyang Kpongette', '+234-801-234-5678', 'Lagos', 'Verified', CURRENT_TIMESTAMP),
    ('CUST2024100002', 'Shamsideen Allamu', '+234-803-456-7890', 'Lagos', 'Verified', CURRENT_TIMESTAMP)
ON CONFLICT (customer_id) DO NOTHING;

-- Create Loan 1 - Inyang Kpongette (FIMR case - missed first 10 days)
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
)
VALUES (
    'LN2024100001', 'CUST2024100001', 'Inyang Kpongette', '+234-801-234-5678',
    'OFF2024012', 'Sarah Johnson', '+234-803-987-6543', 'South West', 'Lagos Main', 'Lagos',
    1000000.00, CURRENT_DATE - INTERVAL '60 days', CURRENT_DATE + INTERVAL '30 days', 90,
    0.10, 50000.00, 'Direct', 'Active', CURRENT_TIMESTAMP
)
ON CONFLICT (loan_id) DO UPDATE SET
    loan_amount = EXCLUDED.loan_amount,
    disbursement_date = EXCLUDED.disbursement_date,
    updated_at = CURRENT_TIMESTAMP;

-- Create Loan 2 - Shamsideen Allamu (Good payer - paid from day 1)
INSERT INTO loans (
    loan_id, customer_id, customer_name, customer_phone,
    officer_id, officer_name, officer_phone, region, branch, state,
    loan_amount, disbursement_date, maturity_date, loan_term_days,
    interest_rate, fee_amount, channel, status, created_at
)
VALUES (
    'LN2024100002', 'CUST2024100002', 'Shamsideen Allamu', '+234-803-456-7890',
    'OFF2024012', 'Sarah Johnson', '+234-803-987-6543', 'South West', 'Lagos Main', 'Lagos',
    2000000.00, CURRENT_DATE - INTERVAL '35 days', CURRENT_DATE + INTERVAL '55 days', 90,
    0.10, 100000.00, 'Direct', 'Active', CURRENT_TIMESTAMP
)
ON CONFLICT (loan_id) DO UPDATE SET
    loan_amount = EXCLUDED.loan_amount,
    disbursement_date = EXCLUDED.disbursement_date,
    updated_at = CURRENT_TIMESTAMP;

-- Create loan schedule for Loan 1 (90 daily installments, starting from day 1 after disbursement)
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'LN2024100001',
    day_num,
    (CURRENT_DATE - INTERVAL '60 days') + day_num * INTERVAL '1 day',
    1000000.00 / 90,
    300000.00 / 90,
    50000.00 / 90,
    1350000.00 / 90,
    CASE
        WHEN day_num BETWEEN 11 AND 20 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 90) AS day_num
ON CONFLICT (loan_id, installment_number) DO NOTHING;

-- Create loan schedule for Loan 2 (90 daily installments, starting from day 1 after disbursement)
INSERT INTO loan_schedule (loan_id, installment_number, due_date, principal_due, interest_due, fee_due, total_due, payment_status)
SELECT
    'LN2024100002',
    day_num,
    (CURRENT_DATE - INTERVAL '35 days') + day_num * INTERVAL '1 day',
    2000000.00 / 90,
    600000.00 / 90,
    100000.00 / 90,
    2700000.00 / 90,
    CASE
        WHEN day_num BETWEEN 1 AND 20 THEN 'Paid'
        ELSE 'Pending'
    END
FROM generate_series(1, 90) AS day_num
ON CONFLICT (loan_id, installment_number) DO NOTHING;

-- Create repayments for Loan 1 (Days 11-20, MISSED Days 1-10)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'REP2024' || LPAD((100000 + day_num)::TEXT, 6, '0'),
    'LN2024100001',
    (CURRENT_DATE - INTERVAL '60 days') + (day_num - 1) * INTERVAL '1 day',
    14999.99,
    11111.11,
    3333.33,
    555.55,
    0,
    'Bank Transfer',
    false,
    false,
    0
FROM generate_series(11, 20) AS day_num
ON CONFLICT (repayment_id) DO NOTHING;

-- Create repayments for Loan 2 (Days 1-20, ALL ON TIME - payment on same day as due date)
INSERT INTO repayments (repayment_id, loan_id, payment_date, payment_amount, principal_paid, interest_paid, fees_paid, penalty_paid, payment_method, is_backdated, is_reversed, waiver_amount)
SELECT
    'REP2024' || LPAD((200000 + day_num)::TEXT, 6, '0'),
    'LN2024100002',
    (CURRENT_DATE - INTERVAL '35 days') + day_num * INTERVAL '1 day',
    29999.99,
    22222.22,
    6666.67,
    1111.10,
    0,
    'Cash',
    false,
    false,
    0
FROM generate_series(1, 20) AS day_num
ON CONFLICT (repayment_id) DO NOTHING;

-- Note: Triggers will automatically update loan metrics when repayments are inserted

EOSQL

echo "✅ Test data created successfully!"
echo ""

# Wait for triggers to process
echo "⏳ Waiting 2 seconds for database triggers to process..."
sleep 2
echo ""

echo "=================================="
echo "📊 VERIFICATION"
echo "=================================="
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
curl -s "$API_URL/officers" | jq '.data.officers[] | select(.officer_id == "OFF2024012") | {officer_id, name, calculatedMetrics: {fimr, ayr, dqi, riskScore}}'
echo ""

# Test portfolio metrics
echo "4️⃣  Testing Portfolio Metrics"
echo "   GET $API_URL/metrics/portfolio"
echo "   --------------------------------"
curl -s "$API_URL/metrics/portfolio" | jq '.data | {totalLoans, totalPortfolio, avgDQI, avgAYR, watchlistCount}'
echo ""

# Show database state
echo "5️⃣  Database State"
echo "   --------------------------------"
PGPASSWORD=19sedimat54 psql -h localhost -p 5432 -U postgres -d seedsmetrics <<'EOSQL'
SELECT
    loan_id,
    customer_name,
    loan_amount,
    disbursement_date,
    first_payment_missed,
    fimr_tagged,
    current_dpd,
    (SELECT MIN(payment_date) FROM repayments WHERE repayments.loan_id = loans.loan_id) as first_payment_date,
    (SELECT MIN(due_date) FROM loan_schedule WHERE loan_schedule.loan_id = loans.loan_id) as first_due_date
FROM loans
WHERE loan_id IN ('LN2024100001', 'LN2024100002')
ORDER BY loan_id;
EOSQL

echo ""
echo "=================================="
echo "✅ FIMR Demo Test Complete!"
echo "=================================="
echo ""
echo "📋 Summary:"
echo "   • Loan 1 (Inyang): ₦1,000,000 - MISSED first 10 days (FIMR)"
echo "   • Loan 2 (Shamsideen): ₦2,000,000 - Paid on time (NO FIMR)"
echo "   • Both loans assigned to officer: OFF2024012"
echo ""
echo "🔍 Next Steps:"
echo "   • Check FIMR dashboard at: http://localhost:5173/"
echo "   • Query specific loan: curl $API_URL/officers/OFF2024012 | jq ."
echo "   • View database: PGPASSWORD=19sedimat54 psql -h localhost -p 5432 -U postgres -d seedsmetrics"
echo ""

