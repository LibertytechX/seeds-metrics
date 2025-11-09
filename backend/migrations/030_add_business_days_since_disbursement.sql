-- ============================================================================
-- Migration 030: Add business_days_since_disbursement Field
-- ============================================================================
-- Description: Adds a new computed field to track total business days elapsed
--              from loan disbursement date to today. This is different from
--              repayment_days_due_today which starts counting from
--              first_payment_due_date.
--
-- Author: Seeds Metrics Team
-- Date: 2025-11-09
-- ============================================================================

BEGIN;

-- ============================================================================
-- STEP 1: Add new column to loans table
-- ============================================================================

ALTER TABLE loans 
ADD COLUMN IF NOT EXISTS business_days_since_disbursement INTEGER DEFAULT 0;

COMMENT ON COLUMN loans.business_days_since_disbursement IS 
'Total business days (Mon-Fri) elapsed from disbursement_date to today. This shows the loan age in business days.';

-- ============================================================================
-- STEP 2: Update trigger function to calculate business_days_since_disbursement
-- ============================================================================

CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_repayment_amount DECIMAL(15, 2);
    v_disbursement_date DATE;
    v_first_due_date DATE;
    v_maturity_date DATE;
    v_max_dpd_ever INTEGER;
    
    v_total_principal_paid DECIMAL(15, 2);
    v_total_interest_paid DECIMAL(15, 2);
    v_total_fees_paid DECIMAL(15, 2);
    v_total_repayments DECIMAL(15, 2);
    v_first_payment_date DATE;
    v_last_payment_date DATE;
    v_repayment_count INTEGER;
    
    v_principal_outstanding DECIMAL(15, 2);
    v_interest_outstanding DECIMAL(15, 2);
    v_fees_outstanding DECIMAL(15, 2);
    v_total_outstanding DECIMAL(15, 2);
    
    v_days_since_last_repayment INTEGER;
    v_days_since_due INTEGER;
    v_loan_age INTEGER;
    v_current_dpd INTEGER;
    v_fimr_tagged BOOLEAN;
    v_repayment_delay_rate DECIMAL(5, 2);
    
    -- New DPD methodology variables
    v_weekend_days_in_tenure INTEGER;
    v_real_loan_tenure_days INTEGER;
    v_daily_repayment_amount DECIMAL(15, 2);
    v_repayment_days_paid DECIMAL(10, 2);
    v_repayment_days_due_today INTEGER;
    v_calculation_end_date DATE;
    v_business_days_since_disbursement INTEGER;
