-- Migration 046: Add loan_type column to loan_credit_bureau_kyc
-- Used to filter out BNPL, RNPL, and MERCHANT_OVERDRAFT loans

BEGIN;

ALTER TABLE loan_credit_bureau_kyc ADD COLUMN IF NOT EXISTS loan_type VARCHAR;

CREATE INDEX IF NOT EXISTS idx_loan_cb_kyc_loan_type ON loan_credit_bureau_kyc(loan_type);

COMMIT;

