-- ============================================================================
-- Migration: Fix Behavior Scores for Brand-New Loans (0 Repayments)
-- ============================================================================
-- Purpose: Assign default scores to loans with no repayments yet
-- Issue: Loans like 19786 (just disbursed) have NULL scores because they have
--        no repayments. These should get default scores instead.
-- Date: 2025-11-05
-- ============================================================================

-- ============================================================================
-- STEP 1: Backfill loans with 0 repayments
-- ============================================================================
-- For brand-new loans (< 7 days old) with no repayments: assign 100
-- For older loans with no repayments: assign 50 (neutral score)

UPDATE loans l
SET 
    timeliness_score = CASE
        WHEN (CURRENT_DATE - l.disbursement_date) < 7 THEN 100.00
        ELSE 50.00
    END,
    repayment_health = CASE
        WHEN (CURRENT_DATE - l.disbursement_date) < 7 THEN 100.00
        ELSE 50.00
    END
WHERE timeliness_score IS NULL
  AND repayment_health IS NULL
  AND NOT EXISTS (
    SELECT 1 FROM repayments r 
    WHERE r.loan_id = l.loan_id 
    AND r.is_reversed = FALSE
  );

-- ============================================================================
-- STEP 2: Update Trigger to Handle Brand-New Loans
-- ============================================================================
-- Modify the trigger to assign default scores when there are 0 repayments

CREATE OR REPLACE FUNCTION update_loan_behavior_scores()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_disbursement_date DATE;
    v_loan_age INTEGER;
    v_days_since_last_repayment INTEGER;
    v_current_dpd INTEGER;
    v_loan_amount DECIMAL(15, 2);
    v_total_principal_paid DECIMAL(15, 2);
    v_repayment_count INTEGER;
    v_timeliness_score DECIMAL(5, 2);
    v_repayment_health DECIMAL(5, 2);
