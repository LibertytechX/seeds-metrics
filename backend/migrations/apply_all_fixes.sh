#!/bin/bash

# Script to apply all three fixes:
# 1. Add total_repayments column
# 2. Update DPD calculation to use linear repayment progress formula
# 3. Frontend changes for "View Entire Portfolio" are code-only (no DB changes)

set -e

echo "🔧 Applying all fixes to the database..."

# Database connection details
DB_HOST="generaldb-do-user-9489371-0.k.db.ondigitalocean.com"
DB_PORT="25060"
DB_USER="seedsuser"
DB_PASSWORD="@seedsuser2020"
DB_NAME="seedsmetrics"

export PGPASSWORD="$DB_PASSWORD"

echo ""
echo "📋 Step 1: Adding total_repayments column..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
  -f backend/migrations/005_add_total_repayments_column.sql

echo ""
echo "✅ total_repayments column added successfully!"

echo ""
echo "📋 Step 2: Updating DPD calculation to use linear repayment progress formula..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
  -f backend/migrations/006_update_dpd_linear_calculation.sql

echo ""
echo "✅ DPD calculation updated successfully!"

echo ""
echo "📊 Step 3: Displaying updated loan data..."

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
SELECT 
    loan_id,
    customer_name,
    TO_CHAR(disbursement_date, 'DD Mon YYYY') as disbursement_date,
    loan_amount,
    loan_term_days,
    total_repayments,
    total_outstanding,
    current_dpd,
    CASE 
        WHEN current_dpd = 0 THEN 'Current'
        WHEN current_dpd BETWEEN 1 AND 6 THEN 'Early Indicator (1-6 days)'
        WHEN current_dpd BETWEEN 7 AND 15 THEN 'Overdue (7-15 days)'
        WHEN current_dpd > 15 THEN 'Overdue > 15 days'
    END as dpd_status
FROM loans
ORDER BY current_dpd DESC, disbursement_date;
EOF

echo ""
echo "📊 Step 4: Detailed calculation for Loan ID 3..."

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
SELECT 
    l.loan_id,
    l.customer_name,
    TO_CHAR(l.disbursement_date, 'DD Mon YYYY') as disbursement_date,
    l.loan_amount,
    l.loan_term_days as duration_days,
    l.interest_rate,
    l.fee_amount,
    -- Total expected amount
    (l.loan_amount + (l.loan_amount * l.interest_rate * l.loan_term_days / 365) + l.fee_amount) as total_expected_amount,
    -- Total repaid
    l.total_repayments,
    -- Total outstanding
    l.total_outstanding,
    -- Days since disbursement
    (CURRENT_DATE - l.disbursement_date) as expected_days,
    -- Daily payment rate
    ROUND((l.loan_amount + (l.loan_amount * l.interest_rate * l.loan_term_days / 365) + l.fee_amount) / l.loan_term_days, 2) as daily_payment_rate,
    -- Paid days
    ROUND(l.total_repayments / ((l.loan_amount + (l.loan_amount * l.interest_rate * l.loan_term_days / 365) + l.fee_amount) / l.loan_term_days), 2) as paid_days,
    -- Current DPD
    l.current_dpd,
    -- Calculation breakdown
    CONCAT(
        'Expected Days: ', (CURRENT_DATE - l.disbursement_date), 
        ' - Paid Days: ', ROUND(l.total_repayments / ((l.loan_amount + (l.loan_amount * l.interest_rate * l.loan_term_days / 365) + l.fee_amount) / l.loan_term_days), 2),
        ' = DPD: ', l.current_dpd
    ) as calculation_breakdown
FROM loans l
WHERE l.loan_id = '3';
EOF

echo ""
echo "🎉 All fixes applied successfully!"
echo ""
echo "Summary of changes:"
echo "1. ✅ Added total_repayments column to loans table"
echo "2. ✅ Updated DPD calculation to use linear repayment progress formula"
echo "3. ✅ Frontend changes for 'View Entire Portfolio' (code-only, no DB changes)"
echo ""
echo "Next steps:"
echo "1. Rebuild the backend: cd backend && go build -o bin/api ./cmd/api"
echo "2. Restart the backend: ./backend/bin/api"
echo "3. Verify the changes in the frontend"

