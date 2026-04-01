-- Migration 045: Create loan_credit_bureau_kyc table
-- Denormalized table combining loan info, borrower KYC, and credit bureau data
-- from three priority sources (credit_bureau_result > borrower_worthiness > credit_bureau_metadata)

BEGIN;

CREATE TABLE IF NOT EXISTS loan_credit_bureau_kyc (
    id                              BIGSERIAL PRIMARY KEY,

    -- === LOAN INFO (from loans_ajoloan) ===
    django_loan_id                  BIGINT UNIQUE NOT NULL,
    loan_ref                        VARCHAR NOT NULL,
    loan_amount                     DOUBLE PRECISION,
    tenor                           VARCHAR,
    tenor_in_days                   INTEGER,
    borrower_full_name              VARCHAR,
    borrower_phone_number           VARCHAR,
    loan_status                     VARCHAR,
    date_disbursed                  TIMESTAMPTZ,

    -- === BORROWER KYC (from loans_borrowerinfo + accounts_customuser) ===
    verification_number             VARCHAR,
    verification_type               VARCHAR,
    nin                             VARCHAR,
    date_of_birth                   VARCHAR,
    is_verified                     BOOLEAN,
    address                         VARCHAR,

    -- === FACE MATCH & IMAGES (from loans_borrowerinfo) ===
    face_match                      BOOLEAN,
    id_card_image                   TEXT,
    selfie_image                    TEXT,

    -- === CREDIT BUREAU (Priority 1: credit_bureau_creditbureauresult) ===
    cb_result                       JSONB,
    cb_status                       BOOLEAN,
    cb_reason                       TEXT,
    cb_decision                     JSONB,
    cb_decision_status              VARCHAR,
    cb_credibility                  JSONB,
    cb_bad_loans_institutions       JSONB,
    cb_bad_loans_institutions_count INTEGER,
    cb_count_of_open_loans          INTEGER,
    cb_total_outstanding            DOUBLE PRECISION,
    cb_debt_threshold               DOUBLE PRECISION,
    cb_high_outstanding_debt        BOOLEAN,
    cb_open_loan_institutions       JSONB,
    cb_max_debt_institution_count   INTEGER,

    -- === CREDIT BUREAU LEGACY (Priority 3: loans_creditbureaumetadata) ===
    cb_legacy_response              JSONB,
    cb_no_of_defaulted_loans        INTEGER,
    cb_monthly_repayment_amount     DOUBLE PRECISION,

    -- === DATA SOURCE ===
    cb_data_source                  VARCHAR NOT NULL DEFAULT 'none',

    -- === TIMESTAMPS ===
    cb_created_at                   TIMESTAMPTZ,
    synced_at                       TIMESTAMPTZ DEFAULT NOW(),
    created_at                      TIMESTAMPTZ DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_loan_cb_kyc_loan_ref ON loan_credit_bureau_kyc(loan_ref);
CREATE INDEX IF NOT EXISTS idx_loan_cb_kyc_data_source ON loan_credit_bureau_kyc(cb_data_source);
CREATE INDEX IF NOT EXISTS idx_loan_cb_kyc_django_loan_id ON loan_credit_bureau_kyc(django_loan_id);
CREATE INDEX IF NOT EXISTS idx_loan_cb_kyc_date_disbursed ON loan_credit_bureau_kyc(date_disbursed DESC NULLS LAST);

COMMIT;

