-- ============================================================================
-- Migration 035: Update Repayment Delay Rate Formula
-- ============================================================================
-- Description: Updates the repayment_delay_rate calculation formula to
--              incorporate both days_since_last_repayment AND current_dpd
--              for a more balanced metric that considers both payment
--              frequency AND payment timeliness.
--
-- Old Formula: (1 - ((days_since_last_repayment / loan_age) / 0.25)) × 100
-- New Formula: (1 - (((days_since_last_repayment + current_dpd) / 2) / loan_age) / 0.25) × 100
--
-- Impact:
-- - Loans with high DPD will have lower (worse) repayment delay rates
-- - Loans with low DPD and recent payments will maintain high (good) rates
-- - Better reflects overall repayment health by combining recency and timeliness
--
-- Author: Seeds Metrics Team
-- Date: 2025-11-21
-- ============================================================================

BEGIN;

-- ============================================================================
-- PART 1: Update the trigger function update_loan_computed_fields()
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

    -- Calculate loan age in days
    v_loan_age := CURRENT_DATE - v_disbursement_date;

    -- ========================================================================
    -- UPDATED: Calculate repayment_delay_rate with NEW FORMULA
    -- ========================================================================
    -- New Formula: (1 - (((days_since_last_repayment + current_dpd) / 2) / loan_age) / 0.25) × 100
    --
    -- This formula combines:
    -- - days_since_last_repayment: Measures payment frequency/recency
    -- - current_dpd: Measures payment timeliness
    --
    -- The average of these two metrics provides a balanced view of repayment health
    --
    -- Edge cases:
    -- - If loan_age = 0, return 0
    -- - If days_since_last_repayment is NULL (no payments), use only current_dpd
    -- - Allow negative values (indicates poor repayment behavior)
    -- ========================================================================

    IF v_loan_age > 0 THEN
        IF v_days_since_last_repayment IS NOT NULL THEN
            -- Use average of days_since_last_repayment and current_dpd
            v_repayment_delay_rate := (1.0 - ((((v_days_since_last_repayment::DECIMAL + v_current_dpd::DECIMAL) / 2.0) / v_loan_age::DECIMAL) / 0.25)) * 100;
        ELSE
            -- No payments yet, use only current_dpd
            v_repayment_delay_rate := (1.0 - ((v_current_dpd::DECIMAL / v_loan_age::DECIMAL) / 0.25)) * 100;
        END IF;
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

        -- UPDATED: Repayment delay rate with new formula
        repayment_delay_rate = v_repayment_delay_rate,

        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION update_loan_computed_fields() IS
'Trigger function that recalculates all computed fields for a loan when repayments or schedules change.
Uses updated repayment_delay_rate formula: (1 - (((days_since_last_repayment + current_dpd) / 2) / loan_age) / 0.25) × 100';

-- ============================================================================
-- PART 2: Update the stored procedure recalculate_all_loan_fields()
-- ============================================================================

CREATE OR REPLACE FUNCTION recalculate_all_loan_fields()
RETURNS TABLE(
    total_loans_processed INTEGER,
    loans_updated INTEGER,
    execution_time_ms INTEGER
) AS $$
DECLARE
    v_start_time TIMESTAMP;
    v_end_time TIMESTAMP;
    v_total_loans INTEGER;
    v_loans_updated INTEGER;
