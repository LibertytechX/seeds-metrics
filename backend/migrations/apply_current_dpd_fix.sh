#!/bin/bash

# Script to apply current_dpd calculation fix
# This script:
# 1. Applies the updated trigger function
# 2. Forces recalculation of current_dpd for all existing loans

set -e

echo "🔧 Applying current_dpd calculation fix..."

# Database connection details
DB_HOST="generaldb-do-user-9489371-0.k.db.ondigitalocean.com"
DB_PORT="25060"
DB_USER="seedsuser"
DB_PASSWORD="@seedsuser2020"
DB_NAME="seedsmetrics"

export PGPASSWORD="$DB_PASSWORD"

echo ""
echo "📋 Step 1: Applying migration 004_update_current_dpd_calculation.sql..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
  -f backend/migrations/004_update_current_dpd_calculation.sql

echo ""
echo "✅ Migration applied successfully!"

echo ""
echo "📋 Step 2: Force recalculation of current_dpd for all existing loans..."
echo "   (This will update all repayments to trigger the recalculation)"

# Update all repayments to trigger the recalculation
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
-- Update all repayments to trigger recalculation
UPDATE repayments 
SET updated_at = CURRENT_TIMESTAMP 
WHERE repayment_id IN (
    SELECT DISTINCT r.repayment_id
    FROM repayments r
    WHERE r.is_reversed = FALSE
);
EOF

echo ""
echo "✅ Recalculation triggered for all loans with repayments!"

echo ""
echo "📋 Step 3: For loans with NO repayments, manually trigger calculation..."

# For loans with no repayments, we need to insert a dummy repayment and then delete it
# OR we can directly update the current_dpd field
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
-- Update current_dpd for loans with no repayments
UPDATE loans
SET 
    current_dpd = GREATEST(0, CURRENT_DATE - disbursement_date - 30),
    updated_at = CURRENT_TIMESTAMP
WHERE loan_id NOT IN (
    SELECT DISTINCT loan_id FROM repayments WHERE is_reversed = FALSE
)
AND total_outstanding > 0;
EOF

echo ""
echo "✅ current_dpd updated for loans with no repayments!"

echo ""
echo "📊 Step 4: Displaying current_dpd values for all loans..."

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
SELECT 
    loan_id,
    customer_name,
    disbursement_date,
    current_dpd,
    total_outstanding,
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
echo "📊 Step 5: Counting loans by DPD bucket..."

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" <<EOF
SELECT 
    CASE 
        WHEN current_dpd = 0 THEN 'Current (0 days)'
        WHEN current_dpd BETWEEN 1 AND 6 THEN 'Early Indicator (1-6 days)'
        WHEN current_dpd BETWEEN 7 AND 15 THEN 'Overdue (7-15 days)'
        WHEN current_dpd > 15 THEN 'Overdue > 15 days'
    END as dpd_bucket,
    COUNT(*) as loan_count,
    SUM(total_outstanding) as total_outstanding
FROM loans
WHERE status = 'ACTIVE'
GROUP BY 
    CASE 
        WHEN current_dpd = 0 THEN 'Current (0 days)'
        WHEN current_dpd BETWEEN 1 AND 6 THEN 'Early Indicator (1-6 days)'
        WHEN current_dpd BETWEEN 7 AND 15 THEN 'Overdue (7-15 days)'
        WHEN current_dpd > 15 THEN 'Overdue > 15 days'
    END
ORDER BY 
    CASE 
        WHEN current_dpd = 0 THEN 1
        WHEN current_dpd BETWEEN 1 AND 6 THEN 2
        WHEN current_dpd BETWEEN 7 AND 15 THEN 3
        WHEN current_dpd > 15 THEN 4
    END;
EOF

echo ""
echo "🎉 current_dpd calculation fix applied successfully!"
echo ""
echo "Next steps:"
echo "1. Verify the current_dpd values are correct"
echo "2. Test the 'overdue > 15 days' metric in the application"
echo "3. Check that the metric updates automatically when new repayments are added"

