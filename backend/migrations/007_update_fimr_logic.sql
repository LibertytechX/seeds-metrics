-- ============================================================================
-- Migration: 007_update_fimr_logic.sql
-- Description: Update FIMR (First Installment Missed and Recovered) tagging logic
--              to check for repayments on the exact first_payment_due_date
-- 
-- OLD Logic: FIMR = TRUE if NO repayments within first 7 days after disbursement
-- NEW Logic: FIMR = TRUE if NO repayment exists on the exact first_payment_due_date
-- 
-- Date: 2025-11-04
-- ============================================================================

-- Step 1: Show current FIMR statistics BEFORE the change
DO $$
DECLARE
    v_total_loans INTEGER;
    v_fimr_true_count INTEGER;
    v_fimr_false_count INTEGER;
    v_fimr_null_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_total_loans FROM loans;
    SELECT COUNT(*) INTO v_fimr_true_count FROM loans WHERE fimr_tagged = TRUE;
    SELECT COUNT(*) INTO v_fimr_false_count FROM loans WHERE fimr_tagged = FALSE;
    SELECT COUNT(*) INTO v_fimr_null_count FROM loans WHERE fimr_tagged IS NULL;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'FIMR STATISTICS BEFORE MIGRATION';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Total loans: %', v_total_loans;
    RAISE NOTICE 'FIMR = TRUE: % (%.2f%%)', v_fimr_true_count, (v_fimr_true_count::DECIMAL / v_total_loans * 100);
    RAISE NOTICE 'FIMR = FALSE: % (%.2f%%)', v_fimr_false_count, (v_fimr_false_count::DECIMAL / v_total_loans * 100);
    RAISE NOTICE 'FIMR = NULL: %', v_fimr_null_count;
    RAISE NOTICE '========================================';
END $$;

-- Step 2: Drop existing trigger and function
DROP TRIGGER IF EXISTS trg_update_loan_after_repayment ON repayments;
DROP TRIGGER IF EXISTS trg_update_loan_after_schedule_change ON loan_schedule;
DROP FUNCTION IF EXISTS update_loan_computed_fields();

-- Step 3: Recreate the trigger function with NEW FIMR logic
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
    v_current_dpd INTEGER;
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_disbursement_date DATE;
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_max_dpd INTEGER;
    v_repayment_count INTEGER;
    v_days_since_last_repayment INTEGER;
    v_total_outstanding DECIMAL(15, 2);
    v_schedule_count INTEGER;
    v_payment_on_due_date_exists BOOLEAN;
BEGIN
    v_loan_id := NEW.loan_id;

    -- Get loan details including disbursement date and first_payment_due_date
    SELECT loan_amount, interest_rate, loan_term_days, fee_amount, max_dpd_ever, disbursement_date, first_payment_due_date
    INTO v_loan_amount, v_interest_rate, v_loan_term_days, v_fee_amount, v_max_dpd, v_disbursement_date, v_first_due_date
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
                          (v_fee_amount - v_total_fees_paid);

    -- If first_due_date is NULL, try to get it from loan_schedule
    IF v_first_due_date IS NULL THEN
        SELECT MIN(due_date) INTO v_first_due_date
        FROM loan_schedule
        WHERE loan_id = v_loan_id;
    END IF;

    -- If still NULL, calculate first due date as 30 days after disbursement
    IF v_first_due_date IS NULL AND v_disbursement_date IS NOT NULL THEN
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

    -- NEW FIMR LOGIC: Check if there exists a repayment on the exact first_payment_due_date
    IF v_first_due_date IS NOT NULL THEN
        SELECT EXISTS (
            SELECT 1
            FROM repayments
            WHERE loan_id = v_loan_id
              AND payment_date = v_first_due_date
              AND is_reversed = FALSE
        ) INTO v_payment_on_due_date_exists;
    ELSE
        -- If no first_payment_due_date, cannot determine FIMR status
        v_payment_on_due_date_exists := FALSE;
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
        -- NEW FIMR (First Installment Missed and Recovered): TRUE if NO repayment on exact first_payment_due_date
        fimr_tagged = CASE
            WHEN v_first_due_date IS NULL THEN TRUE  -- No first payment due date available
            WHEN v_payment_on_due_date_exists THEN FALSE  -- Payment exists on first_payment_due_date
            ELSE TRUE  -- No payment on first_payment_due_date
        END,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),

        -- Days since last repayment
        days_since_last_repayment = v_days_since_last_repayment,

        -- Loan age: Days since disbursement
        loan_age = CASE
            WHEN v_disbursement_date IS NULL THEN 0
            ELSE (CURRENT_DATE - v_disbursement_date)::INTEGER
        END,

        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Step 4: Recreate triggers
