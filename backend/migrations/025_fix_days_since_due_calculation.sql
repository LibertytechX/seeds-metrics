-- ============================================================================
-- Migration: 025_fix_days_since_due_calculation.sql
-- Description: Fix days_since_due calculation in loans table
--
-- Current Issue: days_since_due is using current_dpd (Days Past Due)
-- Correct Logic: days_since_due should be the difference between:
--   - first_payment_due_date and first_payment_received_date (if payment received)
--   - first_payment_due_date and CURRENT_DATE (if no payment received)
--
-- This represents how many days have passed since the first payment was due
--
-- Date: 2025-11-05
-- ============================================================================

-- Step 1: Add new column for days_since_due if it doesn't exist
ALTER TABLE loans
ADD COLUMN IF NOT EXISTS days_since_due INTEGER DEFAULT 0;

-- Step 2: Create index for faster queries
CREATE INDEX IF NOT EXISTS idx_loans_days_since_due ON loans(days_since_due);

-- Step 3: Update all loans with correct days_since_due calculation
UPDATE loans l
SET
    days_since_due = CASE
        -- If first payment was received, calculate days between due date and received date
        WHEN l.first_payment_received_date IS NOT NULL AND l.first_payment_due_date IS NOT NULL THEN
            (l.first_payment_received_date - l.first_payment_due_date)::INTEGER
        -- If no payment received but due date exists, calculate days between due date and today
        WHEN l.first_payment_received_date IS NULL AND l.first_payment_due_date IS NOT NULL THEN
            (CURRENT_DATE - l.first_payment_due_date)::INTEGER
        -- If no due date, cannot calculate
        ELSE 0
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE first_payment_due_date IS NOT NULL;

-- Step 4: Show statistics
DO $$
BEGIN
    RAISE NOTICE '=== DAYS SINCE DUE CALCULATION RESULTS ===';
END $$;

SELECT 
    'Payment Received - Early' as category,
    COUNT(*) as total_loans,
    MIN(days_since_due) as min_days,
    MAX(days_since_due) as max_days,
    ROUND(AVG(days_since_due), 2) as avg_days
FROM loans
WHERE first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL
  AND days_since_due < 0

UNION ALL

SELECT 
    'Payment Received - On-Time' as category,
    COUNT(*) as total_loans,
    MIN(days_since_due) as min_days,
    MAX(days_since_due) as max_days,
    ROUND(AVG(days_since_due), 2) as avg_days
FROM loans
WHERE first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL
  AND days_since_due = 0

UNION ALL

SELECT 
    'Payment Received - Late' as category,
    COUNT(*) as total_loans,
    MIN(days_since_due) as min_days,
    MAX(days_since_due) as max_days,
    ROUND(AVG(days_since_due), 2) as avg_days
FROM loans
WHERE first_payment_received_date IS NOT NULL
  AND first_payment_due_date IS NOT NULL
  AND days_since_due > 0

UNION ALL

SELECT 
    'No Payment - Days Since Due' as category,
    COUNT(*) as total_loans,
    MIN(days_since_due) as min_days,
    MAX(days_since_due) as max_days,
    ROUND(AVG(days_since_due), 2) as avg_days
FROM loans
WHERE first_payment_received_date IS NULL
  AND first_payment_due_date IS NOT NULL
  AND days_since_due > 0

UNION ALL

SELECT 
    'No Payment - Due Date Future' as category,
    COUNT(*) as total_loans,
    MIN(days_since_due) as min_days,
    MAX(days_since_due) as max_days,
    ROUND(AVG(days_since_due), 2) as avg_days
FROM loans
WHERE first_payment_received_date IS NULL
  AND first_payment_due_date IS NOT NULL
  AND days_since_due <= 0;

-- Step 5: Show sample loans with different scenarios
DO $$
BEGIN
    RAISE NOTICE '=== SAMPLE LOANS WITH DAYS_SINCE_DUE ===';
END $$;

SELECT 
    l.loan_id,
    l.customer_name,
    l.first_payment_due_date,
    l.first_payment_received_date,
    l.days_since_due,
    CASE
        WHEN l.first_payment_received_date IS NOT NULL AND l.days_since_due < 0 THEN 'Early Payment'
        WHEN l.first_payment_received_date IS NOT NULL AND l.days_since_due = 0 THEN 'On-Time Payment'
        WHEN l.first_payment_received_date IS NOT NULL AND l.days_since_due > 0 THEN 'Late Payment'
        WHEN l.first_payment_received_date IS NULL AND l.days_since_due > 0 THEN 'No Payment - Overdue'
        WHEN l.first_payment_received_date IS NULL AND l.days_since_due <= 0 THEN 'No Payment - Not Yet Due'
    END as payment_status
FROM loans l
WHERE l.first_payment_due_date IS NOT NULL
ORDER BY l.days_since_due DESC
LIMIT 20;

