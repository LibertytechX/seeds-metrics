-- ============================================================================
-- Migration: 022_recalculate_fimr_correct_logic.sql
-- Description: Recalculate FIMR tags for ALL loans using the correct logic
--
-- Issue: Previous migrations used incorrect FIMR logic that compared
--        first_payment_received_date > first_payment_due_date
--        This missed cases where there were NO payments on or before the due date
--
-- Correct Logic: FIMR = TRUE if NO repayment exists on or before first_payment_due_date
--
-- Date: 2025-11-05
-- ============================================================================

-- Step 1: Show FIMR statistics BEFORE the fix
DO $$
BEGIN
    RAISE NOTICE '=== FIMR STATISTICS BEFORE MIGRATION ===';
END $$;

SELECT 
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as fimr_true_count,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as fimr_false_count,
    ROUND(100.0 * SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) / COUNT(*), 2) as fimr_true_percentage
FROM loans
WHERE first_payment_due_date IS NOT NULL;

-- Step 2: Recalculate FIMR for ALL loans using the correct logic
-- FIMR = TRUE if NO repayment on or before first_payment_due_date
UPDATE loans l
SET
    fimr_tagged = CASE
        WHEN l.first_payment_due_date IS NULL THEN TRUE
        WHEN EXISTS (
            SELECT 1
            FROM repayments r
            WHERE r.loan_id = l.loan_id
              AND r.payment_date <= l.first_payment_due_date
              AND r.is_reversed = FALSE
        ) THEN FALSE
        ELSE TRUE
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE first_payment_due_date IS NOT NULL;

-- Step 3: Show FIMR statistics AFTER the fix
DO $$
BEGIN
    RAISE NOTICE '=== FIMR STATISTICS AFTER MIGRATION ===';
END $$;

SELECT 
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as fimr_true_count,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as fimr_false_count,
    ROUND(100.0 * SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) / COUNT(*), 2) as fimr_true_percentage
FROM loans
WHERE first_payment_due_date IS NOT NULL;

-- Step 4: Verify no loans with early/on-time payments are tagged as FIMR
DO $$
BEGIN
    RAISE NOTICE '=== VERIFICATION: Early/On-Time Payments Tagged as FIMR ===';
END $$;

SELECT 
    COUNT(*) as incorrectly_tagged_count
FROM loans l
WHERE (l.first_payment_received_date <= l.first_payment_due_date
       OR l.first_payment_received_date IS NULL)
  AND l.fimr_tagged = TRUE
  AND l.first_payment_due_date IS NOT NULL;

-- Step 5: Verify no loans with late payments are NOT tagged as FIMR
DO $$
BEGIN
    RAISE NOTICE '=== VERIFICATION: Late Payments NOT Tagged as FIMR ===';
END $$;

SELECT 
    COUNT(*) as incorrectly_tagged_count
FROM loans l
WHERE l.first_payment_received_date > l.first_payment_due_date
  AND l.fimr_tagged = FALSE
  AND l.first_payment_due_date IS NOT NULL;

-- Step 6: Show breakdown by payment status
DO $$
BEGIN
    RAISE NOTICE '=== BREAKDOWN BY PAYMENT STATUS ===';
END $$;

SELECT 
    'Early Payments' as payment_status,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as correctly_tagged_false,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as incorrectly_tagged_true
FROM loans l
WHERE first_payment_received_date < first_payment_due_date
  AND first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL

UNION ALL

SELECT 
    'On-Time Payments' as payment_status,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as correctly_tagged_false,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as incorrectly_tagged_true
FROM loans l
WHERE first_payment_received_date = first_payment_due_date
  AND first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL

UNION ALL

SELECT 
    'Late Payments' as payment_status,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as correctly_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as incorrectly_tagged_false
FROM loans l
WHERE first_payment_received_date > first_payment_due_date
  AND first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL

UNION ALL

SELECT 
    'No Payments' as payment_status,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as correctly_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as incorrectly_tagged_false
FROM loans l
WHERE first_payment_received_date IS NULL
  AND first_payment_due_date IS NOT NULL;

