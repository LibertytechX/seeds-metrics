-- ============================================================================
-- SQL Migration Scripts for Metrics Dashboard Backend
-- ============================================================================
-- Database: PostgreSQL 14+
-- Purpose: Create all tables, indexes, and constraints
-- ============================================================================

-- ============================================================================
-- STEP 1: Create Core Tables
-- ============================================================================

-- Table: customers
CREATE TABLE customers (
    customer_id VARCHAR(50) PRIMARY KEY,
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20),
    customer_email VARCHAR(255),
    date_of_birth DATE,
    gender VARCHAR(10),
    state VARCHAR(100),
    lga VARCHAR(100),
    address TEXT,
    kyc_status VARCHAR(50),
    kyc_verified_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_customers_phone ON customers(customer_phone);
CREATE INDEX idx_customers_state ON customers(state);

-- Table: officers
CREATE TABLE officers (
    officer_id VARCHAR(50) PRIMARY KEY,
    officer_name VARCHAR(255) NOT NULL,
    officer_phone VARCHAR(20),
    officer_email VARCHAR(255),
    region VARCHAR(100) NOT NULL,
    branch VARCHAR(100) NOT NULL,
    employment_status VARCHAR(50) DEFAULT 'Active',
    hire_date DATE,
    termination_date DATE,
    primary_channel VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_officers_branch ON officers(branch);
CREATE INDEX idx_officers_region ON officers(region);
CREATE INDEX idx_officers_status ON officers(employment_status);

-- Table: loans
-- CRITICAL: Fields marked [COMPUTED] are calculated from repayments, NOT from ETL source
-- This reduces compute load on the main business server
CREATE TABLE loans (
    loan_id VARCHAR(50) PRIMARY KEY,

    -- FROM ETL SOURCE (Main Backend)
    customer_id VARCHAR(50) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20),
    officer_id VARCHAR(50) NOT NULL,
    officer_name VARCHAR(255) NOT NULL,
    officer_phone VARCHAR(20),
    region VARCHAR(100) NOT NULL,
    branch VARCHAR(100) NOT NULL,
    state VARCHAR(100),
    loan_amount DECIMAL(15, 2) NOT NULL,
    disbursement_date DATE NOT NULL,
    maturity_date DATE NOT NULL,
    loan_term_days INTEGER NOT NULL,
    interest_rate DECIMAL(5, 4),
    fee_amount DECIMAL(15, 2),
    channel VARCHAR(50) NOT NULL,
    channel_partner VARCHAR(100),
    status VARCHAR(50) NOT NULL,
    closed_date DATE,

    -- [COMPUTED] FROM REPAYMENTS - Calculated by analytics service
    current_dpd INTEGER DEFAULT 0,
    max_dpd_ever INTEGER DEFAULT 0,
    first_payment_missed BOOLEAN DEFAULT FALSE,
    first_payment_due_date DATE,
    first_payment_received_date DATE,
    principal_outstanding DECIMAL(15, 2) DEFAULT 0,
    interest_outstanding DECIMAL(15, 2) DEFAULT 0,
    fees_outstanding DECIMAL(15, 2) DEFAULT 0,
    total_outstanding DECIMAL(15, 2) DEFAULT 0,
    total_principal_paid DECIMAL(15, 2) DEFAULT 0,
    total_interest_paid DECIMAL(15, 2) DEFAULT 0,
    total_fees_paid DECIMAL(15, 2) DEFAULT 0,
    fimr_tagged BOOLEAN DEFAULT FALSE,
    early_indicator_tagged BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_officer FOREIGN KEY (officer_id) REFERENCES officers(officer_id),
    CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(customer_id)
);

CREATE INDEX idx_loans_officer ON loans(officer_id);
CREATE INDEX idx_loans_branch ON loans(branch);
CREATE INDEX idx_loans_region ON loans(region);
CREATE INDEX idx_loans_disbursement_date ON loans(disbursement_date);
CREATE INDEX idx_loans_status ON loans(status);
CREATE INDEX idx_loans_current_dpd ON loans(current_dpd);
CREATE INDEX idx_loans_fimr_tagged ON loans(fimr_tagged) WHERE fimr_tagged = TRUE;
CREATE INDEX idx_loans_early_indicator ON loans(early_indicator_tagged) WHERE early_indicator_tagged = TRUE;
CREATE INDEX idx_loans_officer_status ON loans(officer_id, status);
CREATE INDEX idx_loans_disbursement_officer ON loans(disbursement_date, officer_id);

