-- Migration: Add wave column to loans table
-- Description: Adds a wave categorization system to track loan cohorts/waves
-- Author: System
-- Date: 2025-11-01

-- Add wave column to loans table
ALTER TABLE loans ADD COLUMN IF NOT EXISTS wave VARCHAR(10) DEFAULT 'Wave 2';

-- Add comment to describe the column
COMMENT ON COLUMN loans.wave IS 'Loan wave/cohort identifier (Wave 1, Wave 2, etc.)';

-- Add index for performance on wave filtering
CREATE INDEX IF NOT EXISTS idx_loans_wave ON loans(wave);

-- Add check constraint to ensure only valid wave values
ALTER TABLE loans ADD CONSTRAINT chk_loans_wave CHECK (wave IN ('Wave 1', 'Wave 2'));

-- Update existing test loans with wave values for testing
-- Mix of Wave 1 and Wave 2 loans

-- Set some existing loans to Wave 1 (based on disbursement date - earlier loans are Wave 1)
UPDATE loans
SET wave = 'Wave 1'
WHERE disbursement_date < '2025-03-01';

-- Set remaining loans to Wave 2 (later loans are Wave 2)
UPDATE loans
SET wave = 'Wave 2'
WHERE disbursement_date >= '2025-03-01';

-- Ensure all loans have a wave value (default to Wave 2 for any NULL)
UPDATE loans
SET wave = 'Wave 2'
WHERE wave IS NULL;

-- Verify the wave distribution
-- This is a comment for manual verification after running the migration
-- SELECT wave, COUNT(*) as loan_count FROM loans GROUP BY wave ORDER BY wave;

