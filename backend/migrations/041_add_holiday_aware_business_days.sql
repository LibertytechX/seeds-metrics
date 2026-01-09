-- Migration 041: Make DPD and related metrics holiday-aware
--
-- This migration introduces a holiday-aware business day counter that
-- excludes both company holidays and agent-specific holidays, and then
-- wires that into:
--   * The incremental trigger function update_loan_computed_fields()
--   * The full backfill function recalculate_all_loan_fields()
-- so that DPD (and other business-day-based metrics) are holiday-aware.

BEGIN;

-- =====================================================================
-- STEP 1: Holiday-aware business day counter
-- =====================================================================

CREATE OR REPLACE FUNCTION count_business_days_excluding_holidays(
    start_date DATE,
    end_date   DATE,
    officer_id VARCHAR
)
RETURNS INTEGER
LANGUAGE plpgsql
STABLE
AS $$
DECLARE
    v_base_business_days INTEGER;
    v_holiday_days       INTEGER;
BEGIN
    -- Handle NULL inputs
    IF start_date IS NULL OR end_date IS NULL THEN
        RETURN 0;
    END IF;

    -- Handle case where start_date is after end_date
    IF start_date > end_date THEN
        RETURN 0;
    END IF;

    -- First count weekday business days (Mon-Fri) using existing helper
    v_base_business_days := count_business_days(start_date, end_date);

    IF v_base_business_days = 0 THEN
        RETURN 0;
    END IF;

    -- Count unique holiday dates that fall on weekdays within the range.
    -- Includes:
    --   * Company-wide holidays (type = 'company') for all officers
    --   * Agent-specific holidays (type = 'agent') for the given officer_id
    SELECT COUNT(DISTINCT h.date)
    INTO v_holiday_days
    FROM holiday h
    WHERE h.date BETWEEN start_date AND end_date
      AND EXTRACT(ISODOW FROM h.date) BETWEEN 1 AND 5
      AND (
          h.type = 'company'
          OR (h.type = 'agent' AND officer_id IS NOT NULL AND h.agent_id::TEXT = officer_id)
      );

    RETURN GREATEST(0, v_base_business_days - COALESCE(v_holiday_days, 0));
END;
$$;

COMMENT ON FUNCTION count_business_days_excluding_holidays(DATE, DATE, VARCHAR) IS
'Counts business days (Mon-Fri) between two dates, excluding both company holidays and agent-specific holidays for the given officer_id.';


-- =====================================================================
-- STEP 2: Update incremental trigger function (repayment-based updates)
-- =====================================================================

CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_repayment_amount DECIMAL(15, 2);
    v_disbursement_date DATE;
    v_first_due_date DATE;
    v_maturity_date DATE;
    v_max_dpd_ever INTEGER;
    v_officer_id VARCHAR(50);

    v_total_principal_paid DECIMAL(15, 2);
    v_total_interest_paid DECIMAL(15, 2);
    v_total_fees_paid DECIMAL(15, 2);
    v_total_repayments DECIMAL(15, 2);
    v_first_payment_date DATE;
    v_last_payment_date DATE;
    v_repayment_count INTEGER;

    v_principal_outstanding DECIMAL(15, 2);
    v_interest_outstanding DECIMAL(15, 2);
    v_fees_outstanding DECIMAL(15, 2);
    v_total_outstanding DECIMAL(15, 2);

    v_days_since_last_repayment INTEGER;
    v_days_since_due INTEGER;
    v_loan_age INTEGER;
    v_current_dpd INTEGER;
    v_fimr_tagged BOOLEAN;
    v_repayment_delay_rate DECIMAL(5, 2);

    -- New DPD methodology variables
    v_weekend_days_in_tenure INTEGER;
    v_real_loan_tenure_days INTEGER;
    v_daily_repayment_amount DECIMAL(15, 2);
    v_repayment_days_paid DECIMAL(10, 2);
    v_repayment_days_due_today INTEGER;
    v_calculation_end_date DATE;
    v_business_days_since_disbursement INTEGER;