-- Table: repayments
CREATE TABLE repayments (
    repayment_id VARCHAR(50) PRIMARY KEY,
    loan_id VARCHAR(50) NOT NULL,
    payment_date DATE NOT NULL,
    payment_amount DECIMAL(15, 2) NOT NULL,
    principal_paid DECIMAL(15, 2) DEFAULT 0,
    interest_paid DECIMAL(15, 2) DEFAULT 0,
    fees_paid DECIMAL(15, 2) DEFAULT 0,
    penalty_paid DECIMAL(15, 2) DEFAULT 0,
    payment_method VARCHAR(50),
    payment_reference VARCHAR(100),
    payment_channel VARCHAR(50),
    dpd_at_payment INTEGER DEFAULT 0,
    is_backdated BOOLEAN DEFAULT FALSE,
    is_reversed BOOLEAN DEFAULT FALSE,
    reversal_date DATE,
    reversal_reason TEXT,
    waiver_amount DECIMAL(15, 2) DEFAULT 0,
    waiver_type VARCHAR(50),
    waiver_approved_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_loan FOREIGN KEY (loan_id) REFERENCES loans(loan_id)
);

CREATE INDEX idx_repayments_loan ON repayments(loan_id);
CREATE INDEX idx_repayments_payment_date ON repayments(payment_date);
CREATE INDEX idx_repayments_loan_date ON repayments(loan_id, payment_date);
CREATE INDEX idx_repayments_backdated ON repayments(is_backdated) WHERE is_backdated = TRUE;
CREATE INDEX idx_repayments_reversed ON repayments(is_reversed) WHERE is_reversed = TRUE;

-- ============================================================================
-- STEP 2: Create Derived/Aggregated Tables
-- ============================================================================

-- Table: officer_metrics_daily
CREATE TABLE officer_metrics_daily (
    metric_id SERIAL PRIMARY KEY,
    officer_id VARCHAR(50) NOT NULL,
    calculation_date DATE NOT NULL,
    total_portfolio DECIMAL(15, 2) DEFAULT 0,
    active_loans INTEGER DEFAULT 0,
    total_disbursed_count INTEGER DEFAULT 0,
    total_disbursed_amount DECIMAL(15, 2) DEFAULT 0,
    overdue_15d_amount DECIMAL(15, 2) DEFAULT 0,
    overdue_15d_count INTEGER DEFAULT 0,
    par15_ratio DECIMAL(5, 4) DEFAULT 0,
    first_miss_count INTEGER DEFAULT 0,
    disbursed_count INTEGER DEFAULT 0,
    fimr DECIMAL(5, 4) DEFAULT 0,
    dpd_1to6_balance DECIMAL(15, 2) DEFAULT 0,
    amount_due_7d DECIMAL(15, 2) DEFAULT 0,
    d06_slippage DECIMAL(5, 4) DEFAULT 0,
    moved_to_7to30 DECIMAL(15, 2) DEFAULT 0,
    prev_dpd_1to6_balance DECIMAL(15, 2) DEFAULT 0,
    roll DECIMAL(5, 4) DEFAULT 0,
    fees_collected DECIMAL(15, 2) DEFAULT 0,
    fees_due DECIMAL(15, 2) DEFAULT 0,
    frr DECIMAL(5, 4) DEFAULT 0,
    interest_collected DECIMAL(15, 2) DEFAULT 0,
    par15_mid_month DECIMAL(15, 2) DEFAULT 0,
    ayr DECIMAL(5, 4) DEFAULT 0,
    risk_score_norm DECIMAL(5, 4) DEFAULT 0,
    on_time_rate DECIMAL(5, 4) DEFAULT 0,
    channel_purity DECIMAL(5, 4) DEFAULT 0,
    dqi INTEGER DEFAULT 0,
    porr DECIMAL(5, 4) DEFAULT 0,
    waivers_amount DECIMAL(15, 2) DEFAULT 0,
    backdated_count INTEGER DEFAULT 0,
    total_entries INTEGER DEFAULT 0,
    reversals_count INTEGER DEFAULT 0,
    had_float_gap BOOLEAN DEFAULT FALSE,
    risk_score INTEGER DEFAULT 0,
    risk_band VARCHAR(20),
    total_yield DECIMAL(15, 2) DEFAULT 0,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_officer_metrics FOREIGN KEY (officer_id) REFERENCES officers(officer_id),
    CONSTRAINT unique_officer_date UNIQUE (officer_id, calculation_date)
);