CREATE TRIGGER trg_update_loan_after_repayment
AFTER INSERT OR UPDATE ON repayments
FOR EACH ROW
EXECUTE FUNCTION update_loan_computed_fields();

CREATE TRIGGER trg_update_loan_after_schedule_change
AFTER INSERT OR UPDATE ON loan_schedule
FOR EACH ROW
EXECUTE FUNCTION update_loan_computed_fields();

-- Step 5: Recalculate FIMR tags for ALL existing loans using the new logic
UPDATE loans
SET fimr_tagged = CASE
    WHEN first_payment_due_date IS NULL THEN TRUE  -- No first payment due date available
    WHEN EXISTS (
        SELECT 1
        FROM repayments
        WHERE repayments.loan_id = loans.loan_id
          AND repayments.payment_date = loans.first_payment_due_date
          AND repayments.is_reversed = FALSE
    ) THEN FALSE  -- Payment exists on first_payment_due_date
    ELSE TRUE  -- No payment on first_payment_due_date
END,
updated_at = CURRENT_TIMESTAMP;

-- Step 6: Show FIMR statistics AFTER the change
DO $$
DECLARE
    v_total_loans INTEGER;
    v_fimr_true_count INTEGER;
    v_fimr_false_count INTEGER;
    v_fimr_null_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO v_total_loans FROM loans;
    SELECT COUNT(*) INTO v_fimr_true_count FROM loans WHERE fimr_tagged = TRUE;
    SELECT COUNT(*) INTO v_fimr_false_count FROM loans WHERE fimr_tagged = FALSE;
    SELECT COUNT(*) INTO v_fimr_null_count FROM loans WHERE fimr_tagged IS NULL;
    
    RAISE NOTICE '========================================';
    RAISE NOTICE 'FIMR STATISTICS AFTER MIGRATION';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Total loans: %', v_total_loans;
    RAISE NOTICE 'FIMR = TRUE: % (%.2f%%)', v_fimr_true_count, (v_fimr_true_count::DECIMAL / v_total_loans * 100);
    RAISE NOTICE 'FIMR = FALSE: % (%.2f%%)', v_fimr_false_count, (v_fimr_false_count::DECIMAL / v_total_loans * 100);
    RAISE NOTICE 'FIMR = NULL: %', v_fimr_null_count;
    RAISE NOTICE '========================================';
END $$;

-- Step 7: Show sample of loans with their FIMR status
DO $$
DECLARE
    rec RECORD;
BEGIN
    RAISE NOTICE '';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'SAMPLE LOANS WITH NEW FIMR TAGGING';
    RAISE NOTICE '========================================';
    
    FOR rec IN (
        SELECT 
            l.loan_id,
            l.disbursement_date,
            l.first_payment_due_date,
            l.first_payment_received_date,
            l.fimr_tagged,
            (SELECT COUNT(*) FROM repayments r 
             WHERE r.loan_id = l.loan_id 
               AND r.payment_date = l.first_payment_due_date 
               AND r.is_reversed = FALSE) as payments_on_due_date
        FROM loans l
        WHERE l.first_payment_due_date IS NOT NULL
        ORDER BY l.loan_id DESC
        LIMIT 10
    )
    LOOP
        RAISE NOTICE 'Loan %: Disbursed=%, FirstDue=%, FirstReceived=%, PaymentsOnDueDate=%, FIMR=%',
            rec.loan_id,
            rec.disbursement_date,
            rec.first_payment_due_date,
            rec.first_payment_received_date,
            rec.payments_on_due_date,
            rec.fimr_tagged;
    END LOOP;
    
    RAISE NOTICE '========================================';
END $$;

