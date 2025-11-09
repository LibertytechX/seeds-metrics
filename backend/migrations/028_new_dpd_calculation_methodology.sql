-- ============================================================================
-- Migration 028: New DPD Calculation Methodology
-- ============================================================================
-- Description: Implements a new DPD calculation based on missed repayment days
--              instead of "days since last payment". The new methodology:
--              1. Calculates daily repayment amount based on loan term
--              2. Counts business days (Mon-Fri) from first payment due date
--              3. Calculates how many repayment days have been paid
--              4. DPD = repayment days due - repayment days paid
--
-- Author: Seeds Metrics Team
-- Date: 2025-11-09
-- ============================================================================

BEGIN;

-- ============================================================================
-- STEP 1: Create function to count business days (excluding weekends)
-- ============================================================================

CREATE OR REPLACE FUNCTION count_business_days(start_date DATE, end_date DATE)
RETURNS INTEGER
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    v_business_days INTEGER := 0;
    v_current_date DATE;
    v_day_of_week INTEGER;
BEGIN
    -- Handle NULL inputs
    IF start_date IS NULL OR end_date IS NULL THEN
        RETURN 0;
    END IF;
    
    -- Handle case where start_date is after end_date
    IF start_date > end_date THEN
        RETURN 0;
    END IF;
    
    -- Loop through each day and count business days (Mon-Fri)
    v_current_date := start_date;
    
    WHILE v_current_date <= end_date LOOP
        -- Get day of week (0=Sunday, 1=Monday, ..., 6=Saturday)
        v_day_of_week := EXTRACT(DOW FROM v_current_date);
        
        -- Count only Monday (1) through Friday (5)
        IF v_day_of_week BETWEEN 1 AND 5 THEN
            v_business_days := v_business_days + 1;
        END IF;
        
        -- Move to next day
        v_current_date := v_current_date + 1;
    END LOOP;
    
    RETURN v_business_days;
END;
$$;

COMMENT ON FUNCTION count_business_days(DATE, DATE) IS 
'Counts the number of business days (Monday-Friday) between two dates, inclusive. Excludes weekends (Saturday and Sunday).';

-- ============================================================================
-- STEP 2: Create function to count weekend days within a period
-- ============================================================================

CREATE OR REPLACE FUNCTION count_weekend_days(start_date DATE, end_date DATE)
RETURNS INTEGER
LANGUAGE plpgsql
IMMUTABLE
AS $$
DECLARE
    v_weekend_days INTEGER := 0;
    v_current_date DATE;
    v_day_of_week INTEGER;
BEGIN
    -- Handle NULL inputs
    IF start_date IS NULL OR end_date IS NULL THEN
        RETURN 0;
    END IF;
    
    -- Handle case where start_date is after end_date
    IF start_date > end_date THEN
        RETURN 0;
    END IF;
    
    -- Loop through each day and count weekend days
    v_current_date := start_date;
    
    WHILE v_current_date <= end_date LOOP
        -- Get day of week (0=Sunday, 6=Saturday)
        v_day_of_week := EXTRACT(DOW FROM v_current_date);
        
        -- Count Saturday (6) and Sunday (0)
        IF v_day_of_week = 0 OR v_day_of_week = 6 THEN
            v_weekend_days := v_weekend_days + 1;
        END IF;
        
        -- Move to next day
        v_current_date := v_current_date + 1;
    END LOOP;
    
    RETURN v_weekend_days;
END;
$$;

COMMENT ON FUNCTION count_weekend_days(DATE, DATE) IS 
'Counts the number of weekend days (Saturday and Sunday) between two dates, inclusive.';

-- ============================================================================
-- STEP 3: Add new computed fields to loans table
-- ============================================================================

-- Add daily_repayment_amount column
ALTER TABLE loans 
ADD COLUMN IF NOT EXISTS daily_repayment_amount DECIMAL(15, 2) DEFAULT 0;

COMMENT ON COLUMN loans.daily_repayment_amount IS 
'Expected daily repayment amount = repayment_amount / loan_term_days (business days only)';

