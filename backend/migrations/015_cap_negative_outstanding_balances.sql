-- ============================================================================
-- Migration: Cap Negative Outstanding Balances at Zero
-- ============================================================================
-- Purpose: Update loan outstanding balance calculations to handle over-payment
--          scenarios by capping negative values at zero instead of allowing
--          negative balances.
-- Date: 2025-11-04
-- Business Rule: When total_principal_paid > loan_amount (over-payment),
--                principal_outstanding should be 0, not negative.
--                This prevents negative portfolio totals in officer metrics.
-- Issue: Officer 'adeyinka232803@gmail.com' has negative portfolio total
--        due to loans with negative principal_outstanding values.
-- ============================================================================

-- ============================================================================
-- STEP 1: Update Trigger Function to Cap Negative Values
-- ============================================================================

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
    v_disbursement_date DATE;
    v_current_dpd INTEGER;
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_max_dpd INTEGER;
    v_repayment_count INTEGER;
    v_days_since_last_repayment INTEGER;
    v_total_outstanding DECIMAL(15, 2);
    v_schedule_count INTEGER;
    v_actual_outstanding DECIMAL(15, 2);
    v_loan_age INTEGER;
    v_repayment_delay_rate DECIMAL(8, 2);
    -- NEW: Variables for capped outstanding balances
    v_principal_outstanding DECIMAL(15, 2);
    v_interest_outstanding DECIMAL(15, 2);
    v_fees_outstanding DECIMAL(15, 2);
BEGIN
    v_loan_id := NEW.loan_id;

    -- Get loan details
    SELECT loan_amount, interest_rate, loan_term_days, fee_amount, max_dpd_ever, disbursement_date
    INTO v_loan_amount, v_interest_rate, v_loan_term_days, v_fee_amount, v_max_dpd, v_disbursement_date
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Calculate total payments (excluding reversed payments)
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
    WHERE loan_id = v_loan_id
      AND is_reversed = FALSE;

    -- ========================================================================
    -- UPDATED: Calculate outstanding balances with GREATEST() to cap at 0
    -- ========================================================================
    
    -- Calculate principal outstanding (cap at 0 if over-paid)
    v_principal_outstanding := GREATEST(0, v_loan_amount - v_total_principal_paid);
    
    -- Calculate interest outstanding (cap at 0 if over-paid)
    v_interest_outstanding := GREATEST(0, 
        (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid
    );
    
    -- Calculate fees outstanding (cap at 0 if over-paid)
    v_fees_outstanding := GREATEST(0, 
        COALESCE(v_fee_amount, 0) - v_total_fees_paid
    );
    
    -- Calculate total outstanding (cap at 0)
    v_total_outstanding := GREATEST(0,
        v_principal_outstanding + v_interest_outstanding + v_fees_outstanding
    );

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
        v_days_since_last_repayment := CURRENT_DATE - v_disbursement_date;
    END IF;

    -- Calculate loan age (days since disbursement)
    v_loan_age := CURRENT_DATE - v_disbursement_date;

    -- Calculate repayment delay rate
    -- Formula: (1 - ((days_since_last_repayment / loan_age) / 0.25)) × 100
    IF v_loan_age > 0 THEN
        v_repayment_delay_rate := (1 - ((v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) / 0.25)) * 100;
    ELSE
        v_repayment_delay_rate := NULL;
    END IF;

    -- Check if loan_schedule table has data for this loan
    SELECT COUNT(*) INTO v_schedule_count
    FROM loan_schedule
    WHERE loan_id = v_loan_id;

    -- Calculate current DPD
    IF v_schedule_count > 0 THEN
        -- If loan_schedule exists, use it to calculate DPD
        SELECT
            COALESCE(MAX(CURRENT_DATE - due_date), 0)
        INTO v_current_dpd
        FROM loan_schedule
        WHERE loan_id = v_loan_id
          AND payment_status IN ('Pending', 'Partial')
          AND due_date < CURRENT_DATE;
    ELSE
        -- If no loan_schedule, calculate DPD using linear daily payment method
        DECLARE
            v_daily_payment_rate DECIMAL(15, 2);
            v_expected_days INTEGER;
            v_paid_days DECIMAL(15, 2);
        BEGIN
            -- Calculate daily payment rate (total loan amount / loan term)
            IF v_loan_term_days > 0 THEN
                v_daily_payment_rate := v_loan_amount / v_loan_term_days;
            ELSE
                v_daily_payment_rate := 0;
            END IF;

            -- Calculate expected days (how many days should have been paid by now)
            v_expected_days := LEAST(v_loan_age, v_loan_term_days);

            -- Calculate paid_days (how many days' worth of payments have been made)
            IF v_daily_payment_rate > 0 THEN
                v_paid_days := v_total_repayments / v_daily_payment_rate;
            ELSE
                v_paid_days := 0;
            END IF;

            -- Calculate DPD = expected_days - paid_days
            IF v_total_outstanding > 0 AND v_expected_days > v_paid_days THEN
                v_current_dpd := GREATEST(0, FLOOR(v_expected_days - v_paid_days));
            ELSE
                v_current_dpd := 0;
            END IF;
        END;
    END IF;

    -- Calculate actual outstanding (only installments due to date)
    IF v_schedule_count > 0 THEN
        -- Use loan_schedule if available
        SELECT COALESCE(SUM(total_due - amount_paid), 0)
        INTO v_actual_outstanding
        FROM loan_schedule
        WHERE loan_id = v_loan_id
          AND due_date <= CURRENT_DATE
          AND payment_status IN ('Pending', 'Partial', 'Overdue');
    ELSE
        -- Estimate based on loan age and total outstanding
        IF v_loan_term_days > 0 THEN
            v_actual_outstanding := v_total_outstanding *
                LEAST(1.0, GREATEST(0.0, v_loan_age::DECIMAL / v_loan_term_days::DECIMAL));
        ELSE
            v_actual_outstanding := v_total_outstanding;
        END IF;
    END IF;

    -- Cap actual_outstanding at 0 (cannot be negative)
    v_actual_outstanding := GREATEST(0, v_actual_outstanding);

    -- Update loans table with computed values
    UPDATE loans
    SET
        -- Collections totals
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        total_repayments = v_total_repayments,

        -- Outstanding balances (UPDATED: Now capped at 0)
        principal_outstanding = v_principal_outstanding,
        interest_outstanding = v_interest_outstanding,
        fees_outstanding = v_fees_outstanding,
        total_outstanding = v_total_outstanding,
        actual_outstanding = v_actual_outstanding,

        -- First payment tracking
        first_payment_received_date = v_first_payment_date,
        first_payment_due_date = v_first_due_date,
        first_payment_missed = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),

        -- DPD tracking
        current_dpd = v_current_dpd,
        max_dpd_ever = GREATEST(v_max_dpd, v_current_dpd),

        -- Risk indicators
        -- FIMR LOGIC: Only tag as FIMR if first payment is 4+ days late
        fimr_tagged = CASE
            WHEN v_first_payment_date IS NULL AND v_first_due_date IS NOT NULL THEN
                -- No payment received yet - check if 4+ days overdue
                (CURRENT_DATE - v_first_due_date) >= 4
            WHEN v_first_payment_date IS NOT NULL AND v_first_due_date IS NOT NULL THEN
                -- Payment received - check if it was 4+ days late
                (v_first_payment_date - v_first_due_date) >= 4
            ELSE
                FALSE
        END,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),

        -- Days since last repayment
        days_since_last_repayment = v_days_since_last_repayment,

        -- Repayment delay rate
        repayment_delay_rate = v_repayment_delay_rate,

        -- Repayment count
        repayment_count = v_repayment_count

    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STEP 2: Recalculate All Existing Loans to Fix Negative Balances
