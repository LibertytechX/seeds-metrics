-- Migration: Add loan_age column to loans table
-- 
-- loan_age: Total days since the loan was disbursed
-- This is a computed field that calculates: CURRENT_DATE - disbursement_date
-- 
-- This field will be calculated by the trigger function whenever repayments are posted

-- Step 1: Add loan_age column to loans table
ALTER TABLE loans
ADD COLUMN loan_age INTEGER DEFAULT 0;

-- Step 2: Calculate loan_age for all existing loans
UPDATE loans
SET loan_age = (CURRENT_DATE - disbursement_date::date)::INTEGER,
    updated_at = CURRENT_TIMESTAMP
WHERE disbursement_date IS NOT NULL;

-- Step 3: Update the trigger function to calculate loan_age
DROP TRIGGER IF EXISTS trigger_update_loan_computed_fields ON repayments;
DROP FUNCTION IF EXISTS update_loan_computed_fields() CASCADE;

CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_disbursement_date DATE;
    v_first_due_date DATE;
    v_total_principal_paid DECIMAL(15, 2);
    v_total_interest_paid DECIMAL(15, 2);
    v_total_fees_paid DECIMAL(15, 2);
    v_total_repayments DECIMAL(15, 2);
    v_loan_amount DECIMAL(15, 2);
    v_repayment_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_fee_amount DECIMAL(15, 2);
    v_first_payment_date TIMESTAMP;
    v_last_payment_date TIMESTAMP;
    v_repayment_count INTEGER;
    v_current_dpd INTEGER;
    v_max_dpd_ever INTEGER;