-- Add real_loan_tenure_days column
ALTER TABLE loans 
ADD COLUMN IF NOT EXISTS real_loan_tenure_days INTEGER DEFAULT 0;

COMMENT ON COLUMN loans.real_loan_tenure_days IS 
'Actual calendar days from first_payment_due_date to maturity_date, including weekends';

-- Add repayment_days_paid column
ALTER TABLE loans 
ADD COLUMN IF NOT EXISTS repayment_days_paid DECIMAL(10, 2) DEFAULT 0;

COMMENT ON COLUMN loans.repayment_days_paid IS 
'Number of business days worth of repayments made = total_repayments / daily_repayment_amount';

-- Add repayment_days_due_today column
ALTER TABLE loans 
ADD COLUMN IF NOT EXISTS repayment_days_due_today INTEGER DEFAULT 0;

COMMENT ON COLUMN loans.repayment_days_due_today IS 
'Number of business days of repayments due as of today, counted from first_payment_due_date';

-- ============================================================================
-- STEP 4: Update the update_loan_computed_fields() trigger function
-- ============================================================================

CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
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
    v_repayment_amount DECIMAL(15, 2);
    v_maturity_date DATE;
    v_max_dpd INTEGER;
    v_repayment_count INTEGER;
    v_days_since_last_repayment INTEGER;
    v_total_outstanding DECIMAL(15, 2);
    v_payment_on_due_date_exists BOOLEAN;
    v_days_since_due INTEGER;
    
    -- New variables for new DPD calculation
    v_daily_repayment_amount DECIMAL(15, 2);
    v_real_loan_tenure_days INTEGER;
    v_repayment_days_paid DECIMAL(10, 2);
    v_repayment_days_due_today INTEGER;
    v_weekend_days_in_tenure INTEGER;
    v_calculation_end_date DATE;
BEGIN
    v_loan_id := NEW.loan_id;
    
    -- Get loan details
    SELECT
        loan_amount, interest_rate, disbursement_date, loan_term_days, fee_amount,
        repayment_amount, maturity_date, first_payment_due_date
    INTO
        v_loan_amount, v_interest_rate, v_disbursement_date, v_loan_term_days, v_fee_amount,
        v_repayment_amount, v_maturity_date, v_first_due_date
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
    
    -- Calculate days_since_due
    IF v_first_payment_date IS NOT NULL AND v_first_due_date IS NOT NULL THEN
        v_days_since_due := (v_first_payment_date - v_first_due_date)::INTEGER;
    ELSIF v_first_payment_date IS NULL AND v_first_due_date IS NOT NULL THEN
        v_days_since_due := (CURRENT_DATE - v_first_due_date)::INTEGER;
    ELSE
        v_days_since_due := 0;
    END IF;
    
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
    
    -- Step 4: Calculate repayment days due today
    IF v_first_due_date IS NOT NULL THEN
        -- Use the earlier of CURRENT_DATE or maturity_date
        v_calculation_end_date := LEAST(CURRENT_DATE, COALESCE(v_maturity_date, CURRENT_DATE));
        
        -- Count business days from first_payment_due_date to calculation_end_date
        IF v_calculation_end_date >= v_first_due_date THEN
            v_repayment_days_due_today := count_business_days(v_first_due_date, v_calculation_end_date);
        ELSE
            v_repayment_days_due_today := 0;
        END IF;
    ELSE
        v_repayment_days_due_today := 0;
    END IF;
    
    -- Step 5: Calculate DPD (missed repayment days)
    v_current_dpd := GREATEST(0, v_repayment_days_due_today - v_repayment_days_paid::INTEGER);
    
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
        -- New fields
        daily_repayment_amount = v_daily_repayment_amount,
        real_loan_tenure_days = v_real_loan_tenure_days,
        repayment_days_paid = v_repayment_days_paid,
        repayment_days_due_today = v_repayment_days_due_today,
        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;
    
    RETURN NEW;
END;
$$;

COMMENT ON FUNCTION update_loan_computed_fields() IS 
'Trigger function that recalculates all computed fields for a loan after repayment insert/update. Uses new DPD methodology based on missed repayment days.';

COMMIT;

