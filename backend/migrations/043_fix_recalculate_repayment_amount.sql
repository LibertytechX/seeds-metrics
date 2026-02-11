-- Migration: Fix recalculate_all_loan_fields to use stored repayment_amount
-- Issue: The inner subquery recalculates repayment_amount as:
--        loan_amount * (1 + interest_rate) + COALESCE(fee_amount, 0)
--        instead of using the stored repayment_amount column from the loans table.
--        This causes wrong daily_repayment_amount for loans like BNPL (loan 20080)
--        where repayment_amount != loan_amount * (1 + interest_rate) + fee_amount.
-- Fix: Use l.repayment_amount directly from the loans table.
-- Date: 2026-02-11

CREATE OR REPLACE FUNCTION recalculate_all_loan_fields()
RETURNS VOID AS $$
BEGIN
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
        previous_dpd = CASE WHEN l.updated_at IS NULL OR l.updated_at::date < CURRENT_DATE THEN l.current_dpd ELSE l.previous_dpd END,
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
    FROM (
        SELECT
            lrd.loan_id, lrd.total_principal_paid, lrd.total_interest_paid,
            lrd.total_fees_paid, lrd.total_repayments, lrd.first_payment_date,
            lrd.last_payment_date, lrd.first_payment_due_date,
            (lrd.loan_amount - lrd.total_principal_paid) AS principal_outstanding,
            GREATEST(0, lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) AS interest_outstanding,
            GREATEST(0, COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid) AS fees_outstanding,
            (lrd.loan_amount - lrd.total_principal_paid) + (lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) + (COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid) AS total_outstanding,
            CASE WHEN lrd.last_payment_date IS NOT NULL THEN (CURRENT_DATE - lrd.last_payment_date)::INTEGER ELSE NULL END AS days_since_last_repayment,
            CASE WHEN lrd.first_payment_due_date IS NOT NULL THEN (CURRENT_DATE - lrd.first_payment_due_date)::INTEGER ELSE NULL END AS days_since_due,
            CASE WHEN lrd.disbursement_date IS NOT NULL THEN (CURRENT_DATE - lrd.disbursement_date)::INTEGER ELSE 0 END AS loan_age,
            CASE WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN lrd.repayment_amount / lrd.loan_term_days ELSE 0 END AS daily_repayment_amount,
            CASE WHEN lrd.disbursement_date IS NOT NULL AND lrd.maturity_date IS NOT NULL THEN count_business_days_excluding_holidays(lrd.disbursement_date, lrd.maturity_date, lrd.officer_id) ELSE lrd.loan_term_days END AS real_loan_tenure_days,
            CASE WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN lrd.total_repayments / (lrd.repayment_amount / lrd.loan_term_days) ELSE 0 END AS repayment_days_paid,
            CASE WHEN lrd.first_payment_due_date IS NOT NULL AND lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN CASE WHEN LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)) >= lrd.first_payment_due_date THEN count_business_days_excluding_holidays(lrd.first_payment_due_date, LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)), lrd.officer_id) ELSE 0 END ELSE 0 END AS repayment_days_due_today,
            CASE WHEN lrd.disbursement_date IS NOT NULL THEN count_business_days_excluding_holidays(lrd.disbursement_date, CURRENT_DATE, lrd.officer_id) ELSE 0 END AS business_days_since_disbursement,
            GREATEST(0, CASE WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 AND lrd.first_payment_due_date IS NOT NULL THEN CASE WHEN LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)) >= lrd.first_payment_due_date THEN (lrd.repayment_amount / lrd.loan_term_days) * count_business_days_excluding_holidays(lrd.first_payment_due_date, LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)), lrd.officer_id) - lrd.total_repayments ELSE 0 END ELSE 0 END) AS actual_outstanding,
            CASE WHEN GREATEST(0, (lrd.loan_amount - lrd.total_principal_paid) + (lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) + (COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid)) <= 0 THEN 0 ELSE GREATEST(0, CASE WHEN lrd.first_payment_due_date IS NOT NULL AND lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN CASE WHEN LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)) >= lrd.first_payment_due_date THEN count_business_days_excluding_holidays(lrd.first_payment_due_date, LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)), lrd.officer_id) - (lrd.total_repayments / (lrd.repayment_amount / lrd.loan_term_days))::INTEGER ELSE 0 END ELSE 0 END) END AS current_dpd,
            lrd.max_dpd_ever,
            CASE WHEN lrd.first_payment_due_date IS NULL THEN TRUE WHEN lrd.payment_on_due_date_exists THEN FALSE WHEN lrd.first_payment_date IS NULL AND lrd.first_payment_due_date >= CURRENT_DATE THEN FALSE ELSE TRUE END AS fimr_tagged,
            (1 - (((CASE WHEN lrd.last_payment_date IS NOT NULL THEN (CURRENT_DATE - lrd.last_payment_date)::INTEGER ELSE 0 END + CASE WHEN GREATEST(0, (lrd.loan_amount - lrd.total_principal_paid) + (lrd.loan_amount * lrd.interest_rate - lrd.total_interest_paid) + (COALESCE(lrd.fee_amount, 0) - lrd.total_fees_paid)) <= 0 THEN 0 ELSE GREATEST(0, CASE WHEN lrd.first_payment_due_date IS NOT NULL AND lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 THEN CASE WHEN LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)) >= lrd.first_payment_due_date THEN count_business_days_excluding_holidays(lrd.first_payment_due_date, LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)), lrd.officer_id) - (lrd.total_repayments / (lrd.repayment_amount / lrd.loan_term_days))::INTEGER ELSE 0 END ELSE 0 END) END) / 2) / NULLIF(CASE WHEN lrd.disbursement_date IS NOT NULL THEN (CURRENT_DATE - lrd.disbursement_date)::INTEGER ELSE 0 END, 0)) / 0.25) * 100 AS repayment_delay_rate
        FROM (
            SELECT
                l.loan_id, l.loan_amount, l.interest_rate, l.fee_amount,
                l.disbursement_date, l.maturity_date, l.loan_term_days,
                l.first_payment_due_date, l.officer_id,
                -- FIX: Use stored repayment_amount instead of recalculating
                -- OLD (wrong): l.loan_amount * (1 + l.interest_rate) + COALESCE(l.fee_amount, 0) AS repayment_amount,
                l.repayment_amount,
                COALESCE(SUM(r.principal_paid), 0) AS total_principal_paid,
                COALESCE(SUM(r.interest_paid), 0) AS total_interest_paid,
                COALESCE(SUM(r.fees_paid), 0) AS total_fees_paid,
                COALESCE(SUM(r.payment_amount), 0) AS total_repayments,
                MIN(r.payment_date) AS first_payment_date,
                MAX(r.payment_date) AS last_payment_date,
                COALESCE(MAX(r.dpd_at_payment), 0) AS max_dpd_ever,
                EXISTS (SELECT 1 FROM repayments r2 WHERE r2.loan_id = l.loan_id AND r2.payment_date <= l.first_payment_due_date AND (r2.is_reversed IS NULL OR r2.is_reversed = FALSE)) AS payment_on_due_date_exists
            FROM loans l
            LEFT JOIN repayments r ON l.loan_id = r.loan_id AND (r.is_reversed IS NULL OR r.is_reversed = FALSE)
            GROUP BY l.loan_id, l.loan_amount, l.interest_rate, l.fee_amount,
                     l.disbursement_date, l.maturity_date, l.loan_term_days,
                     l.first_payment_due_date, l.officer_id, l.repayment_amount
        ) lrd
    ) cf
    WHERE l.loan_id = cf.loan_id;

    RAISE NOTICE 'Recalculated all loan fields successfully (fixed: using stored repayment_amount).';
END;
$$ LANGUAGE plpgsql;

-- Run recalculate to fix all affected loans
SELECT recalculate_all_loan_fields();

-- Verify fix for loan 20080
SELECT loan_id, loan_amount, repayment_amount, daily_repayment_amount,
       loan_term_days, total_repayments, actual_outstanding, current_dpd
FROM loans WHERE loan_id = '20080';