CREATE INDEX idx_officer_metrics_officer ON officer_metrics_daily(officer_id);
CREATE INDEX idx_officer_metrics_date ON officer_metrics_daily(calculation_date);
CREATE INDEX idx_officer_metrics_officer_date ON officer_metrics_daily(officer_id, calculation_date);
CREATE INDEX idx_officer_metrics_risk_band ON officer_metrics_daily(risk_band);

-- Table: branch_metrics_daily
CREATE TABLE branch_metrics_daily (
    metric_id SERIAL PRIMARY KEY,
    branch VARCHAR(100) NOT NULL,
    region VARCHAR(100) NOT NULL,
    calculation_date DATE NOT NULL,
    portfolio_total DECIMAL(15, 2) DEFAULT 0,
    active_loans INTEGER DEFAULT 0,
    total_officers INTEGER DEFAULT 0,
    overdue_15d DECIMAL(15, 2) DEFAULT 0,
    par15_ratio DECIMAL(5, 4) DEFAULT 0,
    ayr DECIMAL(5, 4) DEFAULT 0,
    dqi INTEGER DEFAULT 0,
    fimr DECIMAL(5, 4) DEFAULT 0,
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_branch_date UNIQUE (branch, calculation_date)
);

CREATE INDEX idx_branch_metrics_branch ON branch_metrics_daily(branch);
CREATE INDEX idx_branch_metrics_region ON branch_metrics_daily(region);
CREATE INDEX idx_branch_metrics_date ON branch_metrics_daily(calculation_date);

-- ============================================================================
-- STEP 3: Create Supporting Tables
-- ============================================================================

-- Table: loan_schedule
CREATE TABLE loan_schedule (
    schedule_id SERIAL PRIMARY KEY,
    loan_id VARCHAR(50) NOT NULL,
    installment_number INTEGER NOT NULL,
    due_date DATE NOT NULL,
    principal_due DECIMAL(15, 2) NOT NULL,
    interest_due DECIMAL(15, 2) NOT NULL,
    fee_due DECIMAL(15, 2) DEFAULT 0,
    total_due DECIMAL(15, 2) NOT NULL,
    payment_status VARCHAR(50) DEFAULT 'Pending',
    amount_paid DECIMAL(15, 2) DEFAULT 0,
    payment_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_schedule_loan FOREIGN KEY (loan_id) REFERENCES loans(loan_id),
    CONSTRAINT unique_loan_installment UNIQUE (loan_id, installment_number)
);

CREATE INDEX idx_schedule_loan ON loan_schedule(loan_id);
CREATE INDEX idx_schedule_due_date ON loan_schedule(due_date);
CREATE INDEX idx_schedule_status ON loan_schedule(payment_status);
CREATE INDEX idx_schedule_loan_status ON loan_schedule(loan_id, payment_status);

