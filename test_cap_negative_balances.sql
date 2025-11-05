-- ============================================================================
-- Test Script for Migration 015: Cap Negative Outstanding Balances
-- ============================================================================
-- This script tests the new trigger function to ensure it correctly caps
-- negative outstanding balances at zero.
-- ============================================================================

-- ============================================================================
-- TEST 1: Verify Trigger Function Exists and is Updated
-- ============================================================================
SELECT 
    'TEST 1: Trigger Function Check' as test_name,
    CASE 
        WHEN EXISTS (
            SELECT 1 FROM pg_proc 
            WHERE proname = 'update_loan_computed_fields'
        ) THEN 'PASS: Function exists'
        ELSE 'FAIL: Function not found'
    END as result;

-- ============================================================================
-- TEST 2: Check for Negative Balances (Should be 0 after migration)
-- ============================================================================
SELECT 
    'TEST 2: No Negative Balances' as test_name,
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS: No negative balances found'
        ELSE 'FAIL: ' || COUNT(*)::TEXT || ' loans with negative balances'
    END as result
FROM loans
WHERE principal_outstanding < 0 
   OR interest_outstanding < 0 
   OR fees_outstanding < 0 
   OR total_outstanding < 0
   OR actual_outstanding < 0;

-- ============================================================================
-- TEST 3: Verify Officer Portfolio Total is Non-Negative
-- ============================================================================
SELECT 
    'TEST 3: Officer Portfolio Total' as test_name,
    CASE 
        WHEN SUM(principal_outstanding) >= 0 THEN 
            'PASS: Portfolio total = ' || COALESCE(SUM(principal_outstanding), 0)::TEXT
        ELSE 
            'FAIL: Portfolio total is negative: ' || SUM(principal_outstanding)::TEXT
    END as result
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';

-- ============================================================================
-- TEST 4: Verify Over-Paid Loans Show 0 Outstanding
-- ============================================================================
SELECT 
    'TEST 4: Over-Paid Loans' as test_name,
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS: All over-paid loans have 0 outstanding'
        WHEN COUNT(*) > 0 THEN 'FAIL: ' || COUNT(*)::TEXT || ' over-paid loans with non-zero outstanding'
        ELSE 'PASS: No over-paid loans found'
    END as result
FROM loans
WHERE total_principal_paid > loan_amount
  AND principal_outstanding != 0;

-- ============================================================================
-- TEST 5: Verify Total Outstanding Calculation
-- ============================================================================
SELECT 
    'TEST 5: Total Outstanding Calculation' as test_name,
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS: All total_outstanding values are correct'
        ELSE 'FAIL: ' || COUNT(*)::TEXT || ' loans with incorrect total_outstanding'
    END as result
FROM loans
WHERE total_outstanding != (principal_outstanding + interest_outstanding + fees_outstanding);

-- ============================================================================
-- TEST 6: Detailed Check for Officer adeyinka232803@gmail.com
-- ============================================================================
SELECT 
    'TEST 6: Officer Detailed Check' as test_name,
    'Loans: ' || COUNT(*)::TEXT || 
    ', Portfolio: ' || COALESCE(SUM(principal_outstanding), 0)::TEXT ||
    ', Negative: ' || SUM(CASE WHEN principal_outstanding < 0 THEN 1 ELSE 0 END)::TEXT as result
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';

-- ============================================================================
-- TEST 7: Sample Over-Paid Loans (for manual verification)
-- ============================================================================
SELECT 
    'TEST 7: Sample Over-Paid Loans' as section,
    loan_id,
    customer_name,
    loan_amount,
    total_principal_paid,
    principal_outstanding,
    (total_principal_paid - loan_amount) as overpayment,
    CASE 
        WHEN principal_outstanding = 0 THEN 'CORRECT'
        ELSE 'INCORRECT'
    END as status
FROM loans
WHERE total_principal_paid > loan_amount
LIMIT 5;

-- ============================================================================
-- TEST 8: All Officers Portfolio Totals (should all be >= 0)
-- ============================================================================
SELECT 
    'TEST 8: All Officers Check' as test_name,
    CASE 
        WHEN COUNT(*) = 0 THEN 'PASS: No officers with negative portfolio'
        ELSE 'FAIL: ' || COUNT(*)::TEXT || ' officers with negative portfolio'
    END as result
