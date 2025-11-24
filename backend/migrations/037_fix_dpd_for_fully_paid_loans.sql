-- Migration: Fix DPD calculation for fully paid loans
-- Issue: Loans with actual_outstanding <= 0 should have current_dpd = 0
-- Affected: 4,535 loans (3,643 closed, 892 active but overpaid)
-- Date: 2025-11-24

-- ============================================================================
-- PART 1: Update the trigger function to set DPD = 0 when actual_outstanding <= 0
-- ============================================================================

CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
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
    v_maturity_date DATE;
    v_max_dpd INTEGER;
    v_payment_on_due_date_exists BOOLEAN;
    v_days_since_last_repayment INTEGER;
    v_days_since_due INTEGER;
    v_loan_id VARCHAR(100);
    v_repayment_amount DECIMAL(15, 2);
    v_daily_repayment_amount DECIMAL(15, 2);
    v_real_loan_tenure_days INTEGER;
    v_repayment_days_paid DECIMAL(15, 2);
    v_repayment_days_due_today INTEGER;
    v_calculation_end_date DATE;
    v_total_outstanding DECIMAL(15, 2);
    v_actual_outstanding DECIMAL(15, 2);
BEGIN
    -- Determine loan_id based on trigger source
    IF TG_TABLE_NAME = 'repayments' THEN
        v_loan_id := NEW.loan_id;
    ELSIF TG_TABLE_NAME = 'loan_schedule' THEN
        v_loan_id := NEW.loan_id;
    ELSE
        RETURN NEW;
    END IF;

    -- Get loan details
    SELECT
        loan_amount,
        interest_rate,
        disbursement_date,
        loan_term_days,
        fee_amount,
        maturity_date,
        first_payment_due_date
    INTO
        v_loan_amount,
        v_interest_rate,
        v_disbursement_date,
        v_loan_term_days,
        v_fee_amount,
        v_maturity_date,
        v_first_due_date
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Calculate total repayments
    SELECT
        COALESCE(SUM(principal_paid), 0),
        COALESCE(SUM(interest_paid), 0),
        COALESCE(SUM(fees_paid), 0),
        COALESCE(SUM(payment_amount), 0),
        MIN(payment_date),
        MAX(payment_date)
    INTO
        v_total_principal_paid,
        v_total_interest_paid,
        v_total_fees_paid,
        v_total_repayments,
        v_first_payment_date,
        v_last_payment_date
    FROM repayments
    WHERE loan_id = v_loan_id
      AND (is_reversed IS NULL OR is_reversed = FALSE);

    -- Calculate days since last repayment
    IF v_last_payment_date IS NOT NULL THEN
        v_days_since_last_repayment := (CURRENT_DATE - v_last_payment_date)::INTEGER;
    ELSE
        v_days_since_last_repayment := NULL;
    END IF;

    -- Calculate days since due
    IF v_first_due_date IS NOT NULL THEN
        v_days_since_due := (CURRENT_DATE - v_first_due_date)::INTEGER;
    ELSE
        v_days_since_due := NULL;
    END IF;

    -- Calculate repayment amount (total amount due)
    v_repayment_amount := v_loan_amount * (1 + v_interest_rate) + COALESCE(v_fee_amount, 0);

    -- Step 1: Calculate real loan tenure (business days from disbursement to maturity)
    IF v_disbursement_date IS NOT NULL AND v_maturity_date IS NOT NULL THEN
        v_real_loan_tenure_days := count_business_days(v_disbursement_date, v_maturity_date);
    ELSE
        v_real_loan_tenure_days := v_loan_term_days;
    END IF;

    -- Step 2: Calculate daily repayment amount
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

    -- Step 4: Calculate repayment days due today
    IF v_first_due_date IS NOT NULL THEN
        -- Use the earlier of CURRENT_DATE or maturity_date
        v_calculation_end_date := LEAST(CURRENT_DATE, COALESCE(v_maturity_date, CURRENT_DATE));

        -- Count business days from first_payment_due_date to calculation_end_date
        IF v_calculation_end_date >= v_first_due_date THEN
            v_repayment_days_due_today := count_business_days(v_first_due_date, v_calculation_end_date);
        ELSE
            v_repayment_days_due_today := 0;
        END IF;
    ELSE
        v_repayment_days_due_today := 0;
    END IF;

    -- Calculate total_outstanding
    v_total_outstanding := (v_loan_amount - v_total_principal_paid) +
                          (v_loan_amount * v_interest_rate - v_total_interest_paid) +
                          (COALESCE(v_fee_amount, 0) - v_total_fees_paid);

    -- Calculate actual_outstanding (overdue amount based on time elapsed - BUSINESS DAYS)
    -- Formula: (daily_repayment_amount * repayment_days_due_today) - total_repayments
    -- This represents: "what should have been paid by now" - "what was actually paid"
    IF v_daily_repayment_amount > 0 AND v_repayment_days_due_today > 0 THEN
        v_actual_outstanding := GREATEST(0,
            (v_daily_repayment_amount * v_repayment_days_due_today) - v_total_repayments
        );
    ELSE
        v_actual_outstanding := 0;
    END IF;

    -- Step 5: Calculate DPD (missed repayment days)
    -- FIX: Set DPD to 0 if actual_outstanding <= 0 (loan is fully paid off)
    IF v_actual_outstanding <= 0 THEN
        v_current_dpd := 0;
    ELSE
        v_current_dpd := GREATEST(0, v_repayment_days_due_today - v_repayment_days_paid::INTEGER);
    END IF;

    -- Get max DPD ever
    SELECT COALESCE(MAX(dpd_at_payment), 0) INTO v_max_dpd
    FROM repayments
    WHERE loan_id = v_loan_id;

    -- Check if payment exists on or before first_payment_due_date
    SELECT EXISTS (
        SELECT 1
        FROM repayments
        WHERE loan_id = v_loan_id
          AND payment_date <= v_first_due_date
    ) INTO v_payment_on_due_date_exists;

    -- Update the loans table
    UPDATE loans
    SET
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        total_repayments = v_total_repayments,
        principal_outstanding = v_loan_amount - v_total_principal_paid,
        interest_outstanding = GREATEST(0, v_loan_amount * v_interest_rate - v_total_interest_paid),
        fees_outstanding = GREATEST(0, v_fee_amount - v_total_fees_paid),
        total_outstanding = v_total_outstanding,
        actual_outstanding = v_actual_outstanding,
        first_payment_received_date = v_first_payment_date,
        first_payment_due_date = v_first_due_date,
        first_payment_missed = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),
        current_dpd = v_current_dpd,
        max_dpd_ever = GREATEST(v_max_dpd, v_current_dpd),
        days_since_due = v_days_since_due,
        days_since_last_repayment = v_days_since_last_repayment,
        loan_age = CASE
            WHEN v_disbursement_date IS NULL THEN 0
            ELSE (CURRENT_DATE - v_disbursement_date)::INTEGER
        END,
        fimr_tagged = CASE
            WHEN v_first_due_date IS NULL THEN TRUE
            WHEN v_payment_on_due_date_exists THEN FALSE
            WHEN v_first_payment_date IS NULL AND v_first_due_date >= CURRENT_DATE THEN FALSE
            ELSE TRUE
        END,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),
        -- New fields
        daily_repayment_amount = v_daily_repayment_amount,
        real_loan_tenure_days = v_real_loan_tenure_days,
        repayment_days_paid = v_repayment_days_paid,
        repayment_days_due_today = v_repayment_days_due_today,
        business_days_since_disbursement = CASE
            WHEN v_disbursement_date IS NOT NULL THEN
                count_business_days(v_disbursement_date, CURRENT_DATE)
            ELSE 0
        END,
        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PART 2: Update the recalculate_all_loan_fields stored procedure
-- ============================================================================

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
            -- This represents: "what should have been paid by now" - "what was actually paid"
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

    RAISE NOTICE 'Recalculated all loan fields successfully';
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- PART 3: Apply the fix to all affected loans
-- ============================================================================

-- Run the recalculate function to fix all loans
SELECT recalculate_all_loan_fields();

-- Verify the fix
SELECT
    COUNT(*) as total_loans_fixed,
    SUM(CASE WHEN status = 'Closed' THEN 1 ELSE 0 END) as closed_loans_fixed
FROM loans
WHERE actual_outstanding <= 0 AND current_dpd = 0;

-- Show remaining issues (should be 0)
SELECT
    COUNT(*) as remaining_issues
FROM loans
WHERE actual_outstanding <= 0 AND current_dpd > 0;

