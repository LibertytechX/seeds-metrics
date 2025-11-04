#!/bin/bash

# Script to fix repayment_amount for all loans by fetching correct values from Django
# and updating SeedsMetrics database

set -e

# Load environment variables
source /home/seeds-metrics-backend/backend/.env

echo "=== Fixing Repayment Amounts ==="
echo "Fetching correct repayment amounts from Django database..."

# Create a temporary SQL file
TMP_SQL="/tmp/fix_repayment_amounts_$$.sql"

# Generate UPDATE statements by querying Django database
psql "host=164.90.155.2 port=5432 dbname=savings user=metricsuser password=EiRXo6IfeHQuM3wcbZ67\$LzwmVKCXhpUhWg sslmode=require" -t -A -F'|' -c \
"SELECT 
    'UPDATE loans SET repayment_amount = ' || repayment_amount || ', updated_at = CURRENT_TIMESTAMP WHERE loan_id = ''' || id || ''';'
FROM loans_ajoloan 
WHERE is_disbursed = TRUE
ORDER BY id;" > "$TMP_SQL"

echo "Generated $(wc -l < $TMP_SQL) UPDATE statements"

# Show sample before update
echo ""
echo "=== Sample BEFORE update (Loan 6) ==="
psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=$DB_SSLMODE" -c \
"SELECT loan_id, loan_amount, repayment_amount FROM loans WHERE loan_id = '6';"

# Execute the updates
echo ""
echo "Applying updates to SeedsMetrics database..."
psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=$DB_SSLMODE" -f "$TMP_SQL" > /dev/null 2>&1

# Show sample after update
echo ""
echo "=== Sample AFTER update (Loan 6) ==="
psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=$DB_SSLMODE" -c \
"SELECT loan_id, loan_amount, repayment_amount FROM loans WHERE loan_id = '6';"

# Show summary
echo ""
echo "=== Update Summary ==="
psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=$DB_SSLMODE" -c \
"SELECT 
    COUNT(*) as total_loans,
    ROUND(AVG(repayment_amount)::numeric, 2) as avg_repayment_amount,
    ROUND(MIN(repayment_amount)::numeric, 2) as min_repayment_amount,
    ROUND(MAX(repayment_amount)::numeric, 2) as max_repayment_amount
FROM loans;"

# Clean up
rm -f "$TMP_SQL"

echo ""
echo "✅ Repayment amounts fixed successfully!"

