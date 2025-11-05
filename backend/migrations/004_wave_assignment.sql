-- Migration: Wave Assignment System for Loans
-- Description: Implements automatic wave assignment based on officer hire_date and loan disbursement_date
-- Date: 2025-11-05
-- 
-- Wave Assignment Rules:
-- Wave 2: Officer hire_date >= 2025-10-01 OR loan disbursement_date >= 2025-10-20
-- Wave 1: All other loans

-- ============================================================================
-- STEP 1: Create trigger function for automatic wave assignment
-- ============================================================================

CREATE OR REPLACE FUNCTION assign_loan_wave()
RETURNS TRIGGER AS $$
DECLARE
    officer_hire_date DATE;
BEGIN
    -- Get the officer's hire_date
    SELECT hire_date INTO officer_hire_date
    FROM officers
    WHERE officer_id = NEW.officer_id;
    
    -- Assign wave based on the rules:
    -- Wave 2 if: officer hired on/after 2025-10-01 OR loan disbursed on/after 2025-10-20
    -- Wave 1 otherwise
    IF (officer_hire_date >= '2025-10-01'::DATE) OR (NEW.disbursement_date >= '2025-10-20'::DATE) THEN
        NEW.wave := 'Wave 2';
    ELSE
        NEW.wave := 'Wave 1';
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- STEP 2: Create trigger for INSERT operations
-- ============================================================================

DROP TRIGGER IF EXISTS trigger_assign_wave_on_insert ON loans;

CREATE TRIGGER trigger_assign_wave_on_insert
    BEFORE INSERT ON loans
    FOR EACH ROW
    EXECUTE FUNCTION assign_loan_wave();

-- ============================================================================
-- STEP 3: Create trigger for UPDATE operations (when disbursement_date changes)
-- ============================================================================

DROP TRIGGER IF EXISTS trigger_assign_wave_on_update ON loans;

CREATE TRIGGER trigger_assign_wave_on_update
    BEFORE UPDATE OF disbursement_date ON loans
    FOR EACH ROW
    WHEN (OLD.disbursement_date IS DISTINCT FROM NEW.disbursement_date)
    EXECUTE FUNCTION assign_loan_wave();

-- ============================================================================
-- STEP 4: Backfill wave for all existing loans
-- ============================================================================

-- Update all loans to assign correct wave based on current rules
UPDATE loans l
SET wave = CASE
    WHEN (
        SELECT o.hire_date >= '2025-10-01'::DATE
        FROM officers o
        WHERE o.officer_id = l.officer_id
    ) OR l.disbursement_date >= '2025-10-20'::DATE
    THEN 'Wave 2'
    ELSE 'Wave 1'
END;

-- ============================================================================
-- STEP 5: Create index on wave field if not exists (already exists)
-- ============================================================================

-- Index already exists: idx_loans_wave

-- ============================================================================
-- STEP 6: Add comments for documentation
-- ============================================================================

COMMENT ON COLUMN loans.wave IS 'Loan wave assignment: Wave 2 if officer hired >= 2025-10-01 OR disbursement >= 2025-10-20, else Wave 1';
COMMENT ON FUNCTION assign_loan_wave() IS 'Automatically assigns wave to loans based on officer hire_date and loan disbursement_date';
COMMENT ON TRIGGER trigger_assign_wave_on_insert ON loans IS 'Assigns wave on loan creation';
COMMENT ON TRIGGER trigger_assign_wave_on_update ON loans IS 'Reassigns wave when disbursement_date is updated';

-- ============================================================================
-- VERIFICATION QUERIES (for manual testing after migration)
-- ============================================================================

-- Count loans by wave
-- SELECT wave, COUNT(*) as loan_count, 
--        SUM(total_outstanding) as total_outstanding,
--        MIN(disbursement_date) as earliest_disbursement,
--        MAX(disbursement_date) as latest_disbursement
-- FROM loans
-- GROUP BY wave
-- ORDER BY wave;

-- Sample loans from each wave
-- SELECT l.loan_id, l.officer_id, o.hire_date as officer_hire_date, 
--        l.disbursement_date, l.wave, l.total_outstanding
-- FROM loans l
-- JOIN officers o ON l.officer_id = o.officer_id
-- WHERE l.wave = 'Wave 1'
-- ORDER BY l.disbursement_date DESC
-- LIMIT 5;

-- SELECT l.loan_id, l.officer_id, o.hire_date as officer_hire_date, 
--        l.disbursement_date, l.wave, l.total_outstanding
-- FROM loans l
-- JOIN officers o ON l.officer_id = o.officer_id
-- WHERE l.wave = 'Wave 2'
-- ORDER BY l.disbursement_date DESC
-- LIMIT 5;

-- Verify trigger is active
-- SELECT trigger_name, event_manipulation, event_object_table, action_statement
-- FROM information_schema.triggers
-- WHERE event_object_table = 'loans' AND trigger_name LIKE '%wave%';