BEGIN
    -- Get loan_id from the repayment record
    v_loan_id := NEW.loan_id;

    -- Fetch loan details (including officer_id for holiday-aware logic)
    SELECT
        loan_amount, interest_rate, loan_term_days, fee_amount, repayment_amount,
        disbursement_date, first_payment_due_date, maturity_date, max_dpd_ever,
        officer_id
    INTO
        v_loan_amount, v_interest_rate, v_loan_term_days, v_fee_amount, v_repayment_amount,
        v_disbursement_date, v_first_due_date, v_maturity_date, v_max_dpd_ever,
        v_officer_id
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Aggregate repayment data (excluding reversed repayments)
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
    WHERE loan_id = v_loan_id AND is_reversed = FALSE;

    -- Calculate outstanding balances
    v_principal_outstanding := v_loan_amount - v_total_principal_paid;
    v_interest_outstanding := (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid;
    v_fees_outstanding := COALESCE(v_fee_amount, 0) - v_total_fees_paid;
    v_total_outstanding := v_principal_outstanding + v_interest_outstanding + v_fees_outstanding;

    -- Calculate days since last repayment
    IF v_last_payment_date IS NOT NULL THEN
        v_days_since_last_repayment := (CURRENT_DATE - v_last_payment_date)::INTEGER;
    ELSE
        v_days_since_last_repayment := NULL;
    END IF;

    -- Calculate days since due
    IF v_first_payment_date IS NOT NULL AND v_first_due_date IS NOT NULL THEN
        v_days_since_due := (v_first_payment_date - v_first_due_date)::INTEGER;
    ELSIF v_first_payment_date IS NULL AND v_first_due_date IS NOT NULL THEN
        v_days_since_due := (CURRENT_DATE - v_first_due_date)::INTEGER;
    ELSE
        v_days_since_due := 0;
    END IF;

    -- Calculate loan age (calendar days)
    v_loan_age := (CURRENT_DATE - v_disbursement_date)::INTEGER;

    -- =====================================================================
    -- NEW DPD CALCULATION METHODOLOGY (holiday-aware business days)
    -- =====================================================================

    -- Step 1: Calculate real loan tenure (including weekends)
    IF v_first_due_date IS NOT NULL AND v_maturity_date IS NOT NULL THEN
        v_weekend_days_in_tenure := count_weekend_days(v_first_due_date, v_maturity_date);
        v_real_loan_tenure_days := v_loan_term_days + v_weekend_days_in_tenure;
    ELSE
        v_real_loan_tenure_days := v_loan_term_days;
    END IF;

    -- Step 2: Calculate daily repayment amount (based on business days only)
    IF v_loan_term_days > 0 AND v_repayment_amount > 0 THEN
        v_daily_repayment_amount := v_repayment_amount / v_loan_term_days;
    ELSE
        v_daily_repayment_amount := 0;
    END IF;

    -- Step 3: Calculate repayment days paid
    IF v_daily_repayment_amount > 0 THEN
        v_repayment_days_paid := v_total_repayments / v_daily_repayment_amount;
    ELSE
        v_repayment_days_paid := 0;
    END IF;

    -- Step 4: Calculate repayment days due today (from first_payment_due_date)
    IF v_first_due_date IS NOT NULL THEN
        v_calculation_end_date := LEAST(CURRENT_DATE, COALESCE(v_maturity_date, CURRENT_DATE));

        IF v_calculation_end_date >= v_first_due_date THEN
            v_repayment_days_due_today := count_business_days_excluding_holidays(
                v_first_due_date,
                v_calculation_end_date,
                v_officer_id
            );
        ELSE
            v_repayment_days_due_today := 0;
        END IF;
    ELSE
        v_repayment_days_due_today := 0;
    END IF;

    -- Step 5: Calculate business days since disbursement (holiday-aware)
    IF v_disbursement_date IS NOT NULL THEN
        v_business_days_since_disbursement := count_business_days_excluding_holidays(
            v_disbursement_date,
            CURRENT_DATE,
            v_officer_id
        );
    ELSE
        v_business_days_since_disbursement := 0;
    END IF;

    -- Step 6: Calculate DPD (missed repayment days)
    v_current_dpd := GREATEST(0, v_repayment_days_due_today - v_repayment_days_paid::INTEGER);

    -- Calculate FIMR (First Installment Missed Rate)
    IF v_first_due_date IS NULL THEN
        v_fimr_tagged := TRUE;
    ELSIF EXISTS (
        SELECT 1 FROM repayments r
        WHERE r.loan_id = v_loan_id
          AND r.payment_date <= v_first_due_date
          AND r.is_reversed = FALSE
    ) THEN
        v_fimr_tagged := FALSE;
    ELSIF v_first_payment_date IS NULL AND v_first_due_date >= CURRENT_DATE THEN
        v_fimr_tagged := FALSE;
    ELSE
        v_fimr_tagged := TRUE;
    END IF;

    -- Calculate repayment delay rate (unchanged)
    IF v_loan_age > 0 AND v_last_payment_date IS NOT NULL THEN
        v_repayment_delay_rate := (1.0 - ((v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) / 0.25)) * 100;
    ELSIF v_loan_age = 0 THEN
        v_repayment_delay_rate := 0;
    ELSE
        v_repayment_delay_rate := NULL;
    END IF;

    -- Update the loan record with all computed fields
    UPDATE loans
    SET
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        total_repayments = v_total_repayments,
        principal_outstanding = v_principal_outstanding,
        interest_outstanding = v_interest_outstanding,
        fees_outstanding = v_fees_outstanding,
        total_outstanding = v_total_outstanding,
        first_payment_received_date = v_first_payment_date,
        first_payment_missed = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),
        days_since_last_repayment = v_days_since_last_repayment,
        days_since_due = v_days_since_due,
        loan_age = v_loan_age,
        current_dpd = v_current_dpd,
        max_dpd_ever = GREATEST(v_max_dpd_ever, v_current_dpd),
        fimr_tagged = v_fimr_tagged,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),
        repayment_delay_rate = v_repayment_delay_rate,
        -- New DPD methodology fields
        daily_repayment_amount = v_daily_repayment_amount,
        real_loan_tenure_days = v_real_loan_tenure_days,
        repayment_days_paid = v_repayment_days_paid,
        repayment_days_due_today = v_repayment_days_due_today,
        business_days_since_disbursement = v_business_days_since_disbursement,
        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION update_loan_computed_fields() IS
