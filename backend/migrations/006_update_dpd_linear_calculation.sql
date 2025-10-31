-- Migration: Update DPD calculation to use linear repayment progress formula
-- This migration updates the current_dpd calculation to be based on:
-- - Expected linear repayment progress vs actual repayment progress
-- - Formula: DPD = expected_days - paid_days
-- - Where paid_days = total_repayment / (total_expected_amount / duration_in_days)

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
    v_total_expected_amount DECIMAL(15, 2);
    v_expected_days INTEGER;
    v_paid_days DECIMAL(15, 2);
    v_daily_payment_rate DECIMAL(15, 4);
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

    -- Calculate total expected amount (principal + interest + fees)
    v_total_expected_amount := v_loan_amount + 
                               (v_loan_amount * v_interest_rate * v_loan_term_days / 365) + 
                               v_fee_amount;

    -- Calculate total outstanding
    v_total_outstanding := v_total_expected_amount - v_total_repayments;

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

    -- Check if loan_schedule has any records
    SELECT COUNT(*) INTO v_schedule_count
    FROM loan_schedule
    WHERE loan_id = v_loan_id;

    -- Calculate current DPD using LINEAR REPAYMENT PROGRESS formula
    IF v_schedule_count > 0 THEN
        -- If loan_schedule exists, use it to calculate DPD (existing logic for scheduled loans)
        SELECT
            COALESCE(MAX(CURRENT_DATE - due_date), 0)
        INTO v_current_dpd
        FROM loan_schedule
        WHERE loan_id = v_loan_id
          AND payment_status IN ('Pending', 'Partial')
          AND due_date < CURRENT_DATE;
    ELSE
        -- NEW LINEAR REPAYMENT PROGRESS FORMULA
        -- Calculate expected_days (days since disbursement)
        v_expected_days := CURRENT_DATE - v_disbursement_date;

        -- Calculate daily payment rate (how much should be paid per day)
        IF v_loan_term_days > 0 AND v_total_expected_amount > 0 THEN
            v_daily_payment_rate := v_total_expected_amount / v_loan_term_days;
        ELSE
            v_daily_payment_rate := 0;
        END IF;

        -- Calculate paid_days (how many days' worth of payments have been made)
        IF v_daily_payment_rate > 0 THEN
            v_paid_days := v_total_repayments / v_daily_payment_rate;
        ELSE
            v_paid_days := 0;
        END IF;

        -- Calculate DPD = expected_days - paid_days
        -- Only count as overdue if:
        -- 1. There's outstanding balance
        -- 2. Expected days > paid days
        -- 3. Loan hasn't matured yet (or if matured, still has outstanding)
        IF v_total_outstanding > 0 AND v_expected_days > v_paid_days THEN
            v_current_dpd := GREATEST(0, FLOOR(v_expected_days - v_paid_days));
        ELSE
            -- No outstanding balance or payments are ahead of schedule
            v_current_dpd := 0;
        END IF;
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
        -- FIMR (First Installment Missed or Rescheduled): TRUE if first payment was late or never made
        fimr_tagged = CASE
            WHEN v_repayment_count < 10 THEN
                -- For loans with less than 10 repayments, check if first payment was late
                CASE
                    WHEN v_first_payment_date IS NULL THEN TRUE  -- No payment received yet
                    WHEN v_first_due_date IS NULL THEN FALSE     -- No due date available
                    WHEN v_first_payment_date > v_first_due_date THEN TRUE  -- Payment was late
                    ELSE FALSE  -- Payment was on time
                END
            ELSE
                -- For loans with 10+ repayments, keep existing value or set to FALSE
                COALESCE((SELECT fimr_tagged FROM loans WHERE loan_id = v_loan_id), FALSE)
        END,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),

        -- Days since last repayment
        days_since_last_repayment = v_days_since_last_repayment,

        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Force recalculation for all existing loans
UPDATE repayments 
SET updated_at = CURRENT_TIMESTAMP 
WHERE repayment_id IN (
    SELECT DISTINCT r.repayment_id
    FROM repayments r
    WHERE r.is_reversed = FALSE
);

-- For loans with no repayments, manually trigger calculation
UPDATE loans
SET updated_at = CURRENT_TIMESTAMP
WHERE loan_id NOT IN (
    SELECT DISTINCT loan_id FROM repayments WHERE is_reversed = FALSE
);

