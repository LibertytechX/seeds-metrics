-- ============================================================================
-- Migration: Add Repayment Behavior and Officer Performance Fields
-- ============================================================================
-- Purpose: Add new fields to track loan repayment behavior and officer metrics
-- Date: 2025-10-30
-- ============================================================================

-- ============================================================================
-- PART 1: Add New Fields to loans Table
-- ============================================================================

-- Add timeliness_score (0-100 scale, from ETL source)
ALTER TABLE loans ADD COLUMN IF NOT EXISTS timeliness_score DECIMAL(5, 2) DEFAULT NULL
    CHECK (timeliness_score IS NULL OR (timeliness_score >= 0 AND timeliness_score <= 100));

-- Add repayment_health (0-100 scale, from ETL source)
ALTER TABLE loans ADD COLUMN IF NOT EXISTS repayment_health DECIMAL(5, 2) DEFAULT NULL
    CHECK (repayment_health IS NULL OR (repayment_health >= 0 AND repayment_health <= 100));

-- Add days_since_last_repayment (computed internally)
ALTER TABLE loans ADD COLUMN IF NOT EXISTS days_since_last_repayment INTEGER DEFAULT NULL;

COMMENT ON COLUMN loans.timeliness_score IS 'Score (0-100) indicating payment timeliness - from main platform ETL';
COMMENT ON COLUMN loans.repayment_health IS 'Health score (0-100) for repayment pattern - from main platform ETL';
COMMENT ON COLUMN loans.days_since_last_repayment IS 'Days since most recent repayment (excluding reversed) - computed by trigger';

-- ============================================================================
-- PART 2: Add New Fields to officer_metrics_daily Table
-- ============================================================================

-- Add avg_timeliness_score
ALTER TABLE officer_metrics_daily ADD COLUMN IF NOT EXISTS avg_timeliness_score DECIMAL(5, 2) DEFAULT NULL;

-- Add avg_repayment_health
ALTER TABLE officer_metrics_daily ADD COLUMN IF NOT EXISTS avg_repayment_health DECIMAL(5, 2) DEFAULT NULL;

-- Add avg_days_since_last_repayment
ALTER TABLE officer_metrics_daily ADD COLUMN IF NOT EXISTS avg_days_since_last_repayment DECIMAL(8, 2) DEFAULT NULL;

-- Add avg_loan_age
ALTER TABLE officer_metrics_daily ADD COLUMN IF NOT EXISTS avg_loan_age DECIMAL(8, 2) DEFAULT NULL;

-- Add repayment_delay_rate (percentage)
ALTER TABLE officer_metrics_daily ADD COLUMN IF NOT EXISTS repayment_delay_rate DECIMAL(6, 2) DEFAULT NULL;

COMMENT ON COLUMN officer_metrics_daily.avg_timeliness_score IS 'Average timeliness score across active loans';
COMMENT ON COLUMN officer_metrics_daily.avg_repayment_health IS 'Average repayment health across active loans';
COMMENT ON COLUMN officer_metrics_daily.avg_days_since_last_repayment IS 'Average days since last repayment across active loans';
COMMENT ON COLUMN officer_metrics_daily.avg_loan_age IS 'Average age of active loans in days';
COMMENT ON COLUMN officer_metrics_daily.repayment_delay_rate IS 'Composite metric: (1 - ((avg_days_since_last_repayment / avg_loan_age) / 0.25)) × 100';

-- ============================================================================
-- PART 3: Update Trigger to Calculate days_since_last_repayment
-- ============================================================================

-- Drop existing trigger function and recreate with new logic
DROP FUNCTION IF EXISTS update_loan_computed_fields() CASCADE;

CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_total_principal_paid DECIMAL(15, 2);
    v_total_interest_paid DECIMAL(15, 2);
    v_total_fees_paid DECIMAL(15, 2);
    v_first_payment_date DATE;
    v_first_due_date DATE;
    v_current_dpd INTEGER;
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_max_dpd INTEGER;
    v_last_payment_date DATE;
    v_days_since_last_repayment INTEGER;