-- Step 6: Update trigger function to maintain days_since_due on new repayments
CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_total_principal_paid DECIMAL(15, 2);
    v_total_interest_paid DECIMAL(15, 2);
    v_total_fees_paid DECIMAL(15, 2);
    v_total_repayments DECIMAL(15, 2);
    v_first_payment_date DATE;
    v_last_payment_date DATE;
    v_first_due_date DATE;
    v_current_dpd INTEGER;
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_disbursement_date DATE;
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_max_dpd INTEGER;
    v_repayment_count INTEGER;
    v_days_since_last_repayment INTEGER;
    v_total_outstanding DECIMAL(15, 2);
    v_schedule_count INTEGER;
    v_payment_on_due_date_exists BOOLEAN;
    v_days_since_due INTEGER;
BEGIN
    v_loan_id := NEW.loan_id;

    -- Get loan details
    SELECT
        loan_amount, interest_rate, disbursement_date, loan_term_days, fee_amount
    INTO
        v_loan_amount, v_interest_rate, v_disbursement_date, v_loan_term_days, v_fee_amount
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Calculate totals from repayments
    SELECT
        COALESCE(SUM(principal_paid), 0),
        COALESCE(SUM(interest_paid), 0),
        COALESCE(SUM(fees_paid), 0),
        COALESCE(SUM(payment_amount), 0),
        COUNT(*),
        MIN(payment_date),
        MAX(payment_date)
    INTO
        v_total_principal_paid, v_total_interest_paid, v_total_fees_paid,
        v_total_repayments, v_repayment_count, v_first_payment_date, v_last_payment_date
    FROM repayments
    WHERE loan_id = v_loan_id AND is_reversed = FALSE;

    -- Calculate days since last repayment
    IF v_last_payment_date IS NOT NULL THEN
        v_days_since_last_repayment := CURRENT_DATE - v_last_payment_date;
    ELSE
        v_days_since_last_repayment := NULL;
    END IF;

    -- Get first due date from loan_schedule
    SELECT MIN(due_date) INTO v_first_due_date
    FROM loan_schedule
    WHERE loan_id = v_loan_id;
    
    -- If no schedule exists, use first_payment_due_date from loans table
    IF v_first_due_date IS NULL THEN
        SELECT first_payment_due_date INTO v_first_due_date
        FROM loans
        WHERE loan_id = v_loan_id;
    END IF;

    -- Calculate days_since_due
    IF v_first_payment_date IS NOT NULL AND v_first_due_date IS NOT NULL THEN
        v_days_since_due := (v_first_payment_date - v_first_due_date)::INTEGER;
    ELSIF v_first_payment_date IS NULL AND v_first_due_date IS NOT NULL THEN
        v_days_since_due := (CURRENT_DATE - v_first_due_date)::INTEGER;
    ELSE
        v_days_since_due := 0;
    END IF;

    -- Calculate current DPD
    v_current_dpd := CASE
        WHEN v_last_payment_date IS NOT NULL THEN
            GREATEST(0, (CURRENT_DATE - v_last_payment_date)::INTEGER)
        ELSE
            GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)
    END;

    -- Get max DPD ever
    SELECT COALESCE(MAX(dpd_at_payment), 0) INTO v_max_dpd
    FROM repayments
    WHERE loan_id = v_loan_id;

    -- Check if payment exists on or before first_payment_due_date
    SELECT EXISTS (
        SELECT 1
        FROM repayments
        WHERE loan_id = v_loan_id
          AND payment_date <= v_first_due_date
          AND is_reversed = FALSE
    ) INTO v_payment_on_due_date_exists;

    -- Calculate outstanding balances
    v_total_outstanding := GREATEST(0, v_loan_amount - v_total_principal_paid);

    -- Update loans table with computed values
    UPDATE loans
    SET
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        total_repayments = v_total_repayments,
        principal_outstanding = GREATEST(0, v_loan_amount - v_total_principal_paid),
        interest_outstanding = GREATEST(0, (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid),
        fees_outstanding = GREATEST(0, v_fee_amount - v_total_fees_paid),
        total_outstanding = v_total_outstanding,
        first_payment_received_date = v_first_payment_date,
        first_payment_due_date = v_first_due_date,
        first_payment_missed = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),
        current_dpd = v_current_dpd,
        max_dpd_ever = GREATEST(v_max_dpd, v_current_dpd),
        days_since_due = v_days_since_due,
        days_since_last_repayment = v_days_since_last_repayment,
        loan_age = CASE
            WHEN v_disbursement_date IS NULL THEN 0
            ELSE (CURRENT_DATE - v_disbursement_date)::INTEGER
        END,
        fimr_tagged = CASE
            WHEN v_first_due_date IS NULL THEN TRUE
            WHEN v_payment_on_due_date_exists THEN FALSE
            WHEN v_first_payment_date IS NULL AND v_first_due_date >= CURRENT_DATE THEN FALSE
            ELSE TRUE
        END,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),
        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 7: Recreate trigger
DROP TRIGGER IF EXISTS update_loan_computed_fields_trigger ON repayments;
CREATE TRIGGER update_loan_computed_fields_trigger
AFTER INSERT OR UPDATE ON repayments
FOR EACH ROW EXECUTE FUNCTION update_loan_computed_fields();