BEGIN
    -- Get the loan_id from the repayment record
    IF TG_OP = 'INSERT' THEN
        v_loan_id := NEW.loan_id;
    ELSIF TG_OP = 'UPDATE' THEN
        v_loan_id := NEW.loan_id;
    ELSIF TG_OP = 'DELETE' THEN
        v_loan_id := OLD.loan_id;
    END IF;

    -- Get loan details
    SELECT 
        disbursement_date,
        loan_amount,
        repayment_amount,
        interest_rate,
        fee_amount
    INTO 
        v_disbursement_date,
        v_loan_amount,
        v_repayment_amount,
        v_interest_rate,
        v_fee_amount
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Calculate total repayments
    SELECT
        COALESCE(SUM(principal_amount), 0),
        COALESCE(SUM(interest_amount), 0),
        COALESCE(SUM(fees_amount), 0),
        COALESCE(SUM(total_amount), 0),
        MIN(paid_date),
        MAX(paid_date),
        COUNT(*)
    INTO
        v_total_principal_paid,
        v_total_interest_paid,
        v_total_fees_paid,
        v_total_repayments,
        v_first_payment_date,
        v_last_payment_date,
        v_repayment_count
    FROM repayments
    WHERE loan_id = v_loan_id;

    -- Get first due date from loan_schedule (if available)
    SELECT MIN(due_date)
    INTO v_first_due_date
    FROM loan_schedule
    WHERE loan_id = v_loan_id;

    -- If no schedule, estimate first due date as disbursement + 30 days
    IF v_first_due_date IS NULL AND v_disbursement_date IS NOT NULL THEN
        v_first_due_date := v_disbursement_date + INTERVAL '30 days';
    END IF;

    -- Calculate current DPD (Days Past Due)
    -- DPD = days since the earliest unpaid installment's due date
    -- For now, we'll use a simplified calculation based on loan age and repayment progress
    IF v_disbursement_date IS NOT NULL THEN
        -- Simple DPD calculation: if total paid < expected based on loan age
        v_current_dpd := GREATEST(0, 
            (CURRENT_DATE - v_disbursement_date)::INTEGER - 
            COALESCE((v_total_repayments / NULLIF(v_repayment_amount, 0) * 30)::INTEGER, 0)
        );
    ELSE
        v_current_dpd := 0;
    END IF;

    -- Get max DPD ever (keep existing value if higher)
    SELECT COALESCE(max_dpd_ever, 0)
    INTO v_max_dpd_ever
    FROM loans
    WHERE loan_id = v_loan_id;

    v_max_dpd_ever := GREATEST(v_max_dpd_ever, v_current_dpd);

    -- Update the loan with computed fields
    UPDATE loans
    SET
        -- Payment totals
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        total_repayments = v_total_repayments,
        
        -- Outstanding balances
        principal_outstanding = GREATEST(0, v_loan_amount - v_total_principal_paid),
        interest_outstanding = GREATEST(0, 
            COALESCE((v_loan_amount * v_interest_rate), 0) - v_total_interest_paid
        ),
        fees_outstanding = GREATEST(0, COALESCE(v_fee_amount, 0) - v_total_fees_paid),
        total_outstanding = GREATEST(0,
            (v_loan_amount - v_total_principal_paid) +
            (COALESCE((v_loan_amount * v_interest_rate), 0) - v_total_interest_paid) +
            (COALESCE(v_fee_amount, 0) - v_total_fees_paid)
        ),
        
        -- DPD metrics
        current_dpd = v_current_dpd,
        max_dpd_ever = v_max_dpd_ever,
        
        -- Payment dates
        first_payment_due_date = v_first_due_date,
        first_payment_received_date = v_first_payment_date,
        
        -- Risk indicators
        -- FIMR (First Installment Missed and Recovered): TRUE if NO repayments within first 7 days after disbursement
        fimr_tagged = CASE
            WHEN v_first_payment_date IS NULL THEN TRUE  -- No payment received yet
            WHEN v_disbursement_date IS NULL THEN FALSE  -- No disbursement date available
            WHEN (v_first_payment_date - v_disbursement_date) > INTERVAL '7 days' THEN TRUE  -- First payment was more than 7 days after disbursement
            ELSE FALSE  -- First payment was within 7 days
        END,
        
        -- Early indicator: TRUE if first payment was made before due date
        early_indicator_tagged = CASE
            WHEN v_first_payment_date IS NULL THEN FALSE
            WHEN v_first_due_date IS NULL THEN FALSE
            WHEN v_first_payment_date < v_first_due_date THEN TRUE
            ELSE FALSE
        END,
        
        -- Days since last repayment
        days_since_last_repayment = CASE
            WHEN v_last_payment_date IS NULL THEN NULL
            ELSE (CURRENT_DATE - v_last_payment_date::date)::INTEGER
        END,
        
        -- Loan age: Days since disbursement
        loan_age = CASE
            WHEN v_disbursement_date IS NULL THEN 0
            ELSE (CURRENT_DATE - v_disbursement_date)::INTEGER
        END,
        
        -- Repayment count
        repayment_count = v_repayment_count,
        
        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 4: Recreate the trigger
CREATE TRIGGER trigger_update_loan_computed_fields
    AFTER INSERT OR UPDATE ON repayments
    FOR EACH ROW
    EXECUTE FUNCTION update_loan_computed_fields();

-- Step 5: Show summary
DO $$
DECLARE
    v_total_loans INTEGER;
    v_avg_loan_age DECIMAL(10, 2);
    v_min_loan_age INTEGER;
    v_max_loan_age INTEGER;
BEGIN
    SELECT 
        COUNT(*),
        ROUND(AVG(loan_age)::numeric, 2),
        MIN(loan_age),
        MAX(loan_age)
    INTO 
        v_total_loans,
        v_avg_loan_age,
        v_min_loan_age,
        v_max_loan_age
    FROM loans
    WHERE disbursement_date IS NOT NULL;
    
    RAISE NOTICE '=== Loan Age Column Added ===';
    RAISE NOTICE 'Total loans: %', v_total_loans;
    RAISE NOTICE 'Average loan age: % days', v_avg_loan_age;
    RAISE NOTICE 'Min loan age: % days', v_min_loan_age;
    RAISE NOTICE 'Max loan age: % days', v_max_loan_age;
END $$;

