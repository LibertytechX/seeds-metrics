-- ============================================================================
-- Migration 038: Add django_status column to loans table
-- ============================================================================
-- Description: Adds a django_status field to the loans table to store the
--              raw, unmodified status value from the Django loans_ajoloan
--              table. This is in addition to the existing normalized status
--              field used throughout the metrics dashboards.
--
-- New Column:
--   - django_status: Raw Django loan status (e.g., OPEN, APPROVED,
--                    DEFAULTED, PAST_MATURITY, etc.). Nullable.
--
-- Author: Seeds Metrics Team
-- Date: 2025-11-26
-- ============================================================================

BEGIN;

-- Add django_status column to loans table (nullable)
ALTER TABLE loans
ADD COLUMN django_status VARCHAR(100);

-- Add comment to document the column
COMMENT ON COLUMN loans.django_status IS
'Raw status value synced directly from the Django loans_ajoloan.status field. Nullable and kept alongside the normalized status field used by analytics.';

-- Optional index for future filtering on raw Django status
CREATE INDEX idx_loans_django_status ON loans(django_status);

COMMIT;