BEGIN
    v_start_time := CURRENT_TIMESTAMP;

    -- Get total loan count
    SELECT COUNT(*) INTO v_total_loans FROM loans;

    -- ========================================================================
    -- STEP 1: Recalculate all computed fields with UPDATED repayment_delay_rate
    -- ========================================================================

    WITH loan_repayment_data AS (
        SELECT
            l.loan_id,
            l.loan_amount,
            l.interest_rate,
            l.loan_term_days,
            l.fee_amount,
            l.repayment_amount,
            l.max_dpd_ever,
            l.disbursement_date,
            l.first_payment_due_date,
            l.maturity_date,
            COALESCE(SUM(r.principal_paid), 0) as total_principal_paid,
            COALESCE(SUM(r.interest_paid), 0) as total_interest_paid,
            COALESCE(SUM(r.fees_paid), 0) as total_fees_paid,
            COALESCE(SUM(r.payment_amount), 0) as total_repayments,
            MIN(r.payment_date) as first_payment_date,
            MAX(r.payment_date) as last_payment_date,
            COUNT(r.repayment_id) as repayment_count
        FROM loans l
        LEFT JOIN repayments r ON l.loan_id = r.loan_id AND r.is_reversed = FALSE
        GROUP BY l.loan_id, l.loan_amount, l.interest_rate, l.loan_term_days,
                 l.fee_amount, l.repayment_amount, l.max_dpd_ever, l.disbursement_date,
                 l.first_payment_due_date, l.maturity_date
    ),
    calculated_fields AS (
        SELECT
            lrd.loan_id,
            -- Collections totals
            lrd.total_principal_paid,
            lrd.total_interest_paid,
            lrd.total_fees_paid,
            lrd.total_repayments,
            -- Outstanding balances
            lrd.loan_amount - lrd.total_principal_paid as principal_outstanding,
            (lrd.loan_amount * lrd.interest_rate * lrd.loan_term_days / 365) - lrd.total_interest_paid as interest_outstanding,
            COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid as fees_outstanding,
            (lrd.loan_amount - lrd.total_principal_paid) +
            ((lrd.loan_amount * lrd.interest_rate * lrd.loan_term_days / 365) - lrd.total_interest_paid) +
            (COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid) as total_outstanding,
            -- Actual outstanding (overdue amount based on time elapsed - BUSINESS DAYS)
            GREATEST(0,
                CASE
                    WHEN lrd.loan_term_days > 0 AND lrd.disbursement_date IS NOT NULL THEN
                        (
                            (lrd.loan_amount +
                             (lrd.loan_amount * lrd.interest_rate * lrd.loan_term_days / 365) +
                             COALESCE(lrd.fee_amount, 0))
                            / lrd.loan_term_days
                        ) *
                        LEAST(
                            count_business_days(lrd.disbursement_date, CURRENT_DATE),
                            lrd.loan_term_days
                        )
                        - lrd.total_repayments
                    ELSE 0
                END
            ) as actual_outstanding,
            -- First payment tracking
            lrd.first_payment_date,
            lrd.first_payment_due_date,
            -- Days since last repayment
            CASE WHEN lrd.last_payment_date IS NOT NULL
                 THEN (CURRENT_DATE - lrd.last_payment_date)::INTEGER
                 ELSE NULL END as days_since_last_repayment,
            -- Days since due
            CASE
                WHEN lrd.first_payment_date IS NOT NULL AND lrd.first_payment_due_date IS NOT NULL THEN
                    (lrd.first_payment_date - lrd.first_payment_due_date)::INTEGER
                WHEN lrd.first_payment_date IS NULL AND lrd.first_payment_due_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.first_payment_due_date)::INTEGER
                ELSE 0
            END as days_since_due,
            -- Loan age
            (CURRENT_DATE - lrd.disbursement_date)::INTEGER as loan_age,

            -- Real loan tenure (including weekends)
            CASE
                WHEN lrd.first_payment_due_date IS NOT NULL AND lrd.maturity_date IS NOT NULL THEN
                    lrd.loan_term_days + count_weekend_days(lrd.first_payment_due_date, lrd.maturity_date)
                ELSE lrd.loan_term_days
            END as real_loan_tenure_days,

            -- Daily repayment amount
            CASE
                WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                    lrd.repayment_amount / lrd.loan_term_days
                ELSE 0
            END as daily_repayment_amount,

            -- Repayment days paid
            CASE
                WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                    lrd.total_repayments / (lrd.repayment_amount / lrd.loan_term_days)
                ELSE 0
            END as repayment_days_paid,

            -- Repayment days due today
            CASE
                WHEN lrd.first_payment_due_date IS NOT NULL THEN
                    CASE
                        WHEN LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)) >= lrd.first_payment_due_date THEN
                            count_business_days(
                                lrd.first_payment_due_date,
                                LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE))
                            )
                        ELSE 0
                    END
                ELSE 0
            END as repayment_days_due_today,

            -- Business days since disbursement
            CASE
                WHEN lrd.disbursement_date IS NOT NULL THEN
                    count_business_days(lrd.disbursement_date, CURRENT_DATE)
                ELSE 0
            END as business_days_since_disbursement,

            -- Current DPD (missed repayment days)
            GREATEST(0,
                CASE
                    WHEN lrd.first_payment_due_date IS NOT NULL AND lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                        CASE
                            WHEN LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)) >= lrd.first_payment_due_date THEN
                                count_business_days(
                                    lrd.first_payment_due_date,
                                    LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE))
                                ) - (lrd.total_repayments / (lrd.repayment_amount / lrd.loan_term_days))::INTEGER
                            ELSE 0
                        END
                    ELSE 0
                END
            ) as current_dpd,

            lrd.max_dpd_ever,

            -- FIMR: TRUE if NO repayment on or before first_payment_due_date AND due date has passed
            CASE
                WHEN lrd.first_payment_due_date IS NULL THEN TRUE
                WHEN EXISTS (
                    SELECT 1
                    FROM repayments r
                    WHERE r.loan_id = lrd.loan_id
                      AND r.payment_date <= lrd.first_payment_due_date
                      AND r.is_reversed = FALSE
                ) THEN FALSE
                WHEN lrd.first_payment_date IS NULL AND lrd.first_payment_due_date >= CURRENT_DATE THEN FALSE
                ELSE TRUE
            END as fimr_tagged

        FROM loan_repayment_data lrd
    ),
    -- ========================================================================
    -- NEW: Calculate repayment_delay_rate separately with UPDATED FORMULA
    -- ========================================================================
    calculated_fields_with_delay_rate AS (
        SELECT
            cf.*,
            -- UPDATED: Repayment delay rate with new formula
            -- Formula: (1 - (((days_since_last_repayment + current_dpd) / 2) / loan_age) / 0.25) × 100
            CASE
                WHEN cf.loan_age > 0 AND cf.days_since_last_repayment IS NOT NULL THEN
                    (1.0 - ((((cf.days_since_last_repayment::DECIMAL + cf.current_dpd::DECIMAL) / 2.0) / cf.loan_age::DECIMAL) / 0.25)) * 100
                WHEN cf.loan_age > 0 AND cf.days_since_last_repayment IS NULL THEN
                    -- No payments yet, use only current_dpd
                    (1.0 - ((cf.current_dpd::DECIMAL / cf.loan_age::DECIMAL) / 0.25)) * 100
                WHEN cf.loan_age = 0 THEN 0
                ELSE NULL
            END as repayment_delay_rate
        FROM calculated_fields cf
    )
    UPDATE loans l
    SET
        total_principal_paid = cf.total_principal_paid,
        total_interest_paid = cf.total_interest_paid,
        total_fees_paid = cf.total_fees_paid,
        total_repayments = cf.total_repayments,
        principal_outstanding = cf.principal_outstanding,
        interest_outstanding = cf.interest_outstanding,
        fees_outstanding = cf.fees_outstanding,
        total_outstanding = cf.total_outstanding,
        actual_outstanding = cf.actual_outstanding,
        first_payment_received_date = cf.first_payment_date,
        first_payment_due_date = cf.first_payment_due_date,
        first_payment_missed = (cf.first_payment_date IS NULL OR cf.first_payment_date > cf.first_payment_due_date),
        days_since_last_repayment = cf.days_since_last_repayment,
        days_since_due = cf.days_since_due,
        loan_age = cf.loan_age,
        current_dpd = cf.current_dpd,
        max_dpd_ever = GREATEST(cf.max_dpd_ever, cf.current_dpd),
        fimr_tagged = cf.fimr_tagged,
        early_indicator_tagged = (cf.current_dpd BETWEEN 1 AND 6),
        repayment_delay_rate = cf.repayment_delay_rate,
        daily_repayment_amount = cf.daily_repayment_amount,
        real_loan_tenure_days = cf.real_loan_tenure_days,
        repayment_days_paid = cf.repayment_days_paid,
        repayment_days_due_today = cf.repayment_days_due_today,
        business_days_since_disbursement = cf.business_days_since_disbursement,
        updated_at = CURRENT_TIMESTAMP
    FROM calculated_fields_with_delay_rate cf
    WHERE l.loan_id = cf.loan_id;

    GET DIAGNOSTICS v_loans_updated = ROW_COUNT;

    -- ========================================================================
    -- STEP 2: Recalculate behavior scores (timeliness_score, repayment_health)
    -- ========================================================================

    UPDATE loans l
    SET
        timeliness_score = CASE
            WHEN (SELECT COUNT(*) FROM repayments r WHERE r.loan_id = l.loan_id AND r.is_reversed = FALSE) = 0 THEN
                CASE WHEN (CURRENT_DATE - l.disbursement_date) < 7 THEN 100.00 ELSE 50.00 END
            WHEN (CURRENT_DATE - l.disbursement_date) < 7 THEN 100.00
            WHEN l.days_since_last_repayment IS NOT NULL AND l.loan_age > 0 THEN
                CASE
                    WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.15 THEN 95.00
                    WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.25 THEN 85.00
                    WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.35 THEN 70.00
                    WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.50 THEN 55.00
                    WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.75 THEN 35.00
                    ELSE 15.00
                END
            ELSE 50.00
        END,
        repayment_health = CASE
            WHEN (SELECT COUNT(*) FROM repayments r WHERE r.loan_id = l.loan_id AND r.is_reversed = FALSE) = 0 THEN
                CASE WHEN (CURRENT_DATE - l.disbursement_date) < 7 THEN 100.00 ELSE 50.00 END
            WHEN (CURRENT_DATE - l.disbursement_date) < 7 THEN 100.00
            ELSE
                GREATEST(0, LEAST(100,
                    100.00
                    - CASE
                        WHEN l.current_dpd = 0 THEN 0
                        WHEN l.current_dpd BETWEEN 1 AND 7 THEN 10
                        WHEN l.current_dpd BETWEEN 8 AND 15 THEN 20
                        WHEN l.current_dpd BETWEEN 16 AND 30 THEN 30
                        ELSE 40
                    END
                    - CASE
                        WHEN l.loan_amount > 0 AND l.total_principal_paid > 0 THEN
                            CASE
                                WHEN (l.total_principal_paid / l.loan_amount) > 0.75 THEN 0
                                WHEN (l.total_principal_paid / l.loan_amount) > 0.50 THEN 10
                                WHEN (l.total_principal_paid / l.loan_amount) > 0.25 THEN 20
                                ELSE 30
                            END
                        ELSE 15
                    END
                ))
        END,
        updated_at = CURRENT_TIMESTAMP;

    v_end_time := CURRENT_TIMESTAMP;

    -- Return summary
    RETURN QUERY SELECT
        v_total_loans,
        v_loans_updated,
        EXTRACT(MILLISECONDS FROM (v_end_time - v_start_time))::INTEGER;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION recalculate_all_loan_fields() IS
'Recalculates all computed fields for all loans using the updated repayment_delay_rate formula:
(1 - (((days_since_last_repayment + current_dpd) / 2) / loan_age) / 0.25) × 100.
Returns total loans processed, loans updated, and execution time.';

COMMIT;

