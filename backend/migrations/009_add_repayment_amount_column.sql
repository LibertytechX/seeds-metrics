-- Migration: Add repayment_amount column to loans table
-- This migration adds a new field to store the total expected repayment amount
-- as calculated by the source loan application system (not auto-calculated)
--
-- Purpose:
-- - Store the original expected repayment amount from the source system
-- - This is different from actual_outstanding and total_outstanding
-- - This is a static value populated via ETL, not calculated by triggers
--
-- Field Details:
-- - Type: DECIMAL(15, 2) for monetary precision
-- - Nullable: YES (existing loans may not have this value)
-- - Default: NULL (no default value, must be provided by ETL)

-- Step 1: Add the column to the loans table
ALTER TABLE loans ADD COLUMN IF NOT EXISTS repayment_amount DECIMAL(15, 2);

-- Step 2: Add comment to document the field
COMMENT ON COLUMN loans.repayment_amount IS 'Total expected repayment amount from source loan application system (principal + interest + fees). This is a static value from ETL, not auto-calculated.';

-- Step 3: Create index for performance (optional, but recommended for filtering/sorting)
CREATE INDEX IF NOT EXISTS idx_loans_repayment_amount ON loans(repayment_amount);

-- Note: No backfill needed - this field will be populated by future ETL imports
-- Existing loans will have NULL values until updated via ETL