BEGIN
    v_loan_id := NEW.loan_id;

    -- Get loan details
    SELECT 
        disbursement_date,
        loan_age,
        days_since_last_repayment,
        current_dpd,
        loan_amount,
        total_principal_paid
    INTO 
        v_disbursement_date,
        v_loan_age,
        v_days_since_last_repayment,
        v_current_dpd,
        v_loan_amount,
        v_total_principal_paid
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Count repayments
    SELECT COUNT(*) INTO v_repayment_count
    FROM repayments
    WHERE loan_id = v_loan_id
    AND is_reversed = FALSE;

    -- Calculate Timeliness Score
    IF v_repayment_count = 0 THEN
        -- Brand-new loans (< 7 days): 100, older loans with no repayments: 50
        IF (CURRENT_DATE - v_disbursement_date) < 7 THEN
            v_timeliness_score := 100.00;
        ELSE
            v_timeliness_score := 50.00;
        END IF;
    ELSIF (CURRENT_DATE - v_disbursement_date) < 7 THEN
        v_timeliness_score := 100.00;
    ELSIF v_days_since_last_repayment IS NOT NULL AND v_loan_age > 0 THEN
        CASE
            WHEN (v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) < 0.15 THEN v_timeliness_score := 95.00;
            WHEN (v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) < 0.25 THEN v_timeliness_score := 85.00;
            WHEN (v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) < 0.35 THEN v_timeliness_score := 70.00;
            WHEN (v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) < 0.50 THEN v_timeliness_score := 55.00;
            WHEN (v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) < 0.75 THEN v_timeliness_score := 35.00;
            ELSE v_timeliness_score := 15.00;
        END CASE;
    ELSE
        v_timeliness_score := 50.00;
    END IF;

    -- Calculate Repayment Health
    IF v_repayment_count = 0 THEN
        -- Brand-new loans (< 7 days): 100, older loans with no repayments: 50
        IF (CURRENT_DATE - v_disbursement_date) < 7 THEN
            v_repayment_health := 100.00;
        ELSE
            v_repayment_health := 50.00;
        END IF;
    ELSIF (CURRENT_DATE - v_disbursement_date) < 7 THEN
        v_repayment_health := 100.00;
    ELSE
        v_repayment_health := 100.00;
        
        -- Penalty for high DPD (max -40 points)
        IF v_current_dpd = 0 THEN
            v_repayment_health := v_repayment_health - 0;
        ELSIF v_current_dpd BETWEEN 1 AND 7 THEN
            v_repayment_health := v_repayment_health - 10;
        ELSIF v_current_dpd BETWEEN 8 AND 15 THEN
            v_repayment_health := v_repayment_health - 20;
        ELSIF v_current_dpd BETWEEN 16 AND 30 THEN
            v_repayment_health := v_repayment_health - 30;
        ELSE
            v_repayment_health := v_repayment_health - 40;
        END IF;
        
        -- Penalty for low repayment progress (max -30 points)
        IF v_loan_amount > 0 AND v_total_principal_paid > 0 THEN
            IF (v_total_principal_paid / v_loan_amount) > 0.75 THEN
                v_repayment_health := v_repayment_health - 0;
            ELSIF (v_total_principal_paid / v_loan_amount) > 0.50 THEN
                v_repayment_health := v_repayment_health - 10;
            ELSIF (v_total_principal_paid / v_loan_amount) > 0.25 THEN
                v_repayment_health := v_repayment_health - 20;
            ELSE
                v_repayment_health := v_repayment_health - 30;
            END IF;
        ELSE
            v_repayment_health := v_repayment_health - 15;
        END IF;
        
        -- Penalty for infrequent payments (max -30 points)
        IF v_days_since_last_repayment IS NOT NULL AND v_loan_age > 0 THEN
            IF (v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) < 0.20 THEN
                v_repayment_health := v_repayment_health - 0;
            ELSIF (v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) < 0.35 THEN
                v_repayment_health := v_repayment_health - 10;
            ELSIF (v_days_since_last_repayment::DECIMAL / v_loan_age::DECIMAL) < 0.50 THEN
                v_repayment_health := v_repayment_health - 20;
            ELSE
                v_repayment_health := v_repayment_health - 30;
            END IF;
        ELSE
            v_repayment_health := v_repayment_health - 15;
        END IF;
        
        -- Ensure score is between 0 and 100
        v_repayment_health := GREATEST(0, LEAST(100, v_repayment_health));
    END IF;

    -- Update the loans table
    UPDATE loans
    SET 
        timeliness_score = v_timeliness_score,
        repayment_health = v_repayment_health
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STEP 3: Verification
-- ============================================================================

DO $$
DECLARE
    v_total_loans INTEGER;
    v_with_timeliness INTEGER;
    v_with_health INTEGER;
    v_avg_timeliness DECIMAL(5, 2);
    v_avg_health DECIMAL(5, 2);
    v_null_timeliness INTEGER;
    v_null_health INTEGER;
BEGIN
    SELECT 
        COUNT(*),
        COUNT(timeliness_score),
        COUNT(repayment_health),
        ROUND(AVG(timeliness_score), 2),
        ROUND(AVG(repayment_health), 2),
        COUNT(*) - COUNT(timeliness_score),
        COUNT(*) - COUNT(repayment_health)
    INTO 
        v_total_loans,
        v_with_timeliness,
        v_with_health,
        v_avg_timeliness,
        v_avg_health,
        v_null_timeliness,
        v_null_health
    FROM loans;

    RAISE NOTICE '============================================';
    RAISE NOTICE 'Fix New Loans Behavior Scores';
    RAISE NOTICE '============================================';
    RAISE NOTICE 'Total Loans: %', v_total_loans;
    RAISE NOTICE 'Loans with Timeliness Score: % (%.2f%%)', v_with_timeliness, (v_with_timeliness::DECIMAL / v_total_loans * 100);
    RAISE NOTICE 'Loans with Repayment Health: % (%.2f%%)', v_with_health, (v_with_health::DECIMAL / v_total_loans * 100);
    RAISE NOTICE 'NULL Timeliness Scores: %', v_null_timeliness;
    RAISE NOTICE 'NULL Repayment Health: %', v_null_health;
    RAISE NOTICE 'Average Timeliness Score: %', v_avg_timeliness;
    RAISE NOTICE 'Average Repayment Health: %', v_avg_health;
    RAISE NOTICE '============================================';
END $$;

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