BEGIN
    v_loan_id := NEW.loan_id;

    -- Get loan details
    SELECT loan_amount, interest_rate, loan_term_days, fee_amount, max_dpd_ever
    INTO v_loan_amount, v_interest_rate, v_loan_term_days, v_fee_amount, v_max_dpd
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Calculate total payments (excluding reversed payments)
    SELECT
        COALESCE(SUM(principal_paid), 0),
        COALESCE(SUM(interest_paid), 0),
        COALESCE(SUM(fees_paid), 0),
        MIN(payment_date),
        MAX(payment_date)  -- NEW: Get last payment date
    INTO
        v_total_principal_paid,
        v_total_interest_paid,
        v_total_fees_paid,
        v_first_payment_date,
        v_last_payment_date  -- NEW: Store last payment date
    FROM repayments
    WHERE loan_id = v_loan_id
      AND is_reversed = FALSE;

    -- Get first due date from loan_schedule
    SELECT MIN(due_date) INTO v_first_due_date
    FROM loan_schedule
    WHERE loan_id = v_loan_id;

    -- Calculate current DPD (days past due for oldest unpaid installment)
    SELECT
        COALESCE(MAX(CURRENT_DATE - due_date), 0)
    INTO v_current_dpd
    FROM loan_schedule
    WHERE loan_id = v_loan_id
      AND payment_status IN ('Pending', 'Partial')
      AND due_date < CURRENT_DATE;

    -- NEW: Calculate days since last repayment
    IF v_last_payment_date IS NOT NULL THEN
        v_days_since_last_repayment := CURRENT_DATE - v_last_payment_date;
    ELSE
        v_days_since_last_repayment := NULL;
    END IF;

    -- Update loans table with computed values
    UPDATE loans
    SET
        -- Collections totals
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,

        -- Outstanding balances
        principal_outstanding = v_loan_amount - v_total_principal_paid,
        interest_outstanding = (v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid,
        fees_outstanding = v_fee_amount - v_total_fees_paid,
        total_outstanding = (v_loan_amount - v_total_principal_paid) +
                           ((v_loan_amount * v_interest_rate * v_loan_term_days / 365) - v_total_interest_paid) +
                           (v_fee_amount - v_total_fees_paid),

        -- First payment tracking
        first_payment_received_date = v_first_payment_date,
        first_payment_due_date = v_first_due_date,
        first_payment_missed = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),

        -- DPD tracking
        current_dpd = v_current_dpd,
        max_dpd_ever = GREATEST(v_max_dpd, v_current_dpd),

        -- Risk indicators
        fimr_tagged = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),

        -- NEW: Days since last repayment
        days_since_last_repayment = v_days_since_last_repayment,

        updated_at = CURRENT_TIMESTAMP
    WHERE loan_id = v_loan_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach trigger to repayments table
CREATE TRIGGER trg_update_loan_after_repayment
AFTER INSERT OR UPDATE ON repayments
FOR EACH ROW
EXECUTE FUNCTION update_loan_computed_fields();

-- ============================================================================
-- PART 4: Create Indexes for Performance
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_loans_timeliness_score ON loans(timeliness_score) WHERE timeliness_score IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_loans_repayment_health ON loans(repayment_health) WHERE repayment_health IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_loans_days_since_last_repayment ON loans(days_since_last_repayment) WHERE days_since_last_repayment IS NOT NULL;

-- ============================================================================
-- PART 5: Backfill days_since_last_repayment for Existing Loans
-- ============================================================================

-- Calculate days_since_last_repayment for all existing loans
UPDATE loans l
SET days_since_last_repayment = (
    SELECT CURRENT_DATE - MAX(r.payment_date)
    FROM repayments r
    WHERE r.loan_id = l.loan_id
      AND r.is_reversed = FALSE
)
WHERE EXISTS (
    SELECT 1
    FROM repayments r
    WHERE r.loan_id = l.loan_id
      AND r.is_reversed = FALSE
);

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

