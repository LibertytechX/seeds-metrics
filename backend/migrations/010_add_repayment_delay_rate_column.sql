-- ============================================================================
-- Migration: Add Repayment Delay Rate Column to Loans Table
-- ============================================================================
-- Purpose: Add loan-level repayment_delay_rate field calculated using the same
--          formula as the officer-level metric
-- Date: 2025-11-01
-- Formula: (1 - ((days_since_last_repayment / loan_age) / 0.25)) × 100
-- ============================================================================

-- ============================================================================
-- STEP 1: Add repayment_delay_rate Column to loans Table
-- ============================================================================

ALTER TABLE loans ADD COLUMN IF NOT EXISTS repayment_delay_rate DECIMAL(8, 2) DEFAULT NULL;

COMMENT ON COLUMN loans.repayment_delay_rate IS 'Loan-level repayment delay rate: (1 - ((days_since_last_repayment / loan_age) / 0.25)) × 100. Negative values indicate poor repayment frequency.';

-- ============================================================================
-- STEP 2: Create Index for Performance
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_loans_repayment_delay_rate ON loans(repayment_delay_rate) WHERE repayment_delay_rate IS NOT NULL;

-- ============================================================================
-- STEP 3: Update Trigger Function to Calculate repayment_delay_rate
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

    -- Calculate total outstanding
    v_total_outstanding := (v_loan_amount - v_total_principal_paid) +
                          ((v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid) +
                          (COALESCE(v_fee_amount, 0) - v_total_fees_paid);

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
        v_days_since_last_repayment := NULL;
    END IF;

    -- Calculate current DPD (days past due for oldest unpaid installment)
    SELECT COUNT(*) INTO v_schedule_count
    FROM loan_schedule
    WHERE loan_id = v_loan_id;

    IF v_schedule_count > 0 THEN
        -- Use schedule-based DPD calculation
        SELECT
            COALESCE(MAX(CURRENT_DATE - due_date), 0)
        INTO v_current_dpd
        FROM loan_schedule
        WHERE loan_id = v_loan_id
          AND payment_status IN ('Pending', 'Partial')
          AND due_date < CURRENT_DATE;
    ELSE
        -- Fallback: Linear DPD calculation
        v_current_dpd := GREATEST(0, CURRENT_DATE - v_disbursement_date - 30);
    END IF;

    -- Calculate actual_outstanding (repayment_amount - total_repayments)
    SELECT repayment_amount INTO v_actual_outstanding
    FROM loans
    WHERE loan_id = v_loan_id;

    IF v_actual_outstanding IS NOT NULL THEN
        v_actual_outstanding := v_actual_outstanding - v_total_repayments;
    ELSE
        v_actual_outstanding := v_total_outstanding;
    END IF;

    -- NEW: Calculate loan age in days
    v_loan_age := CURRENT_DATE - v_disbursement_date;

    -- NEW: Calculate repayment_delay_rate
    -- Formula: (1 - ((days_since_last_repayment / loan_age) / 0.25)) × 100
    -- Edge cases:
    -- - If loan_age = 0, return 0
    -- - If days_since_last_repayment is NULL, return NULL
    -- - Allow negative values (as per officer-level logic)
    IF v_days_since_last_repayment IS NOT NULL AND v_loan_age > 0 THEN
        v_repayment_delay_rate := (1.0 - ((v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) / 0.25)) * 100;
    ELSIF v_loan_age = 0 THEN
        v_repayment_delay_rate := 0;
    ELSE
        v_repayment_delay_rate := NULL;
    END IF;

    -- Update loans table with computed values
    UPDATE loans
    SET
        -- Collections totals
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        total_repayments = v_total_repayments,

        -- Outstanding balances
        principal_outstanding = v_loan_amount - v_total_principal_paid,
        interest_outstanding = (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid,
        fees_outstanding = COALESCE(v_fee_amount, 0) - v_total_fees_paid,
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
        fimr_tagged = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),

        -- Days since last repayment
        days_since_last_repayment = v_days_since_last_repayment,

        -- NEW: Repayment delay rate
        repayment_delay_rate = v_repayment_delay_rate,

        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STEP 4: Backfill repayment_delay_rate for Existing Loans
-- ============================================================================

-- Calculate repayment_delay_rate for all existing loans
UPDATE loans
SET repayment_delay_rate = CASE
    WHEN days_since_last_repayment IS NOT NULL AND (CURRENT_DATE - disbursement_date) > 0 THEN
        (1.0 - ((days_since_last_repayment::DECIMAL / (CURRENT_DATE - disbursement_date)::DECIMAL) / 0.25)) * 100
    WHEN (CURRENT_DATE - disbursement_date) = 0 THEN
        0
    ELSE
        NULL
END
WHERE loan_id IS NOT NULL;

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