-- ============================================================================

-- Update all loans to cap negative outstanding balances at 0
UPDATE loans
SET
    principal_outstanding = GREATEST(0, loan_amount - total_principal_paid),
    interest_outstanding = GREATEST(0, 
        (loan_amount * interest_rate * loan_term_days / 365) - total_interest_paid
    ),
    fees_outstanding = GREATEST(0, 
        COALESCE(fee_amount, 0) - total_fees_paid
    ),
    total_outstanding = GREATEST(0,
        GREATEST(0, loan_amount - total_principal_paid) +
        GREATEST(0, (loan_amount * interest_rate * loan_term_days / 365) - total_interest_paid) +
        GREATEST(0, COALESCE(fee_amount, 0) - total_fees_paid)
    ),
    actual_outstanding = GREATEST(0, actual_outstanding),
    updated_at = CURRENT_TIMESTAMP
WHERE 
    principal_outstanding < 0 
    OR interest_outstanding < 0 
    OR fees_outstanding < 0 
    OR total_outstanding < 0
    OR actual_outstanding < 0;

-- ============================================================================
-- STEP 3: Verification Queries
-- ============================================================================

-- Check for any remaining negative balances (should return 0 rows)
SELECT 
    'Loans with negative balances (should be 0)' as check_description,
    COUNT(*) as count
FROM loans
WHERE 
    principal_outstanding < 0 
    OR interest_outstanding < 0 
    OR fees_outstanding < 0 
    OR total_outstanding < 0
    OR actual_outstanding < 0;

-- Check officer 'adeyinka232803@gmail.com' portfolio total
SELECT 
    'Officer adeyinka232803@gmail.com portfolio check' as check_description,
    COUNT(*) as total_loans,
    SUM(principal_outstanding) as portfolio_total,
    SUM(CASE WHEN principal_outstanding < 0 THEN 1 ELSE 0 END) as negative_loans
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';

-- Summary of over-paid loans (now showing 0 outstanding instead of negative)
SELECT 
    'Over-paid loans summary' as check_description,
    COUNT(*) as overpaid_loan_count,
    SUM(total_principal_paid - loan_amount) as total_overpayment_amount
FROM loans
WHERE total_principal_paid > loan_amount;

