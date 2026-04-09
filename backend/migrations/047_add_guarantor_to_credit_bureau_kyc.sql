-- Migration 047: Add guarantor fields to loan_credit_bureau_kyc
-- Guarantor details from loans_loanguarantor linked via loans_borrowerinfo

BEGIN;

ALTER TABLE loan_credit_bureau_kyc ADD COLUMN IF NOT EXISTS guarantor_full_name VARCHAR;
ALTER TABLE loan_credit_bureau_kyc ADD COLUMN IF NOT EXISTS guarantor_phone VARCHAR;
ALTER TABLE loan_credit_bureau_kyc ADD COLUMN IF NOT EXISTS guarantor_email VARCHAR;
ALTER TABLE loan_credit_bureau_kyc ADD COLUMN IF NOT EXISTS guarantor_address TEXT;
ALTER TABLE loan_credit_bureau_kyc ADD COLUMN IF NOT EXISTS guarantor_relationship VARCHAR;
ALTER TABLE loan_credit_bureau_kyc ADD COLUMN IF NOT EXISTS guarantor_id_card_image TEXT;
ALTER TABLE loan_credit_bureau_kyc ADD COLUMN IF NOT EXISTS guarantor_selfie_image TEXT;

COMMIT;
