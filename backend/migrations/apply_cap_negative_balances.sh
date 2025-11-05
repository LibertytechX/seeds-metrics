#!/bin/bash

# ============================================================================
# Script to Apply Migration 015: Cap Negative Outstanding Balances
# ============================================================================
# This script applies the migration that caps negative outstanding balances
# at zero to handle over-payment scenarios.
# ============================================================================

set -e  # Exit on error

# Database connection details
DB_HOST="${DB_HOST:-private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com}"
DB_PORT="${DB_PORT:-25060}"
DB_NAME="${DB_NAME:-seedsmetrics}"
DB_USER="${DB_USER:-metricsuser}"

echo "============================================================================"
echo "Applying Migration 015: Cap Negative Outstanding Balances"
echo "============================================================================"
echo ""
echo "This migration will:"
echo "  1. Update the trigger function to cap negative balances at 0"
echo "  2. Recalculate all existing loans to fix negative balances"
echo "  3. Verify the fix for officer 'adeyinka232803@gmail.com'"
echo ""
echo "Database: ${DB_NAME}@${DB_HOST}:${DB_PORT}"
echo ""

# Check if psql is installed
if ! command -v psql &> /dev/null; then
    echo "ERROR: psql is not installed. Please install PostgreSQL client."
    exit 1
fi

# Prompt for password if not set
if [ -z "$DB_PASSWORD" ]; then
    echo "Please enter the database password:"
    read -s DB_PASSWORD
    export DB_PASSWORD
    echo ""
fi

echo "============================================================================"
echo "STEP 1: Backup Current State"
echo "============================================================================"
echo ""

# Create backup of current loan states
echo "Creating backup of loans with negative balances..."
PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    -c "COPY (
        SELECT 
            loan_id, 
            customer_name, 
            officer_id,
            loan_amount, 
            total_principal_paid,
            principal_outstanding, 
            interest_outstanding,
            fees_outstanding,
            total_outstanding,
            actual_outstanding
        FROM loans 
        WHERE principal_outstanding < 0 
           OR interest_outstanding < 0 
           OR fees_outstanding < 0 
           OR total_outstanding < 0
           OR actual_outstanding < 0
    ) TO STDOUT WITH CSV HEADER" > backup_negative_balances_$(date +%Y%m%d_%H%M%S).csv

if [ $? -eq 0 ]; then
    echo "✓ Backup created successfully"
else
    echo "✗ Backup failed"
    exit 1
fi

echo ""
echo "============================================================================"
echo "STEP 2: Check Current State"
echo "============================================================================"
echo ""

echo "Checking officer 'adeyinka232803@gmail.com' current portfolio total..."
PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    -c "SELECT 
            COUNT(*) as total_loans,
            SUM(principal_outstanding) as portfolio_total,
            SUM(CASE WHEN principal_outstanding < 0 THEN 1 ELSE 0 END) as negative_loans,
            SUM(CASE WHEN principal_outstanding < 0 THEN principal_outstanding ELSE 0 END) as negative_amount
        FROM loans
        WHERE officer_id = 'adeyinka232803@gmail.com';"

echo ""
echo "Checking all loans with negative balances..."
PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    -c "SELECT 
            COUNT(*) as total_negative_loans,
            SUM(CASE WHEN principal_outstanding < 0 THEN principal_outstanding ELSE 0 END) as total_negative_principal,
            SUM(CASE WHEN interest_outstanding < 0 THEN interest_outstanding ELSE 0 END) as total_negative_interest,
            SUM(CASE WHEN fees_outstanding < 0 THEN fees_outstanding ELSE 0 END) as total_negative_fees
        FROM loans
        WHERE principal_outstanding < 0 
           OR interest_outstanding < 0 
           OR fees_outstanding < 0;"

echo ""
echo "============================================================================"
echo "STEP 3: Apply Migration"
echo "============================================================================"
echo ""

echo "Applying migration 015_cap_negative_outstanding_balances.sql..."
PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    -f "$(dirname "$0")/015_cap_negative_outstanding_balances.sql"

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Migration applied successfully"
else
    echo ""
    echo "✗ Migration failed"
    exit 1
fi

echo ""
echo "============================================================================"
echo "STEP 4: Verify Results"
echo "============================================================================"
echo ""

echo "Checking officer 'adeyinka232803@gmail.com' portfolio total after fix..."
PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    -c "SELECT 
            COUNT(*) as total_loans,
            SUM(principal_outstanding) as portfolio_total,
            SUM(CASE WHEN principal_outstanding < 0 THEN 1 ELSE 0 END) as negative_loans,
            SUM(CASE WHEN principal_outstanding < 0 THEN principal_outstanding ELSE 0 END) as negative_amount
        FROM loans
        WHERE officer_id = 'adeyinka232803@gmail.com';"

echo ""
echo "Checking for any remaining negative balances (should be 0)..."
PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    -c "SELECT 
            COUNT(*) as remaining_negative_loans
        FROM loans
        WHERE principal_outstanding < 0 
           OR interest_outstanding < 0 
           OR fees_outstanding < 0 
           OR total_outstanding < 0
           OR actual_outstanding < 0;"

echo ""
echo "Summary of over-paid loans (now showing 0 outstanding)..."
PGPASSWORD="${DB_PASSWORD}" psql \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    -c "SELECT 
            COUNT(*) as overpaid_loan_count,
            SUM(total_principal_paid - loan_amount) as total_overpayment_amount,
            SUM(principal_outstanding) as total_principal_outstanding
        FROM loans
        WHERE total_principal_paid > loan_amount;"

echo ""
echo "============================================================================"
echo "Migration Complete!"
echo "============================================================================"
echo ""
echo "Summary:"
echo "  - Trigger function updated to cap negative balances at 0"
echo "  - All existing loans recalculated"
echo "  - Backup saved to: backup_negative_balances_*.csv"
echo ""
echo "Next steps:"
echo "  1. Verify the officer's portfolio total in the dashboard"
echo "  2. Check that the Agent Performance table shows correct values"
echo "  3. Monitor for any new over-payment scenarios"
echo ""

