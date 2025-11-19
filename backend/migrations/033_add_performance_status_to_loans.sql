-- ============================================================================
-- Migration 033: Add performance_status column to loans table
-- ============================================================================
-- Description: Adds the performance_status field to the loans table to sync
--              from the Django database. This field provides a better
--              representation of each loan's current state compared to the
--              existing status field.
--
-- Possible values:
--   - PERFORMING: Loan is being repaid on time
--   - DEFAULTED: Loan has defaulted
--   - LOST: Loan is considered lost (write-off)
--   - PAST_MATURITY: Loan is past its maturity date
--   - OWED_BALANCE: Loan has an outstanding balance
--   - NULL: Status not yet determined or not applicable
--
-- Author: Seeds Metrics Team
-- Date: 2025-11-19
-- ============================================================================

BEGIN;

-- Add performance_status column to loans table
ALTER TABLE loans
ADD COLUMN performance_status VARCHAR(100);

-- Add comment to document the column
COMMENT ON COLUMN loans.performance_status IS 
'Performance status of the loan synced from Django database. Possible values: PERFORMING, DEFAULTED, LOST, PAST_MATURITY, OWED_BALANCE. Provides a more detailed status than the generic status field.';

-- Create index for better query performance when filtering by performance_status
CREATE INDEX idx_loans_performance_status ON loans(performance_status);

COMMIT;

