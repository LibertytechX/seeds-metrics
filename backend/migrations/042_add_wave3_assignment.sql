-- Migration: Add Wave 3 assignment rules for loans
-- Description:
--   - Introduce Wave 3 for loans where officer hire_date >= 2026-01-01
--     or loan disbursement_date >= 2026-01-01
--   - Update the wave check constraint to allow 'Wave 3'
--   - Recalculate wave for all existing loans using the new rules
--
-- Wave rules after this migration:
--   Wave 3: officer hire_date >= 2026-01-01 OR disbursement_date >= 2026-01-01
--   Wave 2: officer hire_date >= 2025-10-01 OR disbursement_date >= 2025-10-20
--   Wave 1: all other loans

BEGIN;

-- ============================================================================
-- STEP 1: Update wave check constraint to include 'Wave 3'
-- ============================================================================

ALTER TABLE loans DROP CONSTRAINT IF EXISTS chk_loans_wave;

ALTER TABLE loans
    ADD CONSTRAINT chk_loans_wave
    CHECK (wave IN ('Wave 1', 'Wave 2', 'Wave 3'));

-- ============================================================================
-- STEP 2: Update assign_loan_wave() function to include Wave 3 logic
-- ============================================================================

CREATE OR REPLACE FUNCTION assign_loan_wave()
RETURNS TRIGGER AS $$
DECLARE
    officer_hire_date DATE;
BEGIN
    -- Get the officer's hire_date (may be NULL if not available)
    SELECT hire_date INTO officer_hire_date
    FROM officers
    WHERE officer_id = NEW.officer_id;

    -- Wave 3: officer hired on/after 2026-01-01 OR loan disbursed on/after 2026-01-01
    IF (officer_hire_date >= '2026-01-01'::DATE)
       OR (NEW.disbursement_date >= '2026-01-01'::DATE) THEN
        NEW.wave := 'Wave 3';

    -- Wave 2: officer hired on/after 2025-10-01 OR loan disbursed on/after 2025-10-20
    ELSIF (officer_hire_date >= '2025-10-01'::DATE)
       OR (NEW.disbursement_date >= '2025-10-20'::DATE) THEN
        NEW.wave := 'Wave 2';

    -- Wave 1: all other loans
    ELSE
        NEW.wave := 'Wave 1';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION assign_loan_wave() IS
'Automatically assigns loan wave based on officer hire_date and loan disbursement_date. Wave 3: hire_date/disbursement >= 2026-01-01; Wave 2: hire_date >= 2025-10-01 or disbursement >= 2025-10-20; else Wave 1.';

-- ============================================================================
-- STEP 3: Recalculate wave for all existing loans using the new rules
-- ============================================================================

UPDATE loans l
SET wave = CASE
    WHEN (
        SELECT o.hire_date >= '2026-01-01'::DATE
        FROM officers o
        WHERE o.officer_id = l.officer_id
    ) OR l.disbursement_date >= '2026-01-01'::DATE
    THEN 'Wave 3'
    WHEN (
        SELECT o.hire_date >= '2025-10-01'::DATE
        FROM officers o
        WHERE o.officer_id = l.officer_id
    ) OR l.disbursement_date >= '2025-10-20'::DATE
    THEN 'Wave 2'
    ELSE 'Wave 1'
END;

COMMIT;