FROM (
    SELECT 
        officer_id,
        SUM(principal_outstanding) as portfolio_total
    FROM loans
    GROUP BY officer_id
    HAVING SUM(principal_outstanding) < 0
) negative_portfolios;

-- ============================================================================
-- TEST SUMMARY
-- ============================================================================
SELECT 
    '========================================' as separator,
    'TEST SUMMARY' as title,
    '========================================' as separator2;

-- Count of tests passed/failed
WITH test_results AS (
    SELECT 
        CASE 
            WHEN COUNT(*) = 0 THEN 1 ELSE 0
        END as test_2_pass
    FROM loans
    WHERE principal_outstanding < 0 
       OR interest_outstanding < 0 
       OR fees_outstanding < 0 
       OR total_outstanding < 0
       OR actual_outstanding < 0
    
    UNION ALL
    
    SELECT 
        CASE 
            WHEN SUM(principal_outstanding) >= 0 THEN 1 ELSE 0
        END as test_3_pass
    FROM loans
    WHERE officer_id = 'adeyinka232803@gmail.com'
    
    UNION ALL
    
    SELECT 
        CASE 
            WHEN COUNT(*) = 0 THEN 1 ELSE 0
        END as test_4_pass
    FROM loans
    WHERE total_principal_paid > loan_amount
      AND principal_outstanding != 0
    
    UNION ALL
    
    SELECT 
        CASE 
            WHEN COUNT(*) = 0 THEN 1 ELSE 0
        END as test_5_pass
    FROM loans
    WHERE total_outstanding != (principal_outstanding + interest_outstanding + fees_outstanding)
    
    UNION ALL
    
    SELECT 
        CASE 
            WHEN COUNT(*) = 0 THEN 1 ELSE 0
        END as test_8_pass
    FROM (
        SELECT officer_id, SUM(principal_outstanding) as portfolio_total
        FROM loans
        GROUP BY officer_id
        HAVING SUM(principal_outstanding) < 0
    ) negative_portfolios
)
SELECT 
    'Total Tests Run: 5' as summary,
    'Tests Passed: ' || SUM(test_2_pass)::TEXT as passed,
    'Tests Failed: ' || (5 - SUM(test_2_pass))::TEXT as failed,
    CASE 
        WHEN SUM(test_2_pass) = 5 THEN '✓ ALL TESTS PASSED'
        ELSE '✗ SOME TESTS FAILED - Review results above'
    END as status
FROM test_results;

-- ============================================================================
-- DETAILED STATISTICS
-- ============================================================================
SELECT 
    '========================================' as separator,
    'DETAILED STATISTICS' as title,
    '========================================' as separator2;

SELECT 
    'Total Loans' as metric,
    COUNT(*)::TEXT as value
FROM loans

UNION ALL

SELECT 
    'Loans with Negative Balances' as metric,
    COUNT(*)::TEXT as value
FROM loans
WHERE principal_outstanding < 0 
   OR interest_outstanding < 0 
   OR fees_outstanding < 0 
   OR total_outstanding < 0

UNION ALL

SELECT 
    'Over-Paid Loans' as metric,
    COUNT(*)::TEXT as value
FROM loans
WHERE total_principal_paid > loan_amount

UNION ALL

SELECT 
    'Total Over-Payment Amount' as metric,
    COALESCE(SUM(total_principal_paid - loan_amount), 0)::TEXT as value
FROM loans
WHERE total_principal_paid > loan_amount

UNION ALL

SELECT 
    'Officers with Negative Portfolio' as metric,
    COUNT(*)::TEXT as value
FROM (
    SELECT officer_id, SUM(principal_outstanding) as portfolio_total
    FROM loans
    GROUP BY officer_id
    HAVING SUM(principal_outstanding) < 0
) negative_portfolios

UNION ALL

SELECT 
    'Officer adeyinka232803@gmail.com Portfolio' as metric,
    COALESCE(SUM(principal_outstanding), 0)::TEXT as value
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';

