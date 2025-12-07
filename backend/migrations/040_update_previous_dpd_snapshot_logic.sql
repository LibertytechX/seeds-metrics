-- Migration 040: Update previous_dpd snapshot logic to be date-aware
--
-- Goal:
--   Ensure previous_dpd represents the last DPD value from the *previous day*,
--   not simply the previous run of recalculate_all_loan_fields().
--
--   We achieve this by only updating previous_dpd when the loan's last
--   updated_at::date is strictly before CURRENT_DATE. This means:
--     - On the first recalculation on a new day, previous_dpd is set to the
--       prior current_dpd.
--     - Subsequent recalculations on the same day will leave previous_dpd
--       unchanged, even if current_dpd moves intra-day.
--
--   This function definition is based on the version introduced in
--   039_add_previous_dpd_to_loans.sql, with only the previous_dpd assignment
--   changed to be conditional on updated_at::date < CURRENT_DATE.

CREATE OR REPLACE FUNCTION recalculate_all_loan_fields()
RETURNS void AS $$
BEGIN
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
        previous_dpd = CASE
            WHEN l.updated_at IS NULL OR l.updated_at::date < CURRENT_DATE THEN l.current_dpd
            ELSE l.previous_dpd
        END,
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
            (lrd.loan_amount - lrd.total_principal_paid) AS principal_outstanding,
            GREATEST(0, lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) AS interest_outstanding,
            GREATEST(0, COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid) AS fees_outstanding,
            (lrd.loan_amount - lrd.total_principal_paid) +
            (lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) +
            (COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid) AS total_outstanding,

            -- Days calculations
            CASE
                WHEN lrd.last_payment_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.last_payment_date)::INTEGER
                ELSE NULL
            END AS days_since_last_repayment,

            CASE
                WHEN lrd.first_payment_due_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.first_payment_due_date)::INTEGER
                ELSE NULL
            END AS days_since_due,

            CASE
                WHEN lrd.disbursement_date IS NOT NULL THEN
                    (CURRENT_DATE - lrd.disbursement_date)::INTEGER
                ELSE 0
            END AS loan_age,

            -- New fields
            CASE
                WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                    lrd.repayment_amount / lrd.loan_term_days
                ELSE 0
            END AS daily_repayment_amount,

            CASE
                WHEN lrd.disbursement_date IS NOT NULL AND lrd.maturity_date IS NOT NULL THEN
                    count_business_days(lrd.disbursement_date, lrd.maturity_date)
                ELSE lrd.loan_term_days
            END AS real_loan_tenure_days,

            CASE
                WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN
                    lrd.total_repayments / (lrd.repayment_amount / lrd.loan_term_days)
                ELSE 0
            END AS repayment_days_paid,

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
            END AS repayment_days_due_today,

            CASE
                WHEN lrd.disbursement_date IS NOT NULL THEN
                    count_business_days(lrd.disbursement_date, CURRENT_DATE)
                ELSE 0
            END AS business_days_since_disbursement,

            -- Actual outstanding (overdue amount based on time elapsed - BUSINESS DAYS)
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
            ) AS actual_outstanding,

            -- DPD calculation with fully-paid fix
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
            END AS current_dpd,

            lrd.max_dpd_ever,

            -- Risk indicators
            CASE
                WHEN lrd.first_payment_due_date IS NULL THEN TRUE
                WHEN lrd.payment_on_due_date_exists THEN FALSE
                WHEN lrd.first_payment_date IS NULL AND lrd.first_payment_due_date >= CURRENT_DATE THEN FALSE
                ELSE TRUE
            END AS fimr_tagged,

            -- Repayment Delay Rate calculation (unchanged from 039)
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
            END, 0)) / 0.25) * 100 AS repayment_delay_rate

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
                l.loan_amount * (1 + l.interest_rate) + COALESCE(l.fee_amount, 0) AS repayment_amount,
                COALESCE(SUM(r.principal_paid), 0) AS total_principal_paid,
                COALESCE(SUM(r.interest_paid), 0) AS total_interest_paid,
                COALESCE(SUM(r.fees_paid), 0) AS total_fees_paid,
                COALESCE(SUM(r.payment_amount), 0) AS total_repayments,
                MIN(r.payment_date) AS first_payment_date,
                MAX(r.payment_date) AS last_payment_date,
                COALESCE(MAX(r.dpd_at_payment), 0) AS max_dpd_ever,
                EXISTS (
                    SELECT 1
                    FROM repayments r2
                    WHERE r2.loan_id = l.loan_id
                      AND r2.payment_date <= l.first_payment_due_date
                      AND (r2.is_reversed IS NULL OR r2.is_reversed = FALSE)
                ) AS payment_on_due_date_exists
            FROM loans l
            LEFT JOIN repayments r ON l.loan_id = r.loan_id
                AND (r.is_reversed IS NULL OR r.is_reversed = FALSE)
            GROUP BY l.loan_id, l.loan_amount, l.interest_rate, l.fee_amount,
                     l.disbursement_date, l.maturity_date, l.loan_term_days, l.first_payment_due_date
        ) lrd
    ) cf
    WHERE l.loan_id = cf.loan_id;

    RAISE NOTICE 'Recalculated all loan fields successfully (previous_dpd is now date-aware)';
END;
$$ LANGUAGE plpgsql;
