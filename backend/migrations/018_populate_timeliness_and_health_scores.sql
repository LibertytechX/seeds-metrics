-- ============================================================================
-- Migration: Populate Timeliness Score and Repayment Health
-- ============================================================================
-- Purpose: Calculate timeliness_score and repayment_health based on loan
--          repayment behavior since these fields are not being synced from Django
-- Date: 2025-11-05
-- ============================================================================

-- ============================================================================
-- STEP 1: Calculate Timeliness Score
-- ============================================================================
-- Timeliness Score (0-100): Measures how timely payments are made
-- Formula: Based on ratio of on-time payments to total payments
-- - 100 = All payments on time or early
-- - 50 = Half payments on time
-- - 0 = No payments on time

UPDATE loans l
SET timeliness_score = (
    SELECT 
        CASE
            -- If no repayments, return NULL
            WHEN COUNT(r.repayment_id) = 0 THEN NULL
            -- If loan is very new (< 7 days), return 100 (benefit of doubt)
            WHEN (CURRENT_DATE - l.disbursement_date) < 7 THEN 100.00
            -- Calculate based on repayment frequency
            -- Good: Payment every 7-14 days = 80-100
            -- Fair: Payment every 15-30 days = 50-79
            -- Poor: Payment every 31+ days = 0-49
            ELSE
                CASE
                    -- Calculate average days between payments
                    WHEN l.days_since_last_repayment IS NOT NULL AND l.loan_age > 0 THEN
                        -- If payments are frequent (< 25% of loan age since last payment), score high
                        CASE
                            WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.15 THEN 95.00
                            WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.25 THEN 85.00
                            WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.35 THEN 70.00
                            WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.50 THEN 55.00
                            WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.75 THEN 35.00
                            ELSE 15.00
                        END
                    ELSE 50.00  -- Default to middle score if no data
                END
        END
    FROM repayments r
    WHERE r.loan_id = l.loan_id
    AND r.is_reversed = FALSE
    GROUP BY l.loan_id, l.disbursement_date, l.days_since_last_repayment, l.loan_age
)
WHERE l.loan_id IS NOT NULL;

-- ============================================================================
-- STEP 2: Calculate Repayment Health
-- ============================================================================
-- Repayment Health (0-100): Measures overall repayment pattern health
-- Formula: Combination of:
-- - Payment consistency (regular payments)
-- - Payment amount adequacy (paying enough to reduce principal)
-- - Current DPD status

UPDATE loans l
SET repayment_health = (
    SELECT 
        CASE
            -- If no repayments, return NULL
            WHEN COUNT(r.repayment_id) = 0 THEN NULL
            -- If loan is very new (< 7 days), return 100 (benefit of doubt)
            WHEN (CURRENT_DATE - l.disbursement_date) < 7 THEN 100.00
            -- Calculate based on multiple factors
            ELSE
                GREATEST(0, LEAST(100,
                    -- Base score: 100
                    100.00
                    -- Penalty for high DPD (max -40 points)
                    - CASE
                        WHEN l.current_dpd = 0 THEN 0
                        WHEN l.current_dpd BETWEEN 1 AND 7 THEN 10
                        WHEN l.current_dpd BETWEEN 8 AND 15 THEN 20
                        WHEN l.current_dpd BETWEEN 16 AND 30 THEN 30
                        ELSE 40
                    END
                    -- Penalty for low repayment progress (max -30 points)
                    - CASE
                        WHEN l.loan_amount > 0 AND l.total_principal_paid > 0 THEN
                            CASE
                                -- Good progress: Paid > 75% of principal
                                WHEN (l.total_principal_paid / l.loan_amount) > 0.75 THEN 0
                                -- Fair progress: Paid 50-75% of principal
                                WHEN (l.total_principal_paid / l.loan_amount) > 0.50 THEN 10
                                -- Poor progress: Paid 25-50% of principal
                                WHEN (l.total_principal_paid / l.loan_amount) > 0.25 THEN 20
                                -- Very poor progress: Paid < 25% of principal
                                ELSE 30
                            END
                        ELSE 15  -- Default penalty if no payments
                    END
                    -- Penalty for infrequent payments (max -30 points)
                    - CASE
                        WHEN l.days_since_last_repayment IS NOT NULL AND l.loan_age > 0 THEN
                            CASE
                                WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.20 THEN 0
                                WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.35 THEN 10
                                WHEN (l.days_since_last_repayment::DECIMAL / l.loan_age::DECIMAL) < 0.50 THEN 20
                                ELSE 30
                            END
                        ELSE 15  -- Default penalty if no data
                    END
                ))
        END
    FROM repayments r
    WHERE r.loan_id = l.loan_id
    AND r.is_reversed = FALSE
    GROUP BY l.loan_id, l.disbursement_date, l.current_dpd, l.loan_amount, l.total_principal_paid, l.days_since_last_repayment, l.loan_age
)
WHERE l.loan_id IS NOT NULL;

-- ============================================================================
-- STEP 3: Update Trigger to Calculate These Fields Automatically
-- ============================================================================
-- Modify the existing trigger function to calculate timeliness_score and
-- repayment_health whenever a repayment is added/updated

-- Note: This will be added to the update_loan_computed_fields() function
-- For now, we'll create a separate trigger that runs after the main one

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
        v_timeliness_score := NULL;
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
        v_repayment_health := NULL;
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

-- Create trigger to run after repayment changes
DROP TRIGGER IF EXISTS trg_update_loan_behavior_scores ON repayments;
CREATE TRIGGER trg_update_loan_behavior_scores
AFTER INSERT OR UPDATE ON repayments
FOR EACH ROW
EXECUTE FUNCTION update_loan_behavior_scores();

-- ============================================================================
-- STEP 4: Create Indexes for Performance
-- ============================================================================

-- Indexes already exist from migration 002, but let's ensure they're there
CREATE INDEX IF NOT EXISTS idx_loans_timeliness_score ON loans(timeliness_score) WHERE timeliness_score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_loans_repayment_health ON loans(repayment_health) WHERE repayment_health IS NOT NULL;

-- ============================================================================
-- STEP 5: Verification Queries
-- ============================================================================

-- Check population status
DO $$
DECLARE
    v_total_loans INTEGER;
    v_with_timeliness INTEGER;
    v_with_health INTEGER;
    v_avg_timeliness DECIMAL(5, 2);
    v_avg_health DECIMAL(5, 2);
BEGIN
    SELECT 
        COUNT(*),
        COUNT(timeliness_score),
        COUNT(repayment_health),
        ROUND(AVG(timeliness_score), 2),
        ROUND(AVG(repayment_health), 2)
    INTO 
        v_total_loans,
        v_with_timeliness,
        v_with_health,
        v_avg_timeliness,
        v_avg_health
    FROM loans;

    RAISE NOTICE '============================================';
    RAISE NOTICE 'Timeliness Score and Repayment Health Update';
    RAISE NOTICE '============================================';
    RAISE NOTICE 'Total Loans: %', v_total_loans;
    RAISE NOTICE 'Loans with Timeliness Score: % (%.2f%%)', v_with_timeliness, (v_with_timeliness::DECIMAL / v_total_loans * 100);
    RAISE NOTICE 'Loans with Repayment Health: % (%.2f%%)', v_with_health, (v_with_health::DECIMAL / v_total_loans * 100);
    RAISE NOTICE 'Average Timeliness Score: %', v_avg_timeliness;
    RAISE NOTICE 'Average Repayment Health: %', v_avg_health;
    RAISE NOTICE '============================================';
END $$;

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

