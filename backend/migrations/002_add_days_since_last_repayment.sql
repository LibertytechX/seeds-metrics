-- Migration: Add days_since_last_repayment column to loans table
-- This column is computed by the update_loan_computed_fields() trigger

ALTER TABLE loans ADD COLUMN IF NOT EXISTS days_since_last_repayment INTEGER DEFAULT 0;

