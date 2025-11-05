-- ============================================================================
-- Migration: Comprehensive Loan Field Recalculation Procedure
-- ============================================================================
-- Purpose: Create a stored procedure to recalculate ALL computed fields for
--          all loans in a single operation. This is used by the "Refresh Fields"
--          button in the UI.
-- Date: 2025-11-05
-- ============================================================================

-- ============================================================================
-- STEP 1: Create Comprehensive Recalculation Procedure
-- ============================================================================
-- This procedure recalculates all computed fields for all loans by:
-- 1. Aggregating repayment data
-- 2. Calculating outstanding balances
-- 3. Computing DPD and risk indicators
-- 4. Calculating behavior scores (timeliness, health)

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
    -- STEP 1: Recalculate all computed fields using the same logic as triggers
    -- ========================================================================
    
    WITH loan_repayment_data AS (
        SELECT
            l.loan_id,
            l.loan_amount,
            l.interest_rate,
            l.loan_term_days,
            l.fee_amount,
            l.max_dpd_ever,
            l.disbursement_date,
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
                 l.fee_amount, l.max_dpd_ever, l.disbursement_date
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
            -- First payment tracking
            lrd.first_payment_date,
            COALESCE(
                (SELECT MIN(due_date) FROM loan_schedule WHERE loan_id = lrd.loan_id),
                lrd.disbursement_date + INTERVAL '30 days'
            ) as first_payment_due_date,
            -- Days since last repayment
            CASE WHEN lrd.last_payment_date IS NOT NULL 
                 THEN (CURRENT_DATE - lrd.last_payment_date)::INTEGER 
                 ELSE NULL END as days_since_last_repayment,
            -- Loan age
            (CURRENT_DATE - lrd.disbursement_date)::INTEGER as loan_age,
            -- DPD calculation
            COALESCE(
                (SELECT COALESCE(MAX(CURRENT_DATE - due_date), 0)
                 FROM loan_schedule
                 WHERE loan_id = lrd.loan_id
                 AND payment_status IN ('Pending', 'Partial')
                 AND due_date < CURRENT_DATE),
                GREATEST(0, CURRENT_DATE - lrd.disbursement_date - 30)
            ) as current_dpd,
            lrd.max_dpd_ever,
            -- Risk indicators
            (lrd.first_payment_date IS NULL OR lrd.first_payment_date > 
             COALESCE((SELECT MIN(due_date) FROM loan_schedule WHERE loan_id = lrd.loan_id),
                      lrd.disbursement_date + INTERVAL '30 days')) as fimr_tagged,
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
        first_payment_received_date = cf.first_payment_date,
        first_payment_due_date = cf.first_payment_due_date,
        first_payment_missed = (cf.first_payment_date IS NULL OR cf.first_payment_date > cf.first_payment_due_date),
        days_since_last_repayment = cf.days_since_last_repayment,
        loan_age = cf.loan_age,
        current_dpd = cf.current_dpd,
        max_dpd_ever = GREATEST(cf.max_dpd_ever, cf.current_dpd),
        fimr_tagged = cf.fimr_tagged,
        early_indicator_tagged = (cf.current_dpd BETWEEN 1 AND 6),
        repayment_delay_rate = cf.repayment_delay_rate,
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
                    - CASE
                        WHEN l.days_since_last_repayment IS NOT NULL AND l.loan_age > 0 THEN
                            CASE
                                WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.20 THEN 0
                                WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.35 THEN 10
                                WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.50 THEN 20
                                ELSE 30
                            END
                        ELSE 15
                    END
                ))
        END
    WHERE loan_id IS NOT NULL;
    
    v_end_time := CURRENT_TIMESTAMP;
    
    RETURN QUERY SELECT 
        v_total_loans,
        v_loans_updated,
        EXTRACT(EPOCH FROM (v_end_time - v_start_time))::INTEGER * 1000;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STEP 2: Verification
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '============================================';
    RAISE NOTICE 'Comprehensive Loan Recalculation Procedure';
    RAISE NOTICE '============================================';
    RAISE NOTICE 'Procedure created: recalculate_all_loan_fields()';
    RAISE NOTICE 'This procedure recalculates:';
    RAISE NOTICE '  - All outstanding balances';
    RAISE NOTICE '  - DPD and risk indicators';
    RAISE NOTICE '  - Repayment delay rates';
    RAISE NOTICE '  - Timeliness scores';
    RAISE NOTICE '  - Repayment health scores';
    RAISE NOTICE '============================================';
END $$;

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

