-- ============================================================================
-- Quick Diagnosis: Check Migration Status and Negative Portfolio
-- ============================================================================
-- Run this with: psql -h <host> -p <port> -U metricsuser -d seedsmetrics -f quick_diagnosis.sql
-- ============================================================================

-- CHECK 1: Has migration 015 been applied?
SELECT 
    '=== CHECK 1: Migration Status ===' as check,
    CASE 
        WHEN pg_get_functiondef(oid) LIKE '%GREATEST(0,%' 
        THEN 'APPLIED - Trigger has GREATEST() logic'
        ELSE 'NOT APPLIED - Trigger missing GREATEST() logic'
    END as status
FROM pg_proc 
WHERE proname = 'update_loan_computed_fields';

-- CHECK 2: Officer portfolio total
SELECT 
    '=== CHECK 2: Officer Portfolio ===' as check,
    COUNT(*) as total_loans,
    SUM(principal_outstanding) as portfolio_total,
    SUM(CASE WHEN principal_outstanding < 0 THEN 1 ELSE 0 END) as negative_loans,
    SUM(CASE WHEN principal_outstanding < 0 THEN principal_outstanding ELSE 0 END) as negative_amount
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';

-- CHECK 3: List negative balance loans
SELECT 
    '=== CHECK 3: Negative Balance Loans ===' as check,
    loan_id,
    customer_name,
    loan_amount,
    total_principal_paid,
    principal_outstanding,
    (total_principal_paid - loan_amount) as overpayment,
    updated_at
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com'
  AND principal_outstanding < 0
ORDER BY principal_outstanding ASC;

