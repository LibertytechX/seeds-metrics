-- ============================================================================
-- Diagnostic Script: Check Migration 015 Status and Officer Portfolio
-- ============================================================================
-- This script checks if migration 015 has been applied and diagnoses why
-- officer 'adeyinka232803@gmail.com' still has a negative portfolio total.
-- ============================================================================

\echo '============================================================================'
\echo 'DIAGNOSTIC 1: Check if Trigger Function Has GREATEST() Logic'
\echo '============================================================================'
\echo ''

-- Get the trigger function source code to verify if it has GREATEST() logic
SELECT 
    'Trigger Function Source Check' as diagnostic,
    CASE 
        WHEN pg_get_functiondef(oid) LIKE '%GREATEST(0,%' 
        THEN '✓ MIGRATION APPLIED: Trigger has GREATEST() logic'
        ELSE '✗ MIGRATION NOT APPLIED: Trigger missing GREATEST() logic'
    END as status,
    LENGTH(pg_get_functiondef(oid)) as function_length
FROM pg_proc 
WHERE proname = 'update_loan_computed_fields';

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC 2: Officer Portfolio Total - Current State'
\echo '============================================================================'
\echo ''

-- Check current portfolio total for the officer
SELECT 
    'adeyinka232803@gmail.com' as officer_id,
    COUNT(*) as total_loans,
    COUNT(*) FILTER (WHERE status = 'Active') as active_loans,
    SUM(principal_outstanding) as portfolio_total,
    SUM(CASE WHEN principal_outstanding < 0 THEN 1 ELSE 0 END) as loans_with_negative_balance,
    SUM(CASE WHEN principal_outstanding < 0 THEN principal_outstanding ELSE 0 END) as total_negative_amount,
    MIN(principal_outstanding) as min_outstanding,
    MAX(principal_outstanding) as max_outstanding
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC 3: Loans with Negative Balances for This Officer'
\echo '============================================================================'
\echo ''

-- List all loans with negative balances for this officer
SELECT 
    loan_id,
    customer_name,
    status,
    loan_amount,
    total_principal_paid,
    principal_outstanding,
    interest_outstanding,
    fees_outstanding,
    total_outstanding,
    (total_principal_paid - loan_amount) as overpayment,
    disbursement_date,
    updated_at
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com'
  AND (principal_outstanding < 0 
       OR interest_outstanding < 0 
       OR fees_outstanding < 0 
       OR total_outstanding < 0)
ORDER BY principal_outstanding ASC;

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC 4: All Loans for This Officer (Summary)'
\echo '============================================================================'
\echo ''

-- Summary of all loans for this officer
SELECT 
    status,
    COUNT(*) as loan_count,
    SUM(loan_amount) as total_loan_amount,
    SUM(total_principal_paid) as total_paid,
    SUM(principal_outstanding) as total_outstanding,
    SUM(CASE WHEN principal_outstanding < 0 THEN 1 ELSE 0 END) as negative_count
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com'
GROUP BY status
ORDER BY status;

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC 5: Check System-Wide Negative Balances'
\echo '============================================================================'
\echo ''

-- Check if there are other loans with negative balances
SELECT 
    'System-Wide Negative Balance Check' as diagnostic,
    COUNT(*) as total_loans_with_negative_balance,
    COUNT(DISTINCT officer_id) as officers_affected,
    SUM(principal_outstanding) as total_negative_amount
FROM loans
WHERE principal_outstanding < 0 
   OR interest_outstanding < 0 
   OR fees_outstanding < 0 
   OR total_outstanding < 0;

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC 6: Sample of Negative Balance Loans (Top 10)'
\echo '============================================================================'
\echo ''

-- Show top 10 loans with most negative balances
SELECT 
    loan_id,
    officer_id,
    customer_name,
    loan_amount,
    total_principal_paid,
    principal_outstanding,
    (total_principal_paid - loan_amount) as overpayment,
    updated_at
FROM loans
WHERE principal_outstanding < 0
ORDER BY principal_outstanding ASC
LIMIT 10;

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC 7: Check Trigger Function Details'
\echo '============================================================================'
\echo ''

-- Get trigger function metadata
SELECT 
    p.proname as function_name,
    pg_get_function_identity_arguments(p.oid) as arguments,
    l.lanname as language,
    CASE p.provolatile
        WHEN 'i' THEN 'IMMUTABLE'
        WHEN 's' THEN 'STABLE'
        WHEN 'v' THEN 'VOLATILE'
    END as volatility,
    pg_size_pretty(pg_relation_size(p.oid)) as size
FROM pg_proc p
JOIN pg_language l ON p.prolang = l.oid
WHERE p.proname = 'update_loan_computed_fields';

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC 8: Check When Loans Were Last Updated'
\echo '============================================================================'
\echo ''

-- Check when loans for this officer were last updated
SELECT 
    'Last Update Times' as diagnostic,
    COUNT(*) as total_loans,
    MIN(updated_at) as earliest_update,
    MAX(updated_at) as latest_update,
    COUNT(*) FILTER (WHERE updated_at < CURRENT_DATE - INTERVAL '1 day') as updated_before_today,
    COUNT(*) FILTER (WHERE updated_at >= CURRENT_DATE) as updated_today
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC 9: Check Repayments for Negative Balance Loans'
\echo '============================================================================'
\echo ''

-- Check repayments for loans with negative balances
SELECT 
    l.loan_id,
    l.customer_name,
    l.principal_outstanding,
    COUNT(r.repayment_id) as repayment_count,
    SUM(r.principal_paid) as total_principal_paid_from_repayments,
    SUM(r.payment_amount) as total_payment_amount,
    MAX(r.payment_date) as last_payment_date,
    SUM(CASE WHEN r.is_reversed = TRUE THEN 1 ELSE 0 END) as reversed_count
FROM loans l
LEFT JOIN repayments r ON l.loan_id = r.loan_id
WHERE l.officer_id = 'adeyinka232803@gmail.com'
  AND l.principal_outstanding < 0
GROUP BY l.loan_id, l.customer_name, l.principal_outstanding
ORDER BY l.principal_outstanding ASC;

\echo ''
\echo '============================================================================'
\echo 'DIAGNOSTIC SUMMARY'
\echo '============================================================================'
\echo ''

-- Summary of findings
SELECT 
    'SUMMARY' as section,
    (SELECT COUNT(*) FROM loans WHERE officer_id = 'adeyinka232803@gmail.com') as total_loans,
    (SELECT SUM(principal_outstanding) FROM loans WHERE officer_id = 'adeyinka232803@gmail.com') as portfolio_total,
    (SELECT COUNT(*) FROM loans WHERE officer_id = 'adeyinka232803@gmail.com' AND principal_outstanding < 0) as negative_loans,
    (SELECT CASE WHEN pg_get_functiondef(oid) LIKE '%GREATEST(0,%' THEN 'YES' ELSE 'NO' END 
     FROM pg_proc WHERE proname = 'update_loan_computed_fields') as migration_applied;

