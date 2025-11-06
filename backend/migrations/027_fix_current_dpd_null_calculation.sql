-- ============================================================================
-- Migration: 027_fix_current_dpd_null_calculation.sql
-- Description: Fix current_dpd calculation to handle NULL first_payment_due_date
--
-- Issue: The trigger function in Migration 025 has a bug where current_dpd
-- becomes NULL when first_payment_due_date is NULL. This happens because:
--   v_current_dpd := GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)
-- When v_first_due_date is NULL, the result is NULL.
--
-- Fix: Add NULL check to default to 0 when no due date is available
--
-- Date: 2025-11-06
-- ============================================================================

-- Step 1: Update the trigger function to handle NULL first_payment_due_date
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

    -- Calculate current DPD with proper NULL handling
    -- FIX: Added NULL check for v_first_due_date to prevent NULL results
    v_current_dpd := CASE
        WHEN v_first_due_date IS NULL THEN 0  -- No due date = not overdue
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

-- Step 2: Verify the fix by checking for any NULL current_dpd values
DO $$
BEGIN
    RAISE NOTICE '=== CURRENT_DPD FIX VERIFICATION ===';
    RAISE NOTICE 'Checking for NULL current_dpd values...';
END $$;

SELECT 
    COUNT(*) as total_loans,
    COUNT(CASE WHEN current_dpd IS NULL THEN 1 END) as null_dpd_count,
    COUNT(CASE WHEN current_dpd IS NOT NULL THEN 1 END) as valid_dpd_count
FROM loans;

-- Step 3: Show distribution of current_dpd values
SELECT 
    CASE 
        WHEN current_dpd IS NULL THEN 'NULL'
        WHEN current_dpd = 0 THEN '0 (Current)'
        WHEN current_dpd BETWEEN 1 AND 6 THEN '1-6 (Early Indicator)'
        WHEN current_dpd BETWEEN 7 AND 15 THEN '7-15 (Overdue)'
        WHEN current_dpd > 15 THEN '>15 (Severely Overdue)'
    END as dpd_category,
    COUNT(*) as count
FROM loans
GROUP BY dpd_category
ORDER BY count DESC;

-- Step 4: Show sample loans with their current_dpd values
SELECT 
    loan_id,
    customer_name,
    status,
    first_payment_due_date,
    first_payment_received_date,
    current_dpd,
    days_since_due
FROM loans
ORDER BY current_dpd DESC
LIMIT 20;

