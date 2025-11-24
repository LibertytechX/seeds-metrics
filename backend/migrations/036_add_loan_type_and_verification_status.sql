-- ============================================================================
-- Migration 036: Add loan_type and verification_status Columns
-- ============================================================================
-- Description: Adds loan_type and verification_status columns to the loans
--              table to support additional loan classification and filtering.
--
-- New Columns:
-- - loan_type: Type of loan (AJO, BNPL, PROSPER, DMO, etc.) from Django
-- - verification_status: Verification status of the loan from Django
--
-- Author: Seeds Metrics Team
-- Date: 2025-11-24
-- ============================================================================

BEGIN;

-- Add loan_type column to loans table
ALTER TABLE loans
ADD COLUMN loan_type VARCHAR(100);

-- Add verification_status column to loans table
ALTER TABLE loans
ADD COLUMN verification_status VARCHAR(100);

-- Add comments to document the columns
COMMENT ON COLUMN loans.loan_type IS
'Type of loan synced from Django database. Possible values: AJO, BNPL, PROSPER, DMO, etc. Used for loan classification and filtering.';

COMMENT ON COLUMN loans.verification_status IS
'Verification status of the loan synced from Django database. Used for loan filtering and risk assessment.';

-- Create indexes for better query performance when filtering
CREATE INDEX idx_loans_loan_type ON loans(loan_type);
CREATE INDEX idx_loans_verification_status ON loans(verification_status);

COMMIT;

