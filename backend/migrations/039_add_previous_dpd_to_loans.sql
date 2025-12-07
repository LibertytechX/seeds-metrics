-- Migration 039: Add previous_dpd snapshot column to loans
-- Purpose: Store the previous value of current_dpd for each loan so that the
-- frontend can show day-on-day DPD changes on the All Loans tab.

-- =============================================================================
-- PART 1: Schema change - add previous_dpd to loans
-- =============================================================================

ALTER TABLE loans
ADD COLUMN IF NOT EXISTS previous_dpd INTEGER;

COMMENT ON COLUMN loans.previous_dpd IS
  'Snapshot of current_dpd from the previous recalculate_all_loan_fields() run.';

-- =============================================================================
-- PART 2: Update recalculate_all_loan_fields to snapshot previous_dpd
-- =============================================================================

-- NOTE: This definition is based on the latest version from
-- 037_fix_dpd_for_fully_paid_loans.sql with an additional assignment:
--   previous_dpd = l.current_dpd
-- before current_dpd is overwritten with the newly computed value.

CREATE OR REPLACE FUNCTION recalculate_all_loan_fields()
RETURNS void AS $$
BEGIN
    -- Update all loans with recalculated fields using a single UPDATE with CTE
    UPDATE loans l
    SET
        -- Repayment totals
        total_principal_paid = cf.total_principal_paid,
        total_interest_paid = cf.total_interest_paid,
        total_fees_paid = cf.total_fees_paid,
        total_repayments = cf.total_repayments,

        -- Outstanding balances
        principal_outstanding = cf.principal_outstanding,
        interest_outstanding = cf.interest_outstanding,
        fees_outstanding = cf.fees_outstanding,
        total_outstanding = cf.total_outstanding,
        actual_outstanding = cf.actual_outstanding,

        -- Payment dates and status
        first_payment_received_date = cf.first_payment_date,
        first_payment_due_date = cf.first_payment_due_date,
        first_payment_missed = (cf.first_payment_date IS NULL OR cf.first_payment_date > cf.first_payment_due_date),
        days_since_last_repayment = cf.days_since_last_repayment,
        days_since_due = cf.days_since_due,
        loan_age = cf.loan_age,

        -- DPD and risk flags
        previous_dpd = l.current_dpd,
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
    FROM (
        SELECT
            lrd.loan_id,
            lrd.total_principal_paid,
            lrd.total_interest_paid,
            lrd.total_fees_paid,
            lrd.total_repayments,
            lrd.first_payment_date,
            lrd.last_payment_date,
            lrd.first_payment_due_date,

            -- Outstanding balances
            (lrd.loan_amount - lrd.total_principal_paid) as principal_outstanding,
            GREATEST(0, lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) as interest_outstanding,
            GREATEST(0, COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid) as fees_outstanding,
            (lrd.loan_amount - lrd.total_principal_paid) +
            (lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) +
            (COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid) as total_outstanding,

            -- Days calculations
            CASE
                WHEN lrd.last_payment_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.last_payment_date)::INTEGER
                ELSE NULL
            END as days_since_last_repayment,

            CASE
                WHEN lrd.first_payment_due_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.first_payment_due_date)::INTEGER
                ELSE NULL
            END as days_since_due,

            CASE
                WHEN lrd.disbursement_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.disbursement_date)::INTEGER
                ELSE 0
            END as loan_age,

            -- New fields
            CASE
                WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                    lrd.repayment_amount / lrd.loan_term_days
                ELSE 0
            END as daily_repayment_amount,

            CASE
                WHEN lrd.disbursement_date IS NOT NULL AND lrd.maturity_date IS NOT NULL THEN
                    count_business_days(lrd.disbursement_date, lrd.maturity_date)
                ELSE lrd.loan_term_days
            END as real_loan_tenure_days,

            CASE
                WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                    lrd.total_repayments / (lrd.repayment_amount / lrd.loan_term_days)
                ELSE 0
            END as repayment_days_paid,

            CASE
                WHEN lrd.first_payment_due_date IS NOT NULL AND lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
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

            CASE
                WHEN lrd.disbursement_date IS NOT NULL THEN
                    count_business_days(lrd.disbursement_date, CURRENT_DATE)
                ELSE 0
            END as business_days_since_disbursement,

            -- Actual outstanding (overdue amount based on time elapsed - BUSINESS DAYS)
            -- Formula: (daily_repayment_amount * repayment_days_due_today) - total_repayments
            GREATEST(0,
                CASE
                    WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 AND lrd.first_payment_due_date IS NOT NULL THEN
                        CASE
                            WHEN LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)) >= lrd.first_payment_due_date THEN
                                (lrd.repayment_amount / lrd.loan_term_days) *
                                count_business_days(
                                    lrd.first_payment_due_date,
                                    LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE))
                                ) - lrd.total_repayments
                            ELSE 0
                        END
                    ELSE 0
                END
            ) as actual_outstanding,

            -- Step 6: Calculate DPD (missed repayment days)
            -- FIX: Set DPD to 0 if actual_outstanding <= 0 (loan is fully paid off)
            CASE
                WHEN GREATEST(0,
                    (lrd.loan_amount - lrd.total_principal_paid) +
                    (lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) +
                    (COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid)
                ) <= 0 THEN 0
                ELSE
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
                    )
            END as current_dpd,

            lrd.max_dpd_ever,

            -- Risk indicators
            -- FIMR: TRUE if NO repayment on or before first_payment_due_date AND due date has passed
            CASE
                WHEN lrd.first_payment_due_date IS NULL THEN TRUE
                WHEN lrd.payment_on_due_date_exists THEN FALSE
                WHEN lrd.first_payment_date IS NULL AND lrd.first_payment_due_date >= CURRENT_DATE THEN FALSE
                ELSE TRUE
            END as fimr_tagged,

            -- Repayment Delay Rate calculation with new formula
            (1 - (((CASE
                WHEN lrd.last_payment_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.last_payment_date)::INTEGER
                ELSE 0
            END +
            CASE
                WHEN GREATEST(0,
                    (lrd.loan_amount - lrd.total_principal_paid) +
                    (lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) +
                    (COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid)
                ) <= 0 THEN 0
                ELSE
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
                    )
            END) / 2) / NULLIF(CASE
                WHEN lrd.disbursement_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.disbursement_date)::INTEGER
                ELSE 0
            END, 0)) / 0.25) * 100 as repayment_delay_rate

        FROM (
            SELECT
                l.loan_id,
                l.loan_amount,
                l.interest_rate,
                l.fee_amount,
                l.disbursement_date,
                l.maturity_date,
                l.loan_term_days,
                l.first_payment_due_date,
                l.loan_amount * (1 + l.interest_rate) + COALESCE(l.fee_amount, 0) as repayment_amount,
                COALESCE(SUM(r.principal_paid), 0) as total_principal_paid,
                COALESCE(SUM(r.interest_paid), 0) as total_interest_paid,
                COALESCE(SUM(r.fees_paid), 0) as total_fees_paid,
                COALESCE(SUM(r.payment_amount), 0) as total_repayments,
                MIN(r.payment_date) as first_payment_date,
                MAX(r.payment_date) as last_payment_date,
                COALESCE(MAX(r.dpd_at_payment), 0) as max_dpd_ever,
                EXISTS (
                    SELECT 1
                    FROM repayments r2
                    WHERE r2.loan_id = l.loan_id
                      AND r2.payment_date <= l.first_payment_due_date
                      AND (r2.is_reversed IS NULL OR r2.is_reversed = FALSE)
                ) as payment_on_due_date_exists
            FROM loans l
            LEFT JOIN repayments r ON l.loan_id = r.loan_id
                AND (r.is_reversed IS NULL OR r.is_reversed = FALSE)
            GROUP BY l.loan_id, l.loan_amount, l.interest_rate, l.fee_amount,
                     l.disbursement_date, l.maturity_date, l.loan_term_days, l.first_payment_due_date
        ) lrd
    ) cf
    WHERE l.loan_id = cf.loan_id;

    RAISE NOTICE 'Recalculated all loan fields successfully (including previous_dpd snapshot)';
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- PART 3: Seed previous_dpd for existing loans
-- =============================================================================

-- Run the recalculate function once so that previous_dpd is initialised to the
-- current current_dpd values before this migration.
SELECT recalculate_all_loan_fields();
