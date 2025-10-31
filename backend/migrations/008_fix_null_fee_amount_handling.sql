-- Migration: Fix NULL fee_amount handling in update_loan_computed_fields trigger
-- This migration updates the trigger function to properly handle NULL fee_amount values
-- using COALESCE to treat NULL as 0 in calculations

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
    v_total_expected_repayment DECIMAL(15, 2);
    v_days_elapsed INTEGER;
    v_projected_due_amount DECIMAL(15, 2);
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

    -- Calculate total outstanding (FIX: Use COALESCE for NULL fee_amount)
    v_total_outstanding := (v_loan_amount - v_total_principal_paid) +
                          ((v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid) +
                          (COALESCE(v_fee_amount, 0) - v_total_fees_paid);

    -- Calculate actual_outstanding
    -- Step 1: Calculate total expected repayment amount (principal + interest + fees)
    -- FIX: Use COALESCE for NULL fee_amount
    v_total_expected_repayment := v_loan_amount +
                                  (v_loan_amount * v_interest_rate * v_loan_term_days / 365) +
                                  COALESCE(v_fee_amount, 0);

    -- Step 2: Calculate days elapsed (capped at loan tenure)
    v_days_elapsed := LEAST(CURRENT_DATE - v_disbursement_date, v_loan_term_days);

    -- Ensure days_elapsed is not negative (for future-dated loans)
    v_days_elapsed := GREATEST(v_days_elapsed, 0);

    -- Step 3: Calculate projected due amount (what should have been paid by now)
    IF v_loan_term_days > 0 THEN
        v_projected_due_amount := (v_total_expected_repayment / v_loan_term_days) * v_days_elapsed;
    ELSE
        v_projected_due_amount := 0;
    END IF;

    -- Step 4: Calculate actual outstanding (projected due - total repayments)
    v_actual_outstanding := v_projected_due_amount - v_total_repayments;

    -- If negative (borrower is ahead), set to 0
    v_actual_outstanding := GREATEST(v_actual_outstanding, 0);

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

    -- Calculate current DPD
    IF v_schedule_count > 0 THEN
        -- If loan_schedule exists, use it to calculate DPD (existing logic)
        SELECT
            COALESCE(MAX(CURRENT_DATE - due_date), 0)
        INTO v_current_dpd
        FROM loan_schedule
        WHERE loan_id = v_loan_id
          AND payment_status IN ('Pending', 'Partial')
          AND due_date < CURRENT_DATE;
    ELSE
        -- If no loan_schedule, calculate DPD based on payment history and outstanding balance
        -- Logic: If there's outstanding balance and it's been > 30 days since last payment (or disbursement), loan is overdue
        IF v_total_outstanding > 0 THEN
            IF v_last_payment_date IS NOT NULL THEN
                -- Calculate DPD based on days since last payment
                -- Assume monthly payments (30 days), so if > 30 days since last payment, it's overdue
                v_current_dpd := GREATEST(0, CURRENT_DATE - v_last_payment_date - 30);
            ELSE
                -- No payments yet - calculate DPD based on days since disbursement
                -- Assume first payment due 30 days after disbursement
                v_current_dpd := GREATEST(0, CURRENT_DATE - v_disbursement_date - 30);
            END IF;
        ELSE
            -- No outstanding balance - loan is fully paid, DPD = 0
            v_current_dpd := 0;
        END IF;
    END IF;

    -- Update loans table with computed values (FIX: Use COALESCE for NULL fee_amount)
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

        -- Update timestamp
        updated_at = CURRENT_TIMESTAMP

    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 2: Don't add a trigger on loans table (would cause infinite recursion)
-- Instead, we'll rely on:
-- 1. The repayments trigger to update fields when payments are made
-- 2. The manual recalculation endpoint for time-based updates
-- 3. The direct UPDATE below to backfill existing loans

-- Step 3: Manually recalculate all existing loans using direct UPDATE
-- This is more reliable than relying on the trigger for backfilling
UPDATE loans l
SET
    principal_outstanding = l.loan_amount - COALESCE(l.total_principal_paid, 0),
    interest_outstanding = (l.loan_amount * l.interest_rate * l.loan_term_days / 365) - COALESCE(l.total_interest_paid, 0),
    fees_outstanding = COALESCE(l.fee_amount, 0) - COALESCE(l.total_fees_paid, 0),
    total_outstanding = (l.loan_amount - COALESCE(l.total_principal_paid, 0)) +
                       ((l.loan_amount * l.interest_rate * l.loan_term_days / 365) - COALESCE(l.total_interest_paid, 0)) +
                       (COALESCE(l.fee_amount, 0) - COALESCE(l.total_fees_paid, 0)),
    actual_outstanding = GREATEST(
        (
            -- Calculate projected due amount
            CASE
                WHEN l.loan_term_days > 0 THEN
                    (
                        -- Total expected repayment (FIX: Use COALESCE for NULL fee_amount)
                        (l.loan_amount +
                         (l.loan_amount * l.interest_rate * l.loan_term_days / 365) +
                         COALESCE(l.fee_amount, 0))
                        / l.loan_term_days
                    ) *
                    -- Days elapsed (capped at loan tenure)
                    LEAST(GREATEST(CURRENT_DATE - l.disbursement_date, 0), l.loan_term_days)
                ELSE 0
            END
        ) - COALESCE(l.total_repayments, 0),
        0  -- If negative (ahead of schedule), set to 0
    )
WHERE l.loan_id IS NOT NULL;