BEGIN
    -- Get loan_id from the repayment record
    v_loan_id := NEW.loan_id;
    
    -- Fetch loan details
    SELECT 
        loan_amount, interest_rate, loan_term_days, fee_amount, repayment_amount,
        disbursement_date, first_payment_due_date, maturity_date, max_dpd_ever
    INTO 
        v_loan_amount, v_interest_rate, v_loan_term_days, v_fee_amount, v_repayment_amount,
        v_disbursement_date, v_first_due_date, v_maturity_date, v_max_dpd_ever
    FROM loans
    WHERE loan_id = v_loan_id;
    
    -- Aggregate repayment data (excluding reversed repayments)
    SELECT 
        COALESCE(SUM(principal_paid), 0),
        COALESCE(SUM(interest_paid), 0),
        COALESCE(SUM(fees_paid), 0),
        COALESCE(SUM(payment_amount), 0),
        MIN(payment_date),
        MAX(payment_date),
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
    WHERE loan_id = v_loan_id AND is_reversed = FALSE;
    
    -- Calculate outstanding balances
    v_principal_outstanding := v_loan_amount - v_total_principal_paid;
    v_interest_outstanding := (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid;
    v_fees_outstanding := COALESCE(v_fee_amount, 0) - v_total_fees_paid;
    v_total_outstanding := v_principal_outstanding + v_interest_outstanding + v_fees_outstanding;
    
    -- Calculate days since last repayment
    IF v_last_payment_date IS NOT NULL THEN
        v_days_since_last_repayment := (CURRENT_DATE - v_last_payment_date)::INTEGER;
    ELSE
        v_days_since_last_repayment := NULL;
    END IF;
    
    -- Calculate days since due
    IF v_first_payment_date IS NOT NULL AND v_first_due_date IS NOT NULL THEN
        v_days_since_due := (v_first_payment_date - v_first_due_date)::INTEGER;
    ELSIF v_first_payment_date IS NULL AND v_first_due_date IS NOT NULL THEN
        v_days_since_due := (CURRENT_DATE - v_first_due_date)::INTEGER;
    ELSE
        v_days_since_due := 0;
    END IF;
    
    -- Calculate loan age
    v_loan_age := (CURRENT_DATE - v_disbursement_date)::INTEGER;
    
    -- ========================================================================
    -- NEW DPD CALCULATION METHODOLOGY
    -- ========================================================================
    
    -- Step 1: Calculate real loan tenure (including weekends)
    IF v_first_due_date IS NOT NULL AND v_maturity_date IS NOT NULL THEN
        v_weekend_days_in_tenure := count_weekend_days(v_first_due_date, v_maturity_date);
        v_real_loan_tenure_days := v_loan_term_days + v_weekend_days_in_tenure;
    ELSE
        v_real_loan_tenure_days := v_loan_term_days;
    END IF;
    
    -- Step 2: Calculate daily repayment amount (based on business days only)
    IF v_loan_term_days > 0 AND v_repayment_amount > 0 THEN
        v_daily_repayment_amount := v_repayment_amount / v_loan_term_days;
    ELSE
        v_daily_repayment_amount := 0;
    END IF;
    
    -- Step 3: Calculate repayment days paid
    IF v_daily_repayment_amount > 0 THEN
        v_repayment_days_paid := v_total_repayments / v_daily_repayment_amount;
    ELSE
        v_repayment_days_paid := 0;
    END IF;
    
    -- Step 4: Calculate repayment days due today (from first_payment_due_date)
    IF v_first_due_date IS NOT NULL THEN
        v_calculation_end_date := LEAST(CURRENT_DATE, COALESCE(v_maturity_date, CURRENT_DATE));
        
        IF v_calculation_end_date >= v_first_due_date THEN
            v_repayment_days_due_today := count_business_days(v_first_due_date, v_calculation_end_date);
        ELSE
            v_repayment_days_due_today := 0;
        END IF;
    ELSE
        v_repayment_days_due_today := 0;
    END IF;
    
    -- Step 5: Calculate business days since disbursement (NEW FIELD)
    IF v_disbursement_date IS NOT NULL THEN
        v_business_days_since_disbursement := count_business_days(v_disbursement_date, CURRENT_DATE);
    ELSE
        v_business_days_since_disbursement := 0;
    END IF;
    
    -- Step 6: Calculate DPD (missed repayment days)
    v_current_dpd := GREATEST(0, v_repayment_days_due_today - v_repayment_days_paid::INTEGER);
    
    -- Calculate FIMR (First Installment Missed Rate)
    IF v_first_due_date IS NULL THEN
        v_fimr_tagged := TRUE;
    ELSIF EXISTS (
        SELECT 1 FROM repayments r
        WHERE r.loan_id = v_loan_id
          AND r.payment_date <= v_first_due_date
          AND r.is_reversed = FALSE
    ) THEN
        v_fimr_tagged := FALSE;
    ELSIF v_first_payment_date IS NULL AND v_first_due_date >= CURRENT_DATE THEN
        v_fimr_tagged := FALSE;
    ELSE
        v_fimr_tagged := TRUE;
    END IF;
    
    -- Calculate repayment delay rate
    IF v_loan_age > 0 AND v_last_payment_date IS NOT NULL THEN
        v_repayment_delay_rate := (1.0 - ((v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) / 0.25)) * 100;
    ELSIF v_loan_age = 0 THEN
        v_repayment_delay_rate := 0;
    ELSE
        v_repayment_delay_rate := NULL;
    END IF;
    
    -- Update the loan record with all computed fields
    UPDATE loans
    SET
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        total_repayments = v_total_repayments,
        principal_outstanding = v_principal_outstanding,
        interest_outstanding = v_interest_outstanding,
        fees_outstanding = v_fees_outstanding,
        total_outstanding = v_total_outstanding,
        first_payment_received_date = v_first_payment_date,
        first_payment_missed = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),
        days_since_last_repayment = v_days_since_last_repayment,
        days_since_due = v_days_since_due,
        loan_age = v_loan_age,
        current_dpd = v_current_dpd,
        max_dpd_ever = GREATEST(v_max_dpd_ever, v_current_dpd),
        fimr_tagged = v_fimr_tagged,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),
        repayment_delay_rate = v_repayment_delay_rate,
        -- New DPD methodology fields
        daily_repayment_amount = v_daily_repayment_amount,
        real_loan_tenure_days = v_real_loan_tenure_days,
        repayment_days_paid = v_repayment_days_paid,
        repayment_days_due_today = v_repayment_days_due_today,
        business_days_since_disbursement = v_business_days_since_disbursement,
        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION update_loan_computed_fields() IS 
'Trigger function that recalculates all computed fields for a loan when a repayment is inserted or updated. Uses new DPD methodology based on missed repayment days.';

-- ============================================================================
-- STEP 3: Backfill business_days_since_disbursement for all existing loans
-- ============================================================================

UPDATE loans
SET business_days_since_disbursement = count_business_days(disbursement_date, CURRENT_DATE)
WHERE disbursement_date IS NOT NULL;

COMMIT;