'Trigger function that recalculates all computed fields for a loan when a repayment or schedule entry is inserted or updated. Uses DPD methodology based on missed repayment days and holiday-aware business days.';

-- Make the column comment explicit about holidays
COMMENT ON COLUMN loans.business_days_since_disbursement IS
'Total business days (Mon-Fri) elapsed from disbursement_date to today, excluding company holidays and agent-specific holidays for the officer. Shows loan age in business business days.';


-- =====================================================================
-- STEP 3: Update full backfill function (recalculate_all_loan_fields)
-- =====================================================================

DROP FUNCTION IF EXISTS recalculate_all_loan_fields();

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
                    count_business_days_excluding_holidays(lrd.disbursement_date, lrd.maturity_date, lrd.officer_id)
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
                            count_business_days_excluding_holidays(
                                lrd.first_payment_due_date,
                                LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)),
                                lrd.officer_id
                            )
                        ELSE 0
                    END
                ELSE 0
            END AS repayment_days_due_today,

            CASE
                WHEN lrd.disbursement_date IS NOT NULL THEN
                    count_business_days_excluding_holidays(lrd.disbursement_date, CURRENT_DATE, lrd.officer_id)
                ELSE 0
            END AS business_days_since_disbursement,

            -- Actual outstanding (overdue amount based on time elapsed - BUSINESS DAYS)
            GREATEST(0,
                CASE
                    WHEN lrd.loan_term_days > 0 AND lrd.repayment_amount > 0 AND lrd.first_payment_due_date IS NOT NULL THEN
                        CASE
                            WHEN LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)) >= lrd.first_payment_due_date THEN
                                (lrd.repayment_amount / lrd.loan_term_days) *
                                count_business_days_excluding_holidays(
                                    lrd.first_payment_due_date,
                                    LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)),
                                    lrd.officer_id
                                ) - lrd.total_repayments
                            ELSE 0
                        END
                    ELSE 0
                END
            ) AS actual_outstanding,

            -- DPD calculation with fully-paid fix (holiday-aware business days)
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
                                        count_business_days_excluding_holidays(
                                            lrd.first_payment_due_date,
                                            LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)),
                                            lrd.officer_id
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

            -- Repayment Delay Rate calculation (unchanged except for holiday-aware DPD)
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
                                        count_business_days_excluding_holidays(
                                            lrd.first_payment_due_date,
                                            LEAST(CURRENT_DATE, COALESCE(lrd.maturity_date, CURRENT_DATE)),
                                            lrd.officer_id
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
                l.officer_id,
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
                     l.disbursement_date, l.maturity_date, l.loan_term_days,
                     l.first_payment_due_date, l.officer_id
        ) lrd
    ) cf
    WHERE l.loan_id = cf.loan_id;

    RAISE NOTICE 'Recalculated all loan fields successfully (previous_dpd is now date-aware and DPD is holiday-aware).';
END;
$$ LANGUAGE plpgsql;


-- =====================================================================
-- STEP 4: Backfill using the new holiday-aware logic
-- =====================================================================

-- Recalculate all loans so that current_dpd, business_days_since_disbursement,
-- actual_outstanding, repayment_delay_rate, etc. all pick up holiday-aware
-- business day calculations immediately after this migration.
SELECT recalculate_all_loan_fields();

COMMIT;

