-- ============================================================================
-- Migration: 023_fix_fimr_for_future_due_dates.sql
-- Description: Fix FIMR logic for loans with no payments and future due dates
--
-- Issue: Loans with no payments but future due dates were incorrectly tagged as FIMR
--        FIMR should only apply to loans where the due date has PASSED
--
-- Correct Logic for Loans with No Payments:
-- - FIMR = TRUE if first_payment_due_date < CURRENT_DATE (due date has passed)
-- - FIMR = FALSE if first_payment_due_date >= CURRENT_DATE (due date is today or future)
--
-- Date: 2025-11-05
-- ============================================================================

-- Step 1: Show statistics BEFORE the fix
DO $$
BEGIN
    RAISE NOTICE '=== FIMR STATISTICS FOR NO-PAYMENT LOANS BEFORE MIGRATION ===';
END $$;

SELECT 
    'No Payments - Due Date Passed' as category,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as correctly_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as incorrectly_tagged_false
FROM loans
WHERE first_payment_received_date IS NULL
  AND first_payment_due_date < CURRENT_DATE

UNION ALL

SELECT 
    'No Payments - Due Date Future' as category,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as incorrectly_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as correctly_tagged_false
FROM loans
WHERE first_payment_received_date IS NULL
  AND first_payment_due_date >= CURRENT_DATE;

-- Step 2: Fix FIMR for loans with no payments
-- For loans with no payments, FIMR should be TRUE only if due date has passed
UPDATE loans l
SET
    fimr_tagged = CASE
        WHEN l.first_payment_received_date IS NOT NULL THEN
            -- Loan has payments: check if any payment on or before due date
            CASE
                WHEN l.first_payment_due_date IS NULL THEN TRUE
                WHEN EXISTS (
                    SELECT 1
                    FROM repayments r
                    WHERE r.loan_id = l.loan_id
                      AND r.payment_date <= l.first_payment_due_date
                      AND r.is_reversed = FALSE
                ) THEN FALSE
                ELSE TRUE
            END
        ELSE
            -- Loan has NO payments: check if due date has passed
            CASE
                WHEN l.first_payment_due_date IS NULL THEN TRUE
                WHEN l.first_payment_due_date < CURRENT_DATE THEN TRUE  -- Due date passed, no payment = FIMR
                ELSE FALSE  -- Due date not yet passed, not FIMR
            END
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE first_payment_due_date IS NOT NULL;

-- Step 3: Show statistics AFTER the fix
DO $$
BEGIN
    RAISE NOTICE '=== FIMR STATISTICS FOR NO-PAYMENT LOANS AFTER MIGRATION ===';
END $$;

SELECT 
    'No Payments - Due Date Passed' as category,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as correctly_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as incorrectly_tagged_false
FROM loans
WHERE first_payment_received_date IS NULL
  AND first_payment_due_date < CURRENT_DATE

UNION ALL

SELECT 
    'No Payments - Due Date Future' as category,
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as incorrectly_tagged_true,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as correctly_tagged_false
FROM loans
WHERE first_payment_received_date IS NULL
  AND first_payment_due_date >= CURRENT_DATE;

-- Step 4: Overall FIMR statistics
DO $$
BEGIN
    RAISE NOTICE '=== OVERALL FIMR STATISTICS AFTER MIGRATION ===';
END $$;

SELECT 
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as fimr_true_count,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as fimr_false_count,
    ROUND(100.0 * SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) / COUNT(*), 2) as fimr_true_percentage
FROM loans
WHERE first_payment_due_date IS NOT NULL;

