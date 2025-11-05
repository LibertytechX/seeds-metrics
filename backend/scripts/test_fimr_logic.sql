-- ============================================================================
-- Test Script: FIMR Logic Verification
-- ============================================================================
-- This script tests the FIMR (First Installment Missed Rate) calculation logic
-- to ensure loans are correctly tagged based on their first payment status.
--
-- Correct FIMR Logic:
-- - FIMR = TRUE if NO repayment on or before first_payment_due_date
-- - FIMR = FALSE if at least one repayment exists on or before first_payment_due_date
-- - Early payments should NOT be tagged as FIMR
-- ============================================================================

-- Test 1: Loans with early payments (should have fimr_tagged = FALSE)
-- These loans made their first payment BEFORE the due date
SELECT 
    'Test 1: Early Payments' as test_name,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as correctly_tagged_false,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as incorrectly_tagged_true
FROM loans l
WHERE first_payment_received_date < first_payment_due_date
  AND first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL;

-- Test 2: Loans with on-time payments (should have fimr_tagged = FALSE)
-- These loans made their first payment ON the due date
SELECT 
    'Test 2: On-Time Payments' as test_name,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as correctly_tagged_false,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as incorrectly_tagged_true
FROM loans l
WHERE first_payment_received_date = first_payment_due_date
  AND first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL;

-- Test 3: Loans with late payments (should have fimr_tagged = TRUE)
-- These loans made their first payment AFTER the due date
SELECT 
    'Test 3: Late Payments' as test_name,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as correctly_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as incorrectly_tagged_false
FROM loans l
WHERE first_payment_received_date > first_payment_due_date
  AND first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL;

-- Test 4: Loans with no payments (should have fimr_tagged = TRUE)
-- These loans have NOT received any payment
SELECT 
    'Test 4: No Payments' as test_name,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as correctly_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as incorrectly_tagged_false
FROM loans l
WHERE first_payment_received_date IS NULL
  AND first_payment_due_date IS NOT NULL;

-- Test 5: Detailed view of any incorrectly tagged loans
-- Show loans that are incorrectly tagged as FIMR
SELECT 
    'INCORRECT: Early/On-Time Payment Tagged as FIMR' as issue,
    l.loan_id,
    l.customer_name,
    l.disbursement_date,
    l.first_payment_due_date,
    l.first_payment_received_date,
    l.fimr_tagged,
    (l.first_payment_received_date - l.first_payment_due_date) as days_early_or_late
FROM loans l
WHERE (l.first_payment_received_date <= l.first_payment_due_date
       OR l.first_payment_received_date IS NULL)
  AND l.fimr_tagged = TRUE
  AND l.first_payment_due_date IS NOT NULL
LIMIT 10;

-- Test 6: Detailed view of any incorrectly tagged loans (opposite case)
-- Show loans that are NOT tagged as FIMR but should be
SELECT 
    'INCORRECT: Late Payment NOT Tagged as FIMR' as issue,
    l.loan_id,
    l.customer_name,
    l.disbursement_date,
    l.first_payment_due_date,
    l.first_payment_received_date,
    l.fimr_tagged,
    (l.first_payment_received_date - l.first_payment_due_date) as days_late
FROM loans l
WHERE l.first_payment_received_date > l.first_payment_due_date
  AND l.fimr_tagged = FALSE
  AND l.first_payment_due_date IS NOT NULL
LIMIT 10;

-- Test 7: Summary statistics
SELECT 
    'SUMMARY' as category,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as fimr_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as fimr_tagged_false,
    ROUND(100.0 * SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) / COUNT(*), 2) as fimr_percentage
FROM loans
WHERE first_payment_due_date IS NOT NULL;

