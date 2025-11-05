-- Migration 016: Update FIMR logic to recognize early payments as on-time
-- FIMR = FALSE if repayment exists on or before first_payment_due_date
-- FIMR = TRUE if no repayment exists on or before first_payment_due_date

-- Step 1: Show current FIMR statistics BEFORE the change
DO $$
BEGIN
    RAISE NOTICE '=== FIMR STATISTICS BEFORE MIGRATION ===';
END $$;

SELECT
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as fimr_true_count,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as fimr_false_count,
    ROUND(100.0 * SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) / COUNT(*), 2) as fimr_true_percentage
FROM loans;

-- Step 2: Drop existing trigger
DROP TRIGGER IF EXISTS trg_update_loan_computed_fields ON repayments;

-- Step 3: Drop existing function
DROP FUNCTION IF EXISTS update_loan_computed_fields();

-- Step 4: Recreate the trigger function with NEW FIMR logic (early payments count as on-time)
CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id TEXT;
    v_loan_amount NUMERIC;
    v_interest_rate NUMERIC;
    v_loan_term_days INTEGER;
    v_fee_amount NUMERIC;
    v_disbursement_date DATE;
    v_first_due_date DATE;
    v_total_principal_paid NUMERIC;
    v_total_interest_paid NUMERIC;
    v_total_fees_paid NUMERIC;
    v_principal_outstanding NUMERIC;
    v_interest_outstanding NUMERIC;
    v_fees_outstanding NUMERIC;
    v_total_outstanding NUMERIC;
    v_current_dpd INTEGER;
    v_max_dpd_ever INTEGER;
    v_first_payment_date DATE;
    v_last_payment_date DATE;
    v_days_since_last_repayment INTEGER;
    v_payment_on_or_before_due_date_exists BOOLEAN;
BEGIN
    IF TG_OP = 'INSERT' OR TG_OP = 'UPDATE' THEN
        v_loan_id := NEW.loan_id;
    ELSIF TG_OP = 'DELETE' THEN
        v_loan_id := OLD.loan_id;
    END IF;

    SELECT
        loan_amount,
        interest_rate,
        loan_term_days,
        fee_amount,
        disbursement_date,
        first_payment_due_date
    INTO
        v_loan_amount,
        v_interest_rate,
        v_loan_term_days,
        v_fee_amount,
        v_disbursement_date,
        v_first_due_date
    FROM loans
    WHERE loan_id = v_loan_id;

    SELECT
        COALESCE(SUM(principal_amount), 0),
        COALESCE(SUM(interest_amount), 0),
        COALESCE(SUM(fees_amount), 0),
        MIN(payment_date),
        MAX(payment_date)
    INTO
        v_total_principal_paid,
        v_total_interest_paid,
        v_total_fees_paid,
        v_first_payment_date,
        v_last_payment_date
    FROM repayments
    WHERE loan_id = v_loan_id
      AND is_reversed = FALSE;

    v_principal_outstanding := GREATEST(0, v_loan_amount - v_total_principal_paid);
    v_interest_outstanding := GREATEST(0,
        (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid
    );
    v_fees_outstanding := GREATEST(0,
        COALESCE(v_fee_amount, 0) - v_total_fees_paid
    );
    v_total_outstanding := GREATEST(0,
        v_principal_outstanding + v_interest_outstanding + v_fees_outstanding
    );

    IF v_total_outstanding > 0 AND v_first_due_date IS NOT NULL THEN
        v_current_dpd := GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER);
    ELSE
        v_current_dpd := 0;
    END IF;

    SELECT COALESCE(MAX(current_dpd), 0)
    INTO v_max_dpd_ever
    FROM loans
    WHERE loan_id = v_loan_id;

    v_max_dpd_ever := GREATEST(v_max_dpd_ever, v_current_dpd);

    IF v_last_payment_date IS NOT NULL THEN
        v_days_since_last_repayment := (CURRENT_DATE - v_last_payment_date)::INTEGER;
    ELSE
        v_days_since_last_repayment := NULL;
    END IF;

    IF v_first_due_date IS NOT NULL THEN
        SELECT EXISTS (
            SELECT 1
            FROM repayments
            WHERE loan_id = v_loan_id
              AND payment_date <= v_first_due_date
              AND is_reversed = FALSE
        ) INTO v_payment_on_or_before_due_date_exists;
    ELSE
        v_payment_on_or_before_due_date_exists := FALSE;
    END IF;

    UPDATE loans
    SET
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        principal_outstanding = v_principal_outstanding,
        interest_outstanding = v_interest_outstanding,
        fees_outstanding = v_fees_outstanding,
        total_outstanding = v_total_outstanding,
        actual_outstanding = GREATEST(0, actual_outstanding),
        current_dpd = v_current_dpd,
        max_dpd_ever = v_max_dpd_ever,
        first_payment_received_date = v_first_payment_date,
        last_payment_received_date = v_last_payment_date,
        days_since_last_repayment = v_days_since_last_repayment,
        fimr_tagged = CASE
            WHEN v_first_due_date IS NULL THEN TRUE
            WHEN v_payment_on_or_before_due_date_exists THEN FALSE
            ELSE TRUE
        END,
        loan_age = (CURRENT_DATE - disbursement_date::date)::INTEGER,
        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 5: Recreate triggers
