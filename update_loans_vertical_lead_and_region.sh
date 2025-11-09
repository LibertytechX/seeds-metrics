#!/bin/bash

# Script to update loans table with vertical lead information and corrected regions from officers table
# This links each loan to its officer's vertical leadership structure

set -e

echo "=========================================="
echo "Updating Loans with Vertical Lead & Region"
echo "=========================================="
echo ""

echo "📊 Current State Analysis..."
echo ""

# Check current state
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -c "
SELECT 
    COUNT(*) as total_loans,
    COUNT(DISTINCT officer_id) as unique_officers,
    COUNT(vertical_lead_email) as loans_with_vertical_lead,
    COUNT(CASE WHEN region != '\''Nigeria'\'' THEN 1 END) as loans_with_specific_region
FROM loans;
" 2>&1' | grep -v "^ssh"

echo ""
echo "📊 Expected Coverage..."
echo ""

ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -c "
SELECT 
    COUNT(DISTINCT l.loan_id) as loans_to_update,
    COUNT(DISTINCT l.officer_id) as officers_with_data
FROM loans l
JOIN officers o ON l.officer_id = o.officer_id
WHERE o.vertical_lead_email IS NOT NULL;
" 2>&1' | grep -v "^ssh"

echo ""
read -p "Do you want to proceed with the update? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "❌ Update cancelled"
    exit 0
fi

echo ""
echo "🚀 Updating loans table..."
echo ""

# Execute the update
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -c "
BEGIN;

-- Update vertical lead information and region from officers table
UPDATE loans l
SET 
    vertical_lead_name = o.vertical_lead_name,
    vertical_lead_email = o.vertical_lead_email,
    region = o.region
FROM officers o
WHERE l.officer_id = o.officer_id;

COMMIT;
" 2>&1'

echo ""
echo "✅ Update completed!"
echo ""

echo "🔍 Verification..."
echo ""

# Verify the updates
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -c "
SELECT 
    COUNT(*) as total_loans,
    COUNT(vertical_lead_email) as loans_with_vertical_lead,
    COUNT(CASE WHEN region != '\''Nigeria'\'' THEN 1 END) as loans_with_specific_region,
    ROUND(100.0 * COUNT(vertical_lead_email) / COUNT(*), 2) as vertical_lead_coverage_pct,
    ROUND(100.0 * COUNT(CASE WHEN region != '\''Nigeria'\'' THEN 1 END) / COUNT(*), 2) as specific_region_pct
FROM loans;
" 2>&1' | grep -v "^ssh"

echo ""
echo "📊 Region Distribution in Loans:"
echo ""

ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -c "
SELECT 
    region,
    COUNT(*) as loan_count,
    ROUND(100.0 * COUNT(*) / SUM(COUNT(*)) OVER (), 2) as percentage
FROM loans
GROUP BY region
ORDER BY loan_count DESC
LIMIT 15;
" 2>&1' | grep -v "^ssh"

echo ""
echo "📊 Vertical Lead Distribution in Loans:"
echo ""

ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -c "
SELECT 
    vertical_lead_name,
    vertical_lead_email,
    COUNT(*) as loan_count
FROM loans
WHERE vertical_lead_email IS NOT NULL
GROUP BY vertical_lead_name, vertical_lead_email
ORDER BY loan_count DESC
LIMIT 15;
" 2>&1' | grep -v "^ssh"

echo ""
echo "✅ Sample Loans with Vertical Lead Data:"
echo ""

ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -c "
SELECT 
    loan_id,
    customer_name,
    officer_name,
    region,
    vertical_lead_name,
    vertical_lead_email
FROM loans
WHERE vertical_lead_email IS NOT NULL
LIMIT 5;
" 2>&1' | grep -v "^ssh"

echo ""
echo "✅ Done!"

