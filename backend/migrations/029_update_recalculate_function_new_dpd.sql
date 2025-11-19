-- ============================================================================
-- Migration 029: Update recalculate_all_loan_fields() with New DPD Logic
-- ============================================================================
-- Description: Updates the recalculate_all_loan_fields() stored procedure to
--              use the new DPD calculation methodology based on missed
--              repayment days instead of days since last payment.
--
-- Author: Seeds Metrics Team
-- Date: 2025-11-09
-- ============================================================================

BEGIN;

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
    -- STEP 1: Recalculate all computed fields using NEW DPD methodology
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
            -- Formula: ((total_expected_repayment / loan_term_business_days) * days_elapsed_business_days) - total_repayments
            GREATEST(0,
                CASE
                    WHEN lrd.loan_term_days > 0 AND lrd.disbursement_date IS NOT NULL THEN
                        (
                            -- Total expected repayment (principal + interest + fees)
                            (lrd.loan_amount +
                             (lrd.loan_amount * lrd.interest_rate * lrd.loan_term_days / 365) +
                             COALESCE(lrd.fee_amount, 0))
                            / lrd.loan_term_days  -- loan_term_days is already business days only
                        ) *
                        -- Days elapsed in business days (capped at loan tenure, minimum 0)
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

            -- ================================================================
            -- NEW DPD CALCULATION METHODOLOGY
            -- ================================================================

            -- Step 1: Calculate real loan tenure (including weekends)
            CASE
                WHEN lrd.first_payment_due_date IS NOT NULL AND lrd.maturity_date IS NOT NULL THEN
                    lrd.loan_term_days + count_weekend_days(lrd.first_payment_due_date, lrd.maturity_date)
                ELSE lrd.loan_term_days
            END as real_loan_tenure_days,

            -- Step 2: Calculate daily repayment amount
            CASE
                WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                    lrd.repayment_amount / lrd.loan_term_days
                ELSE 0
            END as daily_repayment_amount,

            -- Step 3: Calculate repayment days paid
            CASE
                WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                    lrd.total_repayments / (lrd.repayment_amount / lrd.loan_term_days)
                ELSE 0
            END as repayment_days_paid,

            -- Step 4: Calculate repayment days due today
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

            -- Step 5: Calculate business days since disbursement
            CASE
                WHEN lrd.disbursement_date IS NOT NULL THEN
                    count_business_days(lrd.disbursement_date, CURRENT_DATE)
                ELSE 0
            END as business_days_since_disbursement,

            -- Step 6: Calculate DPD (missed repayment days)
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

            -- Risk indicators
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
            END as fimr_tagged,

            -- Repayment delay rate
            CASE
                WHEN (CURRENT_DATE - lrd.disbursement_date)::INTEGER > 0 AND
                     lrd.last_payment_date IS NOT NULL
                THEN (1.0 - (((CURRENT_DATE - lrd.last_payment_date)::DECIMAL /
                      (CURRENT_DATE - lrd.disbursement_date)::DECIMAL) / 0.25)) * 100
                WHEN (CURRENT_DATE - lrd.disbursement_date)::INTEGER = 0 THEN 0
                ELSE NULL
            END as repayment_delay_rate
        FROM loan_repayment_data lrd
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
        -- New fields
        daily_repayment_amount = cf.daily_repayment_amount,
        real_loan_tenure_days = cf.real_loan_tenure_days,
        repayment_days_paid = cf.repayment_days_paid,
        repayment_days_due_today = cf.repayment_days_due_today,
        business_days_since_disbursement = cf.business_days_since_disbursement,
        updated_at = CURRENT_TIMESTAMP
    FROM calculated_fields cf
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
'Recalculates all computed fields for all loans using the new DPD methodology based on missed repayment days. Returns total loans processed, loans updated, and execution time.';

COMMIT;

