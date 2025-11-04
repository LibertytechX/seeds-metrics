-- Migration: Fix FIMR (First Installment Missed and Recovered) Logic
-- 
-- Original Definition: FIMR = TRUE if NO repayments within first 7 days after disbursement
-- 
-- This migration:
-- 1. Drops the existing trigger function
-- 2. Recreates it with the correct FIMR logic
-- 3. Recalculates FIMR tags for all existing loans

-- Step 1: Drop existing trigger and function
DROP TRIGGER IF EXISTS trigger_update_loan_computed_fields ON repayments;
DROP FUNCTION IF EXISTS update_loan_computed_fields() CASCADE;

-- Step 2: Recreate the trigger function with correct FIMR logic
CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_repayment_count INTEGER;
    v_total_principal_paid DECIMAL(15, 2);
    v_total_interest_paid DECIMAL(15, 2);
    v_total_fees_paid DECIMAL(15, 2);
    v_total_repayments DECIMAL(15, 2);
    v_first_payment_date DATE;
    v_last_payment_date DATE;
    v_first_due_date DATE;
    v_disbursement_date DATE;
    v_current_dpd INTEGER;
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_maturity_date DATE;
    v_status VARCHAR(20);
    v_max_dpd INTEGER;
    v_total_outstanding DECIMAL(15, 2);
    v_days_since_last_repayment INTEGER;
BEGIN
    -- Get the loan_id from the inserted/updated repayment
    v_loan_id := NEW.loan_id;

    -- Get loan details
    SELECT 
        loan_amount, 
        interest_rate, 
        loan_term_days, 
        fee_amount, 
        disbursement_date, 
        maturity_date, 
        status,
        COALESCE(max_dpd_ever, 0)
    INTO 
        v_loan_amount, 
        v_interest_rate, 
        v_loan_term_days, 
        v_fee_amount, 
        v_disbursement_date, 
        v_maturity_date, 
        v_status,
        v_max_dpd
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Calculate repayment totals
    SELECT 
        COUNT(*),
        COALESCE(SUM(principal_paid), 0),
        COALESCE(SUM(interest_paid), 0),
        COALESCE(SUM(fees_paid), 0),
        COALESCE(SUM(payment_amount), 0),
        MIN(payment_date),
        MAX(payment_date)
    INTO 
        v_repayment_count,
        v_total_principal_paid,
        v_total_interest_paid,
        v_total_fees_paid,
        v_total_repayments,
        v_first_payment_date,
        v_last_payment_date
    FROM repayments
    WHERE loan_id = v_loan_id;

    -- Calculate total outstanding
    v_total_outstanding := (v_loan_amount - v_total_principal_paid) +
                          ((v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid) +
                          (v_fee_amount - v_total_fees_paid);

    -- Get first due date from loan_schedule
    SELECT MIN(due_date) INTO v_first_due_date
    FROM loan_schedule
    WHERE loan_id = v_loan_id;

    -- If no schedule exists, calculate first due date as 30 days after disbursement
    IF v_first_due_date IS NULL THEN
        v_first_due_date := v_disbursement_date + INTERVAL '30 days';
    END IF;

    -- Calculate days since last repayment
    IF v_last_payment_date IS NOT NULL THEN
        v_days_since_last_repayment := CURRENT_DATE - v_last_payment_date;
    ELSE
        v_days_since_last_repayment := 0;
    END IF;

    -- Calculate current DPD
    IF v_status = 'Closed' THEN
        v_current_dpd := 0;
    ELSIF v_maturity_date IS NOT NULL AND CURRENT_DATE > v_maturity_date THEN
        v_current_dpd := CURRENT_DATE - v_maturity_date;
    ELSE
        v_current_dpd := 0;
    END IF;

    -- Update the loan with computed fields
    UPDATE loans
    SET
        -- Repayment totals
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        total_repayments = v_total_repayments,
        repayment_count = v_repayment_count,

        -- Outstanding balances
        principal_outstanding = v_loan_amount - v_total_principal_paid,
        interest_outstanding = (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid,
        fees_outstanding = v_fee_amount - v_total_fees_paid,
        total_outstanding = v_total_outstanding,

        -- First payment tracking
        first_payment_received_date = v_first_payment_date,
        first_payment_due_date = v_first_due_date,
        first_payment_missed = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),

        -- DPD tracking
        current_dpd = v_current_dpd,
        max_dpd_ever = GREATEST(v_max_dpd, v_current_dpd),

        -- Risk indicators
        -- FIMR (First Installment Missed and Recovered): TRUE if NO repayments within first 7 days after disbursement
        fimr_tagged = CASE
            WHEN v_first_payment_date IS NULL THEN TRUE  -- No payment received yet
            WHEN v_disbursement_date IS NULL THEN FALSE  -- No disbursement date available
            WHEN (v_first_payment_date - v_disbursement_date) > 7 THEN TRUE  -- First payment was more than 7 days after disbursement
            ELSE FALSE  -- First payment was within 7 days
        END,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),

        -- Days since last repayment
        days_since_last_repayment = v_days_since_last_repayment,

        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 3: Recreate the trigger
CREATE TRIGGER trigger_update_loan_computed_fields
    AFTER INSERT OR UPDATE ON repayments
    FOR EACH ROW
    EXECUTE FUNCTION update_loan_computed_fields();

-- Step 4: Recalculate FIMR tags for all existing loans
-- This updates all loans based on their actual first payment date vs disbursement date
UPDATE loans
SET fimr_tagged = CASE
    WHEN first_payment_received_date IS NULL THEN TRUE  -- No payment received yet
    WHEN disbursement_date IS NULL THEN FALSE  -- No disbursement date available
    WHEN (first_payment_received_date - disbursement_date) > 7 THEN TRUE  -- First payment was more than 7 days after disbursement
    ELSE FALSE  -- First payment was within 7 days
END,
updated_at = CURRENT_TIMESTAMP;

-- Step 5: Show summary of FIMR tag changes
DO $$
DECLARE
    v_total_loans INTEGER;
    v_fimr_tagged_count INTEGER;
    v_fimr_percentage DECIMAL(5, 2);
BEGIN
    SELECT COUNT(*) INTO v_total_loans FROM loans;
    SELECT COUNT(*) INTO v_fimr_tagged_count FROM loans WHERE fimr_tagged = TRUE;
    v_fimr_percentage := (v_fimr_tagged_count::DECIMAL / NULLIF(v_total_loans, 0) * 100);
    
    RAISE NOTICE '=== FIMR Recalculation Summary ===';
    RAISE NOTICE 'Total loans: %', v_total_loans;
    RAISE NOTICE 'FIMR tagged loans: %', v_fimr_tagged_count;
    RAISE NOTICE 'FIMR percentage: %%%', ROUND(v_fimr_percentage, 2);
END $$;