CREATE TRIGGER trg_update_loan_computed_fields
AFTER INSERT OR UPDATE OR DELETE ON repayments
FOR EACH ROW
EXECUTE FUNCTION update_loan_computed_fields();

-- Step 6: Recalculate FIMR tags for ALL existing loans using the new logic
DO $$
BEGIN
    RAISE NOTICE '=== RECALCULATING FIMR TAGS FOR ALL LOANS ===';
END $$;

UPDATE loans l
SET
    fimr_tagged = CASE
        WHEN l.first_payment_due_date IS NULL THEN TRUE
        WHEN EXISTS (
            SELECT 1
            FROM repayments r
            WHERE r.loan_id = l.loan_id
              AND r.payment_date <= l.first_payment_due_date
              AND r.is_reversed = FALSE
        ) THEN FALSE
        WHEN l.first_payment_received_date IS NULL AND l.first_payment_due_date >= CURRENT_DATE THEN FALSE  -- No payment yet but due date not passed
        ELSE TRUE
    END,
    updated_at = CURRENT_TIMESTAMP;

-- Step 7: Show FIMR statistics AFTER the change
DO $$
BEGIN
    RAISE NOTICE '=== FIMR STATISTICS AFTER MIGRATION ===';
END $$;

SELECT
    COUNT(*) as total_loans,
    SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) as fimr_true_count,
    SUM(CASE WHEN fimr_tagged = FALSE THEN 1 ELSE 0 END) as fimr_false_count,
    ROUND(100.0 * SUM(CASE WHEN fimr_tagged = TRUE THEN 1 ELSE 0 END) / COUNT(*), 2) as fimr_true_percentage
FROM loans;

-- Step 8: Show sample of loans with early payments that are now correctly tagged as FIMR = FALSE
DO $$
BEGIN
    RAISE NOTICE '=== SAMPLE: LOANS WITH EARLY PAYMENTS (NOW FIMR = FALSE) ===';
END $$;

SELECT
    l.loan_id,
    l.first_payment_due_date,
    l.first_payment_received_date,
    (l.first_payment_received_date - l.first_payment_due_date) as days_early,
    l.fimr_tagged,
    (SELECT COUNT(*) FROM repayments r WHERE r.loan_id = l.loan_id AND r.payment_date <= l.first_payment_due_date AND r.is_reversed = FALSE) as payments_on_or_before_due_date
FROM loans l
WHERE l.first_payment_due_date IS NOT NULL
  AND l.first_payment_received_date IS NOT NULL
  AND l.first_payment_received_date < l.first_payment_due_date
  AND l.fimr_tagged = FALSE
ORDER BY (l.first_payment_received_date - l.first_payment_due_date) ASC
LIMIT 10;

-- Step 9: Show sample of loans with late first payments (FIMR = TRUE)
DO $$
BEGIN
    RAISE NOTICE '=== SAMPLE: LOANS WITH LATE FIRST PAYMENTS (FIMR = TRUE) ===';
END $$;

SELECT
    l.loan_id,
    l.first_payment_due_date,
    l.first_payment_received_date,
    (l.first_payment_received_date - l.first_payment_due_date) as days_late,
    l.fimr_tagged,
    (SELECT COUNT(*) FROM repayments r WHERE r.loan_id = l.loan_id AND r.payment_date <= l.first_payment_due_date AND r.is_reversed = FALSE) as payments_on_or_before_due_date
FROM loans l
WHERE l.first_payment_due_date IS NOT NULL
  AND l.first_payment_received_date IS NOT NULL
  AND l.first_payment_received_date > l.first_payment_due_date
  AND l.fimr_tagged = TRUE
ORDER BY (l.first_payment_received_date - l.first_payment_due_date) DESC
LIMIT 10;

-- Step 10: Summary of changes
DO $$
BEGIN
    RAISE NOTICE '=== MIGRATION 016 COMPLETED ===';
    RAISE NOTICE 'FIMR logic updated to recognize early payments as on-time';
    RAISE NOTICE 'FIMR = FALSE if repayment exists on or before first_payment_due_date';
    RAISE NOTICE 'FIMR = TRUE if no repayment exists on or before first_payment_due_date';
END $$;

