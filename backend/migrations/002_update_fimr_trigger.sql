-- Migration: Update FIMR tagging logic in loan metrics trigger
-- This migration updates the update_loan_computed_fields() function to:
-- 1. Calculate first_payment_due_date if not in loan_schedule (30 days after disbursement)
-- 2. Properly set fimr_tagged based on whether first payment was late
-- 3. Only apply FIMR tagging for loans with less than 10 repayments

CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_total_principal_paid DECIMAL(15, 2);
    v_total_interest_paid DECIMAL(15, 2);
    v_total_fees_paid DECIMAL(15, 2);
    v_first_payment_date DATE;
    v_first_due_date DATE;
    v_disbursement_date DATE;
    v_current_dpd INTEGER;
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_max_dpd INTEGER;
    v_repayment_count INTEGER;
BEGIN
    v_loan_id := NEW.loan_id;

    -- Get loan details including disbursement date
    SELECT loan_amount, interest_rate, loan_term_days, fee_amount, max_dpd_ever, disbursement_date
    INTO v_loan_amount, v_interest_rate, v_loan_term_days, v_fee_amount, v_max_dpd, v_disbursement_date
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Calculate total payments (excluding reversed payments)
    SELECT
        COALESCE(SUM(principal_paid), 0),
        COALESCE(SUM(interest_paid), 0),
        COALESCE(SUM(fees_paid), 0),
        MIN(payment_date),
        COUNT(*)
    INTO
        v_total_principal_paid,
        v_total_interest_paid,
        v_total_fees_paid,
        v_first_payment_date,
        v_repayment_count
    FROM repayments
    WHERE loan_id = v_loan_id
      AND is_reversed = FALSE;

    -- Get first due date from loan_schedule
    SELECT MIN(due_date) INTO v_first_due_date
    FROM loan_schedule
    WHERE loan_id = v_loan_id;
    
    -- If no schedule exists, calculate first due date as 30 days after disbursement
    IF v_first_due_date IS NULL THEN
        v_first_due_date := v_disbursement_date + INTERVAL '30 days';
    END IF;

    -- Calculate current DPD (days past due for oldest unpaid installment)
    SELECT
        COALESCE(MAX(CURRENT_DATE - due_date), 0)
    INTO v_current_dpd
    FROM loan_schedule
    WHERE loan_id = v_loan_id
      AND payment_status IN ('Pending', 'Partial')
      AND due_date < CURRENT_DATE;

    -- Update loans table with computed values
    UPDATE loans
    SET
        -- Collections totals
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,

        -- Outstanding balances
        principal_outstanding = v_loan_amount - v_total_principal_paid,
        interest_outstanding = (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid,
        fees_outstanding = v_fee_amount - v_total_fees_paid,
        total_outstanding = (v_loan_amount - v_total_principal_paid) +
                           ((v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid) +
                           (v_fee_amount - v_total_fees_paid),

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

        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