-- Table: dpd_transitions
CREATE TABLE dpd_transitions (
    transition_id SERIAL PRIMARY KEY,
    loan_id VARCHAR(50) NOT NULL,
    officer_id VARCHAR(50) NOT NULL,
    transition_date DATE NOT NULL,
    from_dpd_bucket VARCHAR(20),
    to_dpd_bucket VARCHAR(20),
    outstanding_balance DECIMAL(15, 2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_transition_loan FOREIGN KEY (loan_id) REFERENCES loans(loan_id),
    CONSTRAINT fk_transition_officer FOREIGN KEY (officer_id) REFERENCES officers(officer_id)
);

CREATE INDEX idx_transitions_loan ON dpd_transitions(loan_id);
CREATE INDEX idx_transitions_officer ON dpd_transitions(officer_id);
CREATE INDEX idx_transitions_date ON dpd_transitions(transition_date);
CREATE INDEX idx_transitions_buckets ON dpd_transitions(from_dpd_bucket, to_dpd_bucket);

-- Table: par15_snapshots
CREATE TABLE par15_snapshots (
    snapshot_id SERIAL PRIMARY KEY,
    snapshot_date DATE NOT NULL,
    officer_id VARCHAR(50) NOT NULL,
    par15_exposure DECIMAL(15, 2) NOT NULL,
    par15_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_snapshot_officer FOREIGN KEY (officer_id) REFERENCES officers(officer_id),
    CONSTRAINT unique_officer_snapshot UNIQUE (officer_id, snapshot_date)
);

CREATE INDEX idx_snapshots_officer ON par15_snapshots(officer_id);
CREATE INDEX idx_snapshots_date ON par15_snapshots(snapshot_date);

-- Table: team_members
CREATE TABLE team_members (
    member_id VARCHAR(50) PRIMARY KEY,
    member_name VARCHAR(255) NOT NULL,
    member_email VARCHAR(255),
    member_phone VARCHAR(20),
    role VARCHAR(100),
    department VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_team_members_active ON team_members(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_team_members_role ON team_members(role);

-- Table: audit_tracking
CREATE TABLE audit_tracking (
    audit_id SERIAL PRIMARY KEY,
    officer_id VARCHAR(50) NOT NULL,
    assignee_id VARCHAR(50),
    assignee_name VARCHAR(255),
    assigned_date TIMESTAMP,
    audit_status VARCHAR(50) NOT NULL,
    status_changed_date TIMESTAMP,
    last_audit_date DATE,
    audit_notes TEXT,
    risk_level VARCHAR(20),
    action_type VARCHAR(100),
    action_date TIMESTAMP,
    action_notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_audit_officer FOREIGN KEY (officer_id) REFERENCES officers(officer_id)
);

CREATE INDEX idx_audit_officer ON audit_tracking(officer_id);
CREATE INDEX idx_audit_assignee ON audit_tracking(assignee_id);
CREATE INDEX idx_audit_status ON audit_tracking(audit_status);
CREATE INDEX idx_audit_last_date ON audit_tracking(last_audit_date);

-- Table: metric_calculation_log
CREATE TABLE metric_calculation_log (
    log_id SERIAL PRIMARY KEY,
    calculation_type VARCHAR(100) NOT NULL,
    calculation_date DATE NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_seconds INTEGER,
    records_processed INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,
    status VARCHAR(50) NOT NULL,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_calc_log_type ON metric_calculation_log(calculation_type);
CREATE INDEX idx_calc_log_date ON metric_calculation_log(calculation_date);
CREATE INDEX idx_calc_log_status ON metric_calculation_log(status);

-- ============================================================================
-- STEP 4: Create Triggers for Updated_At
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_customers_updated_at BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_officers_updated_at BEFORE UPDATE ON officers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_loans_updated_at BEFORE UPDATE ON loans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_repayments_updated_at BEFORE UPDATE ON repayments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_loan_schedule_updated_at BEFORE UPDATE ON loan_schedule
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_team_members_updated_at BEFORE UPDATE ON team_members
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_audit_tracking_updated_at BEFORE UPDATE ON audit_tracking
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- STEP 5: Create Trigger to Compute Loan Fields from Repayments
-- ============================================================================
-- This trigger automatically updates computed fields in the loans table
-- whenever a repayment is inserted or updated

CREATE OR REPLACE FUNCTION update_loan_computed_fields()
RETURNS TRIGGER AS $$
DECLARE
    v_loan_id VARCHAR(50);
    v_total_principal_paid DECIMAL(15, 2);
    v_total_interest_paid DECIMAL(15, 2);
    v_total_fees_paid DECIMAL(15, 2);
    v_first_payment_date DATE;
    v_last_payment_date DATE;
    v_first_due_date DATE;
    v_disbursement_date DATE;
    v_current_dpd INTEGER;
    v_loan_amount DECIMAL(15, 2);
    v_interest_rate DECIMAL(5, 4);
    v_loan_term_days INTEGER;
    v_fee_amount DECIMAL(15, 2);
    v_max_dpd INTEGER;
    v_repayment_count INTEGER;
    v_days_since_last_repayment INTEGER;
BEGIN
    v_loan_id := NEW.loan_id;

    -- Get loan details including disbursement date
    SELECT loan_amount, interest_rate, loan_term_days, fee_amount, max_dpd_ever, disbursement_date
    INTO v_loan_amount, v_interest_rate, v_loan_term_days, v_fee_amount, v_max_dpd, v_disbursement_date
    FROM loans
    WHERE loan_id = v_loan_id;

    -- Calculate total payments (excluding reversed payments)
    SELECT
        COALESCE(SUM(principal_paid), 0),
        COALESCE(SUM(interest_paid), 0),
        COALESCE(SUM(fees_paid), 0),
        MIN(payment_date),
        MAX(payment_date),
        COUNT(*)
    INTO
        v_total_principal_paid,
        v_total_interest_paid,
        v_total_fees_paid,
        v_first_payment_date,
        v_last_payment_date,
        v_repayment_count
    FROM repayments
    WHERE loan_id = v_loan_id
      AND is_reversed = FALSE;

    -- Calculate days since last repayment
    IF v_last_payment_date IS NOT NULL THEN
        v_days_since_last_repayment := CURRENT_DATE - v_last_payment_date;
    ELSE
        v_days_since_last_repayment := NULL;
    END IF;

    -- Get first due date from loan_schedule
    SELECT MIN(due_date) INTO v_first_due_date
    FROM loan_schedule
    WHERE loan_id = v_loan_id;

    -- If no schedule exists, calculate first due date as 30 days after disbursement
    IF v_first_due_date IS NULL THEN
        v_first_due_date := v_disbursement_date + INTERVAL '30 days';
    END IF;

    -- Calculate current DPD (days past due for oldest unpaid installment)
    SELECT
        COALESCE(MAX(CURRENT_DATE - due_date), 0)
    INTO v_current_dpd
    FROM loan_schedule
    WHERE loan_id = v_loan_id
      AND payment_status IN ('Pending', 'Partial')
      AND due_date < CURRENT_DATE;

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
        -- FIMR (First Installment Missed or Rescheduled): TRUE if first payment was late or never made
        fimr_tagged = CASE
            WHEN v_repayment_count < 10 THEN
                -- For loans with less than 10 repayments, check if first payment was late
                CASE
                    WHEN v_first_payment_date IS NULL THEN TRUE  -- No payment received yet
                    WHEN v_first_due_date IS NULL THEN FALSE     -- No due date available
                    WHEN v_first_payment_date > v_first_due_date THEN TRUE  -- Payment was late
                    ELSE FALSE  -- Payment was on time
                END
            ELSE
                -- For loans with 10+ repayments, keep existing value or set to FALSE
                COALESCE((SELECT fimr_tagged FROM loans WHERE loan_id = v_loan_id), FALSE)
        END,
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),

        -- Days since last repayment
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

-- Also create trigger for loan_schedule updates (affects DPD calculation)
CREATE TRIGGER trg_update_loan_after_schedule_change
AFTER INSERT OR UPDATE ON loan_schedule
FOR EACH ROW
EXECUTE FUNCTION update_loan_computed_fields();

-- ============================================================================
-- STEP 6: Insert Sample Team Members
-- ============================================================================

INSERT INTO team_members (member_id, member_name, member_email, role, department, is_active) VALUES
('TM001', 'Sarah Johnson', 'sarah.johnson@company.com', 'Audit Manager', 'Risk Management', TRUE),
('TM002', 'John Smith', 'john.smith@company.com', 'Senior Auditor', 'Audit', TRUE),
('TM003', 'Michael Chen', 'michael.chen@company.com', 'Risk Analyst', 'Risk Management', TRUE),
('TM004', 'Amina Bello', 'amina.bello@company.com', 'Compliance Officer', 'Compliance', TRUE),
('TM005', 'David Okafor', 'david.okafor@company.com', 'Portfolio Manager', 'Portfolio Management', TRUE);

-- ============================================================================
-- VERIFICATION QUERIES
-- ============================================================================

-- Check all tables created
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
ORDER BY table_name;

-- Check all indexes
SELECT tablename, indexname
FROM pg_indexes
WHERE schemaname = 'public'
ORDER BY tablename, indexname;

-- Check all foreign keys
SELECT
    tc.table_name,
    kcu.column_name,
    ccu.table_name AS foreign_table_name,
    ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints AS tc
JOIN information_schema.key_column_usage AS kcu
  ON tc.constraint_name = kcu.constraint_name
JOIN information_schema.constraint_column_usage AS ccu
  ON ccu.constraint_name = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
ORDER BY tc.table_name;

