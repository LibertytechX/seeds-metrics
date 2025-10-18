# Backend Architecture for Metrics Dashboard

## Executive Summary

This document outlines the complete backend architecture required to support the Loan Officer Metrics Dashboard in production. The design prioritizes **performance**, **scalability**, and **real-time insights** while maintaining data integrity.

**Recommended Tech Stack:**
- **Language**: **Go (Golang)** or **Node.js (TypeScript)**
- **Database**: **PostgreSQL** (primary) + **Redis** (caching)
- **Architecture**: **Microservice** (separate from main business backend)
- **API**: **REST** with **GraphQL** option for complex queries

---

## 1. Architecture Overview

### 1.1 Recommended Approach: **Separate Microservice**

**Why Separate Microservice?**
- ✅ **Performance Isolation**: Heavy analytical queries won't impact core business operations
- ✅ **Independent Scaling**: Scale analytics separately based on dashboard usage
- ✅ **Technology Flexibility**: Use optimal tech stack for analytics (different from main backend)
- ✅ **Deployment Independence**: Deploy dashboard updates without touching core systems
- ✅ **Security Boundary**: Separate read-only analytics from transactional systems

**Architecture Pattern:**
```
Main Business Backend (Transactional)
    ↓ (Event Stream / Scheduled ETL)
Analytics Database (Read-Optimized)
    ↓ (REST API)
Metrics Dashboard Frontend
```

### 1.2 Data Synchronization Strategy

**Recommended: Hybrid Approach**

1. **Batch Processing (Primary)**:
   - **Frequency**: Every 15-30 minutes
   - **Purpose**: Calculate and cache all metrics
   - **Technology**: Cron jobs or scheduled workers
   - **What**: Aggregate loan data, calculate metrics, update materialized views

2. **Real-Time Updates (Secondary)**:
   - **Frequency**: On-demand for critical events
   - **Purpose**: Update specific metrics when loans are disbursed/repaid
   - **Technology**: Event-driven (Kafka, RabbitMQ, or Redis Pub/Sub)
   - **What**: Trigger recalculation for affected officers/branches

3. **On-Demand Calculation**:
   - **Frequency**: User-triggered
   - **Purpose**: Drilldown queries with custom filters
   - **What**: Loan-level data with dynamic filtering

**Data Flow:**
```
Main Backend → Event Bus → Analytics Service → PostgreSQL → Redis Cache → API → Frontend
     ↓                                              ↓
  (Writes)                                    (Reads Only)
```

---

## 2. Database Schema Design

### 2.1 Core Tables (Source of Truth)

#### **Table: `loans`**
Primary source for all loan-level data.

**⚠️ CRITICAL DESIGN NOTE:**
Fields marked with `[COMPUTED]` are **calculated from repayments**, NOT sent from the ETL source.
This design decision **reduces compute load on the main business server** by offloading all derived calculations to the analytics service.

```sql
CREATE TABLE loans (
    -- Primary Key
    loan_id VARCHAR(50) PRIMARY KEY,

    -- ========================================
    -- SECTION 1: FROM ETL SOURCE (Main Backend)
    -- ========================================

    -- Customer Information
    customer_id VARCHAR(50) NOT NULL,
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20),

    -- Loan Officer Information
    officer_id VARCHAR(50) NOT NULL,
    officer_name VARCHAR(255) NOT NULL,
    officer_phone VARCHAR(20),

    -- Geographic Information
    region VARCHAR(100) NOT NULL,
    branch VARCHAR(100) NOT NULL,
    state VARCHAR(100),

    -- Loan Details
    loan_amount DECIMAL(15, 2) NOT NULL,
    disbursement_date DATE NOT NULL,
    maturity_date DATE NOT NULL,
    loan_term_days INTEGER NOT NULL,
    interest_rate DECIMAL(5, 4),
    fee_amount DECIMAL(15, 2),

    -- Channel Information
    channel VARCHAR(50) NOT NULL, -- 'Direct', 'Partner', etc.
    channel_partner VARCHAR(100),

    -- Loan Status
    status VARCHAR(50) NOT NULL, -- 'Active', 'Closed', 'Written Off', etc.
    closed_date DATE,

    -- ========================================
    -- SECTION 2: [COMPUTED] FROM REPAYMENTS
    -- These fields are calculated by the analytics service
    -- ========================================

    -- Delinquency Tracking [COMPUTED from repayments + loan_schedule]
    current_dpd INTEGER DEFAULT 0, -- Days Past Due
    max_dpd_ever INTEGER DEFAULT 0,
    first_payment_missed BOOLEAN DEFAULT FALSE,
    first_payment_due_date DATE,
    first_payment_received_date DATE,

    -- Outstanding Balances [COMPUTED from loan_amount - repayments]
    principal_outstanding DECIMAL(15, 2) DEFAULT 0,
    interest_outstanding DECIMAL(15, 2) DEFAULT 0,
    fees_outstanding DECIMAL(15, 2) DEFAULT 0,
    total_outstanding DECIMAL(15, 2) DEFAULT 0,

    -- Collections [COMPUTED from SUM(repayments)]
    total_principal_paid DECIMAL(15, 2) DEFAULT 0,
    total_interest_paid DECIMAL(15, 2) DEFAULT 0,
    total_fees_paid DECIMAL(15, 2) DEFAULT 0,

    -- Risk Indicators [COMPUTED from DPD and first payment status]
    fimr_tagged BOOLEAN DEFAULT FALSE, -- TRUE if first_payment_missed = TRUE
    early_indicator_tagged BOOLEAN DEFAULT FALSE, -- TRUE if current_dpd BETWEEN 1 AND 6

    -- Audit Fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Indexes
    CONSTRAINT fk_officer FOREIGN KEY (officer_id) REFERENCES officers(officer_id),
    CONSTRAINT fk_customer FOREIGN KEY (customer_id) REFERENCES customers(customer_id)
);

-- Indexes for Performance
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
```

**Computation Logic for [COMPUTED] Fields:**

The analytics service will calculate these fields using database triggers or scheduled jobs:

```sql
-- Trigger to update computed fields when repayments are inserted/updated
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
BEGIN
    v_loan_id := NEW.loan_id;

    -- Calculate total payments
    SELECT
        COALESCE(SUM(principal_paid), 0),
        COALESCE(SUM(interest_paid), 0),
        COALESCE(SUM(fees_paid), 0),
        MIN(payment_date)
    INTO
        v_total_principal_paid,
        v_total_interest_paid,
        v_total_fees_paid,
        v_first_payment_date
    FROM repayments
    WHERE loan_id = v_loan_id
      AND is_reversed = FALSE;

    -- Get first due date from loan_schedule
    SELECT MIN(due_date) INTO v_first_due_date
    FROM loan_schedule
    WHERE loan_id = v_loan_id;

    -- Calculate current DPD
    SELECT
        COALESCE(MAX(CURRENT_DATE - due_date), 0)
    INTO v_current_dpd
    FROM loan_schedule
    WHERE loan_id = v_loan_id
      AND payment_status IN ('Pending', 'Partial')
      AND due_date < CURRENT_DATE;

    -- Update loans table
    UPDATE loans
    SET
        total_principal_paid = v_total_principal_paid,
        total_interest_paid = v_total_interest_paid,
        total_fees_paid = v_total_fees_paid,
        principal_outstanding = loan_amount - v_total_principal_paid,
        interest_outstanding = (loan_amount * interest_rate * loan_term_days / 365) - v_total_interest_paid,
        fees_outstanding = fee_amount - v_total_fees_paid,
        total_outstanding = (loan_amount - v_total_principal_paid) +
                           ((loan_amount * interest_rate * loan_term_days / 365) - v_total_interest_paid) +
                           (fee_amount - v_total_fees_paid),
        first_payment_received_date = v_first_payment_date,
        first_payment_due_date = v_first_due_date,
        first_payment_missed = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),
        current_dpd = v_current_dpd,
        max_dpd_ever = GREATEST(max_dpd_ever, v_current_dpd),
        fimr_tagged = (v_first_payment_date IS NULL OR v_first_payment_date > v_first_due_date),
        early_indicator_tagged = (v_current_dpd BETWEEN 1 AND 6),
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
```

**Alternative: Scheduled Batch Update**

For better performance with high-volume repayments, use a scheduled job instead of triggers:

```sql
-- Run this every 15-30 minutes
UPDATE loans l
SET
    total_principal_paid = COALESCE(r.total_principal, 0),
    total_interest_paid = COALESCE(r.total_interest, 0),
    total_fees_paid = COALESCE(r.total_fees, 0),
    principal_outstanding = l.loan_amount - COALESCE(r.total_principal, 0),
    interest_outstanding = (l.loan_amount * l.interest_rate * l.loan_term_days / 365) - COALESCE(r.total_interest, 0),
    fees_outstanding = l.fee_amount - COALESCE(r.total_fees, 0),
    total_outstanding = (l.loan_amount - COALESCE(r.total_principal, 0)) +
                       ((l.loan_amount * l.interest_rate * l.loan_term_days / 365) - COALESCE(r.total_interest, 0)) +
                       (l.fee_amount - COALESCE(r.total_fees, 0)),
    first_payment_received_date = r.first_payment_date,
    first_payment_missed = (r.first_payment_date IS NULL OR r.first_payment_date > s.first_due_date),
    current_dpd = COALESCE(dpd.current_dpd, 0),
    max_dpd_ever = GREATEST(l.max_dpd_ever, COALESCE(dpd.current_dpd, 0)),
    fimr_tagged = (r.first_payment_date IS NULL OR r.first_payment_date > s.first_due_date),
    early_indicator_tagged = (COALESCE(dpd.current_dpd, 0) BETWEEN 1 AND 6),
    updated_at = CURRENT_TIMESTAMP
FROM (
    SELECT
        loan_id,
        SUM(principal_paid) as total_principal,
        SUM(interest_paid) as total_interest,
        SUM(fees_paid) as total_fees,
        MIN(payment_date) as first_payment_date
    FROM repayments
    WHERE is_reversed = FALSE
    GROUP BY loan_id
) r
LEFT JOIN (
    SELECT loan_id, MIN(due_date) as first_due_date
    FROM loan_schedule
    GROUP BY loan_id
) s ON l.loan_id = s.loan_id
LEFT JOIN (
    SELECT
        loan_id,
        MAX(CURRENT_DATE - due_date) as current_dpd
    FROM loan_schedule
    WHERE payment_status IN ('Pending', 'Partial')
      AND due_date < CURRENT_DATE
    GROUP BY loan_id
) dpd ON l.loan_id = dpd.loan_id
WHERE l.loan_id = r.loan_id;
```

---

#### **Table: `repayments`**
Tracks all payment transactions.

```sql
CREATE TABLE repayments (
    -- Primary Key
    repayment_id VARCHAR(50) PRIMARY KEY,

    -- Loan Reference
    loan_id VARCHAR(50) NOT NULL,

    -- Payment Details
    payment_date DATE NOT NULL,
    payment_amount DECIMAL(15, 2) NOT NULL,

    -- Payment Breakdown
    principal_paid DECIMAL(15, 2) DEFAULT 0,
    interest_paid DECIMAL(15, 2) DEFAULT 0,
    fees_paid DECIMAL(15, 2) DEFAULT 0,
    penalty_paid DECIMAL(15, 2) DEFAULT 0,

    -- Payment Metadata
    payment_method VARCHAR(50), -- 'Bank Transfer', 'Cash', 'Mobile Money', etc.
    payment_reference VARCHAR(100),
    payment_channel VARCHAR(50),

    -- Delinquency at Payment Time
    dpd_at_payment INTEGER DEFAULT 0,

    -- Flags
    is_backdated BOOLEAN DEFAULT FALSE,
    is_reversed BOOLEAN DEFAULT FALSE,
    reversal_date DATE,
    reversal_reason TEXT,

    -- Waiver Information
    waiver_amount DECIMAL(15, 2) DEFAULT 0,
    waiver_type VARCHAR(50), -- 'Interest', 'Fee', 'Penalty'
    waiver_approved_by VARCHAR(100),

    -- Audit Fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_loan FOREIGN KEY (loan_id) REFERENCES loans(loan_id)
);

-- Indexes
CREATE INDEX idx_repayments_loan ON repayments(loan_id);
CREATE INDEX idx_repayments_payment_date ON repayments(payment_date);
CREATE INDEX idx_repayments_loan_date ON repayments(loan_id, payment_date);
CREATE INDEX idx_repayments_backdated ON repayments(is_backdated) WHERE is_backdated = TRUE;
CREATE INDEX idx_repayments_reversed ON repayments(is_reversed) WHERE is_reversed = TRUE;
```

---

#### **Table: `officers`**
Loan officer master data.

```sql
CREATE TABLE officers (
    -- Primary Key
    officer_id VARCHAR(50) PRIMARY KEY,

    -- Personal Information
    officer_name VARCHAR(255) NOT NULL,
    officer_phone VARCHAR(20),
    officer_email VARCHAR(255),

    -- Assignment
    region VARCHAR(100) NOT NULL,
    branch VARCHAR(100) NOT NULL,

    -- Employment
    employment_status VARCHAR(50) DEFAULT 'Active', -- 'Active', 'Inactive', 'Suspended'
    hire_date DATE,
    termination_date DATE,

    -- Channel
    primary_channel VARCHAR(50), -- 'Direct', 'Partner'

    -- Audit Fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_officers_branch ON officers(branch);
CREATE INDEX idx_officers_region ON officers(region);
CREATE INDEX idx_officers_status ON officers(employment_status);
```

---

#### **Table: `customers`**
Customer master data.

```sql
CREATE TABLE customers (
    -- Primary Key
    customer_id VARCHAR(50) PRIMARY KEY,

    -- Personal Information
    customer_name VARCHAR(255) NOT NULL,
    customer_phone VARCHAR(20),
    customer_email VARCHAR(255),

    -- Demographics
    date_of_birth DATE,
    gender VARCHAR(10),

    -- Location
    state VARCHAR(100),
    lga VARCHAR(100),
    address TEXT,

    -- KYC
    kyc_status VARCHAR(50),
    kyc_verified_date DATE,

    -- Audit Fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_customers_phone ON customers(customer_phone);
CREATE INDEX idx_customers_state ON customers(state);
```

---

### 2.2 Derived/Aggregated Tables (Performance Optimization)

#### **Table: `officer_metrics_daily`**
Pre-calculated daily metrics per officer (materialized view).

```sql
CREATE TABLE officer_metrics_daily (
    -- Composite Primary Key
    metric_id SERIAL PRIMARY KEY,
    officer_id VARCHAR(50) NOT NULL,
    calculation_date DATE NOT NULL,

    -- Portfolio Metrics
    total_portfolio DECIMAL(15, 2) DEFAULT 0,
    active_loans INTEGER DEFAULT 0,
    total_disbursed_count INTEGER DEFAULT 0,
    total_disbursed_amount DECIMAL(15, 2) DEFAULT 0,

    -- Delinquency Metrics
    overdue_15d_amount DECIMAL(15, 2) DEFAULT 0,
    overdue_15d_count INTEGER DEFAULT 0,
    par15_ratio DECIMAL(5, 4) DEFAULT 0,

    -- FIMR Components
    first_miss_count INTEGER DEFAULT 0,
    disbursed_count INTEGER DEFAULT 0,
    fimr DECIMAL(5, 4) DEFAULT 0,

    -- D0-6 Slippage Components
    dpd_1to6_balance DECIMAL(15, 2) DEFAULT 0,
    amount_due_7d DECIMAL(15, 2) DEFAULT 0,
    d06_slippage DECIMAL(5, 4) DEFAULT 0,

    -- Roll Components
    moved_to_7to30 DECIMAL(15, 2) DEFAULT 0,
    prev_dpd_1to6_balance DECIMAL(15, 2) DEFAULT 0,
    roll DECIMAL(5, 4) DEFAULT 0,

    -- FRR Components
    fees_collected DECIMAL(15, 2) DEFAULT 0,
    fees_due DECIMAL(15, 2) DEFAULT 0,
    frr DECIMAL(5, 4) DEFAULT 0,

    -- AYR Components
    interest_collected DECIMAL(15, 2) DEFAULT 0,
    par15_mid_month DECIMAL(15, 2) DEFAULT 0,
    ayr DECIMAL(5, 4) DEFAULT 0,

    -- DQI Components
    risk_score_norm DECIMAL(5, 4) DEFAULT 0,
    on_time_rate DECIMAL(5, 4) DEFAULT 0,
    channel_purity DECIMAL(5, 4) DEFAULT 0,
    dqi INTEGER DEFAULT 0,

    -- Risk Score Components
    porr DECIMAL(5, 4) DEFAULT 0,
    waivers_amount DECIMAL(15, 2) DEFAULT 0,
    backdated_count INTEGER DEFAULT 0,
    total_entries INTEGER DEFAULT 0,
    reversals_count INTEGER DEFAULT 0,
    had_float_gap BOOLEAN DEFAULT FALSE,
    risk_score INTEGER DEFAULT 0,
    risk_band VARCHAR(20),

    -- Revenue
    total_yield DECIMAL(15, 2) DEFAULT 0,

    -- Audit
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_officer_metrics FOREIGN KEY (officer_id) REFERENCES officers(officer_id),
    CONSTRAINT unique_officer_date UNIQUE (officer_id, calculation_date)
);

-- Indexes
CREATE INDEX idx_officer_metrics_officer ON officer_metrics_daily(officer_id);
CREATE INDEX idx_officer_metrics_date ON officer_metrics_daily(calculation_date);
CREATE INDEX idx_officer_metrics_officer_date ON officer_metrics_daily(officer_id, calculation_date);
CREATE INDEX idx_officer_metrics_risk_band ON officer_metrics_daily(risk_band);
```

---

#### **Table: `branch_metrics_daily`**
Pre-calculated daily metrics per branch.

```sql
CREATE TABLE branch_metrics_daily (
    -- Composite Primary Key
    metric_id SERIAL PRIMARY KEY,
    branch VARCHAR(100) NOT NULL,
    region VARCHAR(100) NOT NULL,
    calculation_date DATE NOT NULL,

    -- Portfolio Metrics
    portfolio_total DECIMAL(15, 2) DEFAULT 0,
    active_loans INTEGER DEFAULT 0,
    total_officers INTEGER DEFAULT 0,

    -- Delinquency
    overdue_15d DECIMAL(15, 2) DEFAULT 0,
    par15_ratio DECIMAL(5, 4) DEFAULT 0,

    -- Performance Metrics
    ayr DECIMAL(5, 4) DEFAULT 0,
    dqi INTEGER DEFAULT 0,
    fimr DECIMAL(5, 4) DEFAULT 0,

    -- Audit
    calculated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT unique_branch_date UNIQUE (branch, calculation_date)
);

-- Indexes
CREATE INDEX idx_branch_metrics_branch ON branch_metrics_daily(branch);
CREATE INDEX idx_branch_metrics_region ON branch_metrics_daily(region);
CREATE INDEX idx_branch_metrics_date ON branch_metrics_daily(calculation_date);
```

---

#### **Table: `audit_tracking`**
Tracks audit assignments and status for officers.

```sql
CREATE TABLE audit_tracking (
    -- Primary Key
    audit_id SERIAL PRIMARY KEY,

    -- Officer Reference
    officer_id VARCHAR(50) NOT NULL,

    -- Assignment
    assignee_id VARCHAR(50), -- User ID of assigned auditor
    assignee_name VARCHAR(255),
    assigned_date TIMESTAMP,

    -- Status
    audit_status VARCHAR(50) NOT NULL, -- 'In Progress', 'Assigned', 'Resolved'
    status_changed_date TIMESTAMP,

    -- Audit Details
    last_audit_date DATE,
    audit_notes TEXT,
    risk_level VARCHAR(20), -- 'High', 'Medium', 'Low'

    -- Action Taken
    action_type VARCHAR(100), -- 'Audit 20 Top Risk Loans', 'Review Officer', etc.
    action_date TIMESTAMP,
    action_notes TEXT,

    -- Audit Fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_audit_officer FOREIGN KEY (officer_id) REFERENCES officers(officer_id)
);

-- Indexes
CREATE INDEX idx_audit_officer ON audit_tracking(officer_id);
CREATE INDEX idx_audit_assignee ON audit_tracking(assignee_id);
CREATE INDEX idx_audit_status ON audit_tracking(audit_status);
CREATE INDEX idx_audit_last_date ON audit_tracking(last_audit_date);
```

---

#### **Table: `team_members`**
Team members who can be assigned audits.

```sql
CREATE TABLE team_members (
    -- Primary Key
    member_id VARCHAR(50) PRIMARY KEY,

    -- Personal Information
    member_name VARCHAR(255) NOT NULL,
    member_email VARCHAR(255),
    member_phone VARCHAR(20),

    -- Role
    role VARCHAR(100), -- 'Senior Auditor', 'Audit Manager', 'Risk Analyst', etc.
    department VARCHAR(100),

    -- Status
    is_active BOOLEAN DEFAULT TRUE,

    -- Audit Fields
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_team_members_active ON team_members(is_active) WHERE is_active = TRUE;
CREATE INDEX idx_team_members_role ON team_members(role);
```

---

#### **Table: `metric_calculation_log`**
Tracks when metrics were last calculated (for monitoring).

```sql
CREATE TABLE metric_calculation_log (
    -- Primary Key
    log_id SERIAL PRIMARY KEY,

    -- Calculation Details
    calculation_type VARCHAR(100) NOT NULL, -- 'officer_daily', 'branch_daily', 'full_refresh'
    calculation_date DATE NOT NULL,

    -- Performance
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    duration_seconds INTEGER,

    -- Results
    records_processed INTEGER DEFAULT 0,
    records_updated INTEGER DEFAULT 0,
    records_failed INTEGER DEFAULT 0,

    -- Status
    status VARCHAR(50) NOT NULL, -- 'Running', 'Completed', 'Failed'
    error_message TEXT,

    -- Audit
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_calc_log_type ON metric_calculation_log(calculation_type);
CREATE INDEX idx_calc_log_date ON metric_calculation_log(calculation_date);
CREATE INDEX idx_calc_log_status ON metric_calculation_log(status);
```

---

## 3. Metric Calculation Logic

### 3.1 Core Metrics Formulas

All metrics are calculated from the `loans` and `repayments` tables. Here's how each metric is derived:

#### **FIMR (First-Installment Miss Rate)**
```sql
-- Formula: FIMR = firstMiss / disbursed
-- firstMiss: Count of loans where first_payment_missed = TRUE
-- disbursed: Total count of loans disbursed in the period

SELECT
    officer_id,
    COUNT(*) FILTER (WHERE first_payment_missed = TRUE) AS first_miss_count,
    COUNT(*) AS disbursed_count,
    COALESCE(
        COUNT(*) FILTER (WHERE first_payment_missed = TRUE)::DECIMAL / NULLIF(COUNT(*), 0),
        0
    ) AS fimr
FROM loans
WHERE disbursement_date >= :start_date
  AND disbursement_date <= :end_date
  AND status IN ('Active', 'Closed')
GROUP BY officer_id;
```

**Logic:**
- A loan is tagged as `first_payment_missed = TRUE` if:
  - `first_payment_due_date` has passed
  - `first_payment_received_date` is NULL OR `first_payment_received_date > first_payment_due_date`

---

#### **D0-6 Slippage (Early Slippage)**
```sql
-- Formula: D0-6 Slippage = dpd1to6Bal / amountDue7d
-- dpd1to6Bal: Sum of outstanding balance for loans with DPD between 1-6
-- amountDue7d: Sum of all payments due in the next 7 days

WITH dpd_1to6 AS (
    SELECT
        officer_id,
        SUM(total_outstanding) AS dpd_1to6_balance
    FROM loans
    WHERE current_dpd BETWEEN 1 AND 6
      AND status = 'Active'
    GROUP BY officer_id
),
amount_due_7d AS (
    SELECT
        l.officer_id,
        SUM(ls.amount_due) AS amount_due_7d
    FROM loans l
    JOIN loan_schedule ls ON l.loan_id = ls.loan_id
    WHERE ls.due_date BETWEEN CURRENT_DATE AND CURRENT_DATE + INTERVAL '7 days'
      AND ls.payment_status = 'Pending'
      AND l.status = 'Active'
    GROUP BY l.officer_id
)
SELECT
    COALESCE(d.officer_id, a.officer_id) AS officer_id,
    COALESCE(d.dpd_1to6_balance, 0) AS dpd_1to6_balance,
    COALESCE(a.amount_due_7d, 0) AS amount_due_7d,
    COALESCE(d.dpd_1to6_balance, 0) / NULLIF(COALESCE(a.amount_due_7d, 0), 0) AS d06_slippage
FROM dpd_1to6 d
FULL OUTER JOIN amount_due_7d a ON d.officer_id = a.officer_id;
```

**Note:** This requires a `loan_schedule` table (see Additional Tables section).

---

#### **Roll (0-6 → 7-30)**
```sql
-- Formula: Roll = movedTo7to30 / prevDpd1to6Bal
-- movedTo7to30: Amount that moved from DPD 1-6 to DPD 7-30 in the current period
-- prevDpd1to6Bal: DPD 1-6 balance from the previous period

-- This requires tracking DPD transitions over time
-- Simplified version using snapshots:

WITH current_period AS (
    SELECT
        officer_id,
        SUM(total_outstanding) AS current_7to30_balance
    FROM loans
    WHERE current_dpd BETWEEN 7 AND 30
      AND status = 'Active'
    GROUP BY officer_id
),
previous_period AS (
    SELECT
        officer_id,
        dpd_1to6_balance AS prev_dpd_1to6_balance
    FROM officer_metrics_daily
    WHERE calculation_date = CURRENT_DATE - INTERVAL '1 day'
)
SELECT
    c.officer_id,
    c.current_7to30_balance AS moved_to_7to30,
    p.prev_dpd_1to6_balance,
    c.current_7to30_balance / NULLIF(p.prev_dpd_1to6_balance, 0) AS roll
FROM current_period c
LEFT JOIN previous_period p ON c.officer_id = p.officer_id;
```

**Better Approach:** Track DPD transitions in a separate `dpd_transitions` table.

---

#### **FRR (Fees Realization Rate)**
```sql
-- Formula: FRR = feesCollected / feesDue
-- feesCollected: Sum of fees_paid in the period
-- feesDue: Sum of fee_amount for all active loans

SELECT
    l.officer_id,
    SUM(r.fees_paid) AS fees_collected,
    SUM(l.fee_amount) AS fees_due,
    SUM(r.fees_paid) / NULLIF(SUM(l.fee_amount), 0) AS frr
FROM loans l
LEFT JOIN repayments r ON l.loan_id = r.loan_id
WHERE r.payment_date >= :start_date
  AND r.payment_date <= :end_date
  AND r.is_reversed = FALSE
GROUP BY l.officer_id;
```

---

#### **AYR (Adjusted Yield Ratio)**
```sql
-- Formula: AYR = (Interest + Fees collected) / PAR15 at mid-month
-- Numerator: Interest + Fees collected in the current month
-- Denominator: PAR15 exposure at mid-month (15th of the month)

WITH monthly_collections AS (
    SELECT
        l.officer_id,
        SUM(r.interest_paid + r.fees_paid) AS total_yield
    FROM loans l
    JOIN repayments r ON l.loan_id = r.loan_id
    WHERE r.payment_date >= DATE_TRUNC('month', CURRENT_DATE)
      AND r.payment_date < DATE_TRUNC('month', CURRENT_DATE) + INTERVAL '1 month'
      AND r.is_reversed = FALSE
    GROUP BY l.officer_id
),
par15_mid_month AS (
    SELECT
        officer_id,
        SUM(total_outstanding) AS par15_exposure
    FROM loans
    WHERE current_dpd > 15
      AND status = 'Active'
      -- Snapshot taken on 15th of the month
    GROUP BY officer_id
)
SELECT
    COALESCE(c.officer_id, p.officer_id) AS officer_id,
    COALESCE(c.total_yield, 0) AS total_yield,
    COALESCE(p.par15_exposure, 0) AS par15_mid_month,
    COALESCE(c.total_yield, 0) / NULLIF(COALESCE(p.par15_exposure, 0), 0) AS ayr
FROM monthly_collections c
FULL OUTER JOIN par15_mid_month p ON c.officer_id = p.officer_id;
```

**Note:** PAR15 mid-month should be captured as a snapshot on the 15th of each month.

---

#### **DQI (Delinquency Quality Index)**
```javascript
// Formula: DQI = 100 * (0.4 * RQ + 0.35 * OTI + 0.25 * (1 - FIMR)) * CP_toggle
// This is calculated in application code, not SQL

function calculateDQI(riskScoreNorm, onTimeRate, fimr, channelPurity, cpToggle = true) {
    const rq = Math.max(0, Math.min(1, riskScoreNorm));
    const oti = Math.max(0, Math.min(1, onTimeRate));
    const fimrClamped = Math.max(0, Math.min(1, fimr));
    const cp = cpToggle ? Math.max(0, Math.min(1, channelPurity)) : 1;

    const dqi = 100 * (0.4 * rq + 0.35 * oti + 0.25 * (1 - fimrClamped)) * cp;
    return Math.round(dqi);
}
```

**Components:**
- `riskScoreNorm`: Normalized risk score (0-1)
- `onTimeRate`: Percentage of on-time payments (0-1)
- `fimr`: First-Installment Miss Rate (0-1)
- `channelPurity`: Channel purity score (0-1)

---

#### **Risk Score (Composite Officer Risk Score)**
```javascript
// Formula: RiskScore = 100 - penalties
// Calculated in application code

function calculateRiskScore(params) {
    const {
        porr = 0,
        fimr = 0,
        roll = 0,
        waivers = 0,
        amountDue7d = 0,
        backdated = 0,
        entries = 0,
        reversals = 0,
        frr = 0,
        channelPurity = 1,
        hadFloatGap = false,
    } = params;

    let score = 100;
    score -= 20 * Math.min(1, porr);
    score -= 15 * Math.min(1, fimr);
    score -= 10 * Math.min(1, roll);
    score -= 10 * (amountDue7d > 0 ? waivers / amountDue7d : 0);
    score -= 10 * (entries > 0 ? backdated / entries : 0);
    score -= 10 * (entries > 0 ? reversals / entries : 0);
    score -= 10 * Math.min(1, 1 - frr);
    score -= 5 * Math.min(1, 1 - channelPurity);
    score -= 10 * (hadFloatGap ? 1 : 0);

    return Math.max(0, Math.round(score));
}
```

**Risk Bands:**
- **Green**: ≥ 80
- **Watch**: 60-79
- **Amber**: 40-59
- **Red/Flag**: < 40

---

### 3.2 Calculation Strategy: Pre-Aggregation vs On-Demand

| Metric | Strategy | Reason |
|--------|----------|--------|
| **Officer Daily Metrics** | Pre-aggregated | Heavy calculations, updated every 15-30 min |
| **Branch Daily Metrics** | Pre-aggregated | Aggregation of officer metrics |
| **FIMR Drilldown** | On-demand | Filtered loan-level data, dynamic filters |
| **Early Indicators Drilldown** | On-demand | Filtered loan-level data, dynamic filters |
| **Agent Performance** | Pre-aggregated + Cache | Read from `officer_metrics_daily` table |
| **Credit Health by Branch** | Pre-aggregated + Cache | Read from `branch_metrics_daily` table |

**Caching Strategy:**
- **Redis Cache**: Store frequently accessed data (officer metrics, branch metrics)
- **TTL**: 15 minutes (aligned with batch calculation frequency)
- **Cache Keys**:
  - `officer_metrics:{officer_id}:{date}`
  - `branch_metrics:{branch}:{date}`
  - `filter_options:regions`
  - `filter_options:branches`
  - `filter_options:officers`

---

## 4. Additional Required Tables

### 4.1 Loan Schedule Table

Required for calculating `amountDue7d` (D0-6 Slippage).

```sql
CREATE TABLE loan_schedule (
    -- Primary Key
    schedule_id SERIAL PRIMARY KEY,

    -- Loan Reference
    loan_id VARCHAR(50) NOT NULL,

    -- Schedule Details
    installment_number INTEGER NOT NULL,
    due_date DATE NOT NULL,

    -- Amounts
    principal_due DECIMAL(15, 2) NOT NULL,
    interest_due DECIMAL(15, 2) NOT NULL,
    fee_due DECIMAL(15, 2) DEFAULT 0,
    total_due DECIMAL(15, 2) NOT NULL,

    -- Payment Status
    payment_status VARCHAR(50) DEFAULT 'Pending', -- 'Pending', 'Paid', 'Partial', 'Overdue'
    amount_paid DECIMAL(15, 2) DEFAULT 0,
    payment_date DATE,

    -- Audit
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_schedule_loan FOREIGN KEY (loan_id) REFERENCES loans(loan_id),
    CONSTRAINT unique_loan_installment UNIQUE (loan_id, installment_number)
);

-- Indexes
CREATE INDEX idx_schedule_loan ON loan_schedule(loan_id);
CREATE INDEX idx_schedule_due_date ON loan_schedule(due_date);
CREATE INDEX idx_schedule_status ON loan_schedule(payment_status);
CREATE INDEX idx_schedule_loan_status ON loan_schedule(loan_id, payment_status);
```

---

### 4.2 DPD Transitions Table

Required for accurately calculating Roll (0-6 → 7-30).

```sql
CREATE TABLE dpd_transitions (
    -- Primary Key
    transition_id SERIAL PRIMARY KEY,

    -- Loan Reference
    loan_id VARCHAR(50) NOT NULL,
    officer_id VARCHAR(50) NOT NULL,

    -- Transition Details
    transition_date DATE NOT NULL,
    from_dpd_bucket VARCHAR(20), -- '0', '1-6', '7-30', '31-60', '61-90', '90+'
    to_dpd_bucket VARCHAR(20),

    -- Amount
    outstanding_balance DECIMAL(15, 2) NOT NULL,

    -- Audit
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_transition_loan FOREIGN KEY (loan_id) REFERENCES loans(loan_id),
    CONSTRAINT fk_transition_officer FOREIGN KEY (officer_id) REFERENCES officers(officer_id)
);

-- Indexes
CREATE INDEX idx_transitions_loan ON dpd_transitions(loan_id);
CREATE INDEX idx_transitions_officer ON dpd_transitions(officer_id);
CREATE INDEX idx_transitions_date ON dpd_transitions(transition_date);
CREATE INDEX idx_transitions_buckets ON dpd_transitions(from_dpd_bucket, to_dpd_bucket);
```

---

### 4.3 PAR15 Snapshots Table

Required for AYR calculation (PAR15 at mid-month).

```sql
CREATE TABLE par15_snapshots (
    -- Primary Key
    snapshot_id SERIAL PRIMARY KEY,

    -- Snapshot Details
    snapshot_date DATE NOT NULL,
    officer_id VARCHAR(50) NOT NULL,

    -- PAR15 Metrics
    par15_exposure DECIMAL(15, 2) NOT NULL,
    par15_count INTEGER DEFAULT 0,

    -- Audit
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_snapshot_officer FOREIGN KEY (officer_id) REFERENCES officers(officer_id),
    CONSTRAINT unique_officer_snapshot UNIQUE (officer_id, snapshot_date)
);

-- Indexes
CREATE INDEX idx_snapshots_officer ON par15_snapshots(officer_id);
CREATE INDEX idx_snapshots_date ON par15_snapshots(snapshot_date);
```

---

## 5. REST API Endpoints

### 5.1 Officer Metrics Endpoints

#### **GET /api/v1/metrics/officers**
Get aggregated officer metrics with filters.

**Query Parameters:**
- `region` (optional): Filter by region
- `branch` (optional): Filter by branch
- `officer_id` (optional): Filter by specific officer
- `date` (optional): Specific date (default: latest)
- `start_date` (optional): Date range start
- `end_date` (optional): Date range end
- `risk_band` (optional): Filter by risk band (Green, Watch, Amber, Red)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "officer_id": "OFF001",
      "officer_name": "John Doe",
      "region": "Lagos",
      "branch": "Lagos Main",
      "risk_score": 85,
      "risk_band": "Green",
      "assignee": "Assigned to Sarah Johnson",
      "audit_status": "In Progress",
      "last_audit_date": "2024-10-15",
      "ayr": 1.85,
      "dqi": 92,
      "fimr": 0.030,
      "all_time_fimr": 0.085,
      "d06_slippage": 0.050,
      "roll": 0.250,
      "frr": 0.900,
      "portfolio_total": 50000000,
      "overdue_15d": 1200000,
      "active_loans": 5000,
      "channel": "Direct",
      "yield": 2550000,
      "porr": 0.024,
      "channel_purity": 0.95,
      "rank": 1
    }
  ],
  "meta": {
    "total": 150,
    "page": 1,
    "per_page": 50,
    "calculation_date": "2024-10-17"
  }
}
```

---

#### **GET /api/v1/metrics/officers/:officer_id**
Get detailed metrics for a specific officer.

**Response:**
```json
{
  "success": true,
  "data": {
    "officer_id": "OFF001",
    "officer_name": "John Doe",
    "officer_phone": "+234-803-123-4567",
    "region": "Lagos",
    "branch": "Lagos Main",
    "metrics": {
      "current": { /* current period metrics */ },
      "previous": { /* previous period metrics */ },
      "trend": "improving" // "improving", "declining", "stable"
    },
    "audit_info": {
      "assignee": "Assigned to Sarah Johnson",
      "audit_status": "In Progress",
      "last_audit_date": "2024-10-15",
      "audit_notes": "..."
    }
  }
}
```

---

### 5.2 Loan Drilldown Endpoints

#### **GET /api/v1/loans/fimr-drilldown**
Get loan-level data for FIMR analysis.

**Query Parameters:**
- `region` (optional)
- `branch` (optional)
- `officer_id` (optional)
- `start_date` (optional): Disbursement date range start
- `end_date` (optional): Disbursement date range end
- `fimr_tagged` (optional): true/false
- `page` (optional): Page number
- `per_page` (optional): Results per page (default: 50, max: 500)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "loan_id": "LN001",
      "customer_name": "Adebayo Ogunlesi",
      "customer_phone": "+234-803-456-7890",
      "officer_name": "John Doe",
      "region": "Lagos",
      "branch": "Lagos Main",
      "disbursement_date": "2024-09-15",
      "loan_amount": 500000,
      "first_payment_due_date": "2024-09-22",
      "first_payment_received_date": "2024-09-25",
      "days_late": 3,
      "fimr_tagged": true,
      "current_dpd": 0,
      "status": "Active",
      "outstanding_balance": 450000
    }
  ],
  "meta": {
    "total": 1250,
    "page": 1,
    "per_page": 50,
    "total_pages": 25
  }
}
```

---

#### **GET /api/v1/loans/early-indicators-drilldown**
Get loan-level data for early indicators (DPD 1-6).

**Query Parameters:**
- Same as FIMR drilldown
- `early_indicator_tagged` (optional): true/false

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "loan_id": "LN002",
      "customer_name": "Chioma Nwosu",
      "customer_phone": "+234-701-234-5678",
      "officer_name": "John Doe",
      "region": "Lagos",
      "branch": "Lagos Main",
      "disbursement_date": "2024-08-20",
      "loan_amount": 750000,
      "current_dpd": 4,
      "outstanding_balance": 680000,
      "last_payment_date": "2024-09-15",
      "next_payment_due": "2024-10-10",
      "early_indicator_tagged": true,
      "fimr_tagged": false,
      "status": "Active"
    }
  ],
  "meta": {
    "total": 850,
    "page": 1,
    "per_page": 50
  }
}
```

---

### 5.3 Branch Metrics Endpoints

#### **GET /api/v1/metrics/branches**
Get branch-level aggregated metrics.

**Query Parameters:**
- `region` (optional)
- `date` (optional): Specific date (default: latest)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "branch": "Lagos Main",
      "region": "Lagos",
      "portfolio_total": 50000000,
      "overdue_15d": 1200000,
      "par15_ratio": 0.024,
      "ayr": 1.85,
      "dqi": 92,
      "fimr": 0.030,
      "active_loans": 5000,
      "total_officers": 15
    },
    {
      "branch": "Abuja Central",
      "region": "Abuja",
      "portfolio_total": 45000000,
      "overdue_15d": 2800000,
      "par15_ratio": 0.062,
      "ayr": 1.42,
      "dqi": 78,
      "fimr": 0.067,
      "active_loans": 4200,
      "total_officers": 12
    }
  ],
  "meta": {
    "total": 25,
    "calculation_date": "2024-10-17"
  }
}
```

---

### 5.4 Audit Management Endpoints

#### **GET /api/v1/audit/team-members**
Get list of team members for assignment.

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "member_id": "TM001",
      "member_name": "Sarah Johnson",
      "role": "Audit Manager",
      "department": "Risk Management",
      "is_active": true
    },
    {
      "member_id": "TM002",
      "member_name": "John Smith",
      "role": "Senior Auditor",
      "department": "Audit",
      "is_active": true
    }
  ]
}
```

---

#### **PUT /api/v1/audit/officers/:officer_id/assignee**
Update assignee for an officer.

**Request Body:**
```json
{
  "assignee_id": "TM001",
  "assignee_name": "Sarah Johnson"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Assignee updated successfully",
  "data": {
    "officer_id": "OFF001",
    "assignee_id": "TM001",
    "assignee_name": "Sarah Johnson",
    "assigned_date": "2024-10-17T10:30:00Z"
  }
}
```

---

#### **PUT /api/v1/audit/officers/:officer_id/status**
Update audit status for an officer.

**Request Body:**
```json
{
  "audit_status": "In Progress",
  "audit_notes": "Started reviewing top 20 risk loans"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Audit status updated successfully",
  "data": {
    "officer_id": "OFF001",
    "audit_status": "In Progress",
    "last_audit_date": "2024-10-17",
    "status_changed_date": "2024-10-17T10:30:00Z"
  }
}
```

---

#### **POST /api/v1/audit/officers/:officer_id/actions**
Record an action taken on an officer.

**Request Body:**
```json
{
  "action_type": "Audit 20 Top Risk Loans",
  "action_notes": "Reviewed top 20 loans by risk score"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Action recorded successfully",
  "data": {
    "officer_id": "OFF001",
    "action_type": "Audit 20 Top Risk Loans",
    "action_date": "2024-10-17T10:30:00Z",
    "top_risk_loans": [
      {
        "loan_id": "LN001",
        "customer_name": "...",
        "risk_score": 35,
        "current_dpd": 45
      }
    ]
  }
}
```

---

### 5.5 Filter Options Endpoints

#### **GET /api/v1/filters/options**
Get all available filter options.

**Response:**
```json
{
  "success": true,
  "data": {
    "regions": ["Lagos", "Abuja", "Kano", "Port Harcourt"],
    "branches": [
      { "branch": "Lagos Main", "region": "Lagos" },
      { "branch": "Abuja Central", "region": "Abuja" }
    ],
    "officers": [
      { "officer_id": "OFF001", "officer_name": "John Doe", "branch": "Lagos Main" }
    ],
    "risk_bands": ["Green", "Watch", "Amber", "Red"],
    "audit_statuses": ["In Progress", "Assigned", "Resolved"],
    "loan_statuses": ["Active", "Closed", "Written Off"]
  }
}
```

---

## 5.5 ETL Data Payload Examples

### Complete JSON Payloads for ETL Integration

#### **1. Add New Loan - Complete Payload**

```json
{
  "loan_id": "LN2024001234",
  "customer_id": "CUST20240567",
  "customer_name": "Adebayo Oluwaseun",
  "customer_phone": "+234-803-456-7890",
  "officer_id": "OFF2024012",
  "officer_name": "Sarah Johnson",
  "officer_phone": "+234-803-987-6543",
  "region": "South West",
  "branch": "Lagos Main",
  "state": "Lagos",
  "loan_amount": 500000.00,
  "disbursement_date": "2024-10-15",
  "maturity_date": "2025-04-15",
  "loan_term_days": 180,
  "interest_rate": 0.1500,
  "fee_amount": 25000.00,
  "channel": "Direct",
  "channel_partner": null,
  "status": "Active",
  "closed_date": null
}
```

**⚠️ CRITICAL:** Do NOT include computed fields (`current_dpd`, `principal_outstanding`, `total_principal_paid`, `fimr_tagged`, etc.). These will be automatically calculated by the analytics service.

---

#### **2. Add New Repayment - Complete Payload**

```json
{
  "repayment_id": "REP2024005678",
  "loan_id": "LN2024001234",
  "payment_date": "2024-11-01",
  "payment_amount": 100000.00,
  "principal_paid": 80000.00,
  "interest_paid": 15000.00,
  "fees_paid": 5000.00,
  "penalty_paid": 0.00,
  "payment_method": "Bank Transfer",
  "payment_reference": "TXN20241101123456",
  "payment_channel": "Mobile App",
  "dpd_at_payment": 0,
  "is_backdated": false,
  "is_reversed": false,
  "reversal_date": null,
  "reversal_reason": null,
  "waiver_amount": 0.00,
  "waiver_type": null,
  "waiver_approved_by": null
}
```

**Validation:** `payment_amount` should equal `principal_paid + interest_paid + fees_paid + penalty_paid`

---

## 6. Technology Stack Recommendations

### 6.1 Backend Language: **Go (Golang)** ⭐ RECOMMENDED

**Why Go?**
- ✅ **Performance**: 10-100x faster than Node.js for CPU-intensive calculations
- ✅ **Concurrency**: Built-in goroutines perfect for parallel metric calculations
- ✅ **Memory Efficiency**: Lower memory footprint for large datasets
- ✅ **Type Safety**: Compile-time error detection
- ✅ **Deployment**: Single binary, easy containerization
- ✅ **Database Performance**: Excellent PostgreSQL drivers

**Use Cases:**
- Heavy metric calculations (FIMR, AYR, DQI, Risk Score)
- Batch processing jobs
- High-throughput API endpoints
- Real-time data aggregation

**Sample Go Structure:**
```
analytics-service/
├── cmd/
│   ├── api/          # API server
│   └── worker/       # Background workers
├── internal/
│   ├── models/       # Database models
│   ├── metrics/      # Metric calculation logic
│   ├── handlers/     # HTTP handlers
│   ├── services/     # Business logic
│   └── repository/   # Database access
├── pkg/
│   ├── cache/        # Redis cache
│   └── utils/        # Utilities
└── migrations/       # Database migrations
```

---

### 6.2 Alternative #1: **Python (FastAPI/Django)**

**Why Python?**
- ✅ **Data Science Ecosystem**: NumPy, Pandas for complex calculations
- ✅ **Rapid Development**: Clean syntax, fast prototyping
- ✅ **Team Familiarity**: Widely known language
- ✅ **Rich Libraries**: Extensive ecosystem for analytics
- ✅ **Type Hints**: Modern Python supports type safety
- ✅ **Async Support**: FastAPI provides async/await for concurrency

**Use Cases:**
- Teams with strong Python background
- Integration with data science workflows
- Rapid prototyping and iteration
- Complex statistical calculations

**Performance Considerations:**

| Aspect | Python | Go | Node.js |
|--------|--------|----|----|
| **Raw Performance** | ⭐⭐ (2-5x slower than Go) | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Concurrency** | ⭐⭐⭐ (asyncio, limited by GIL) | ⭐⭐⭐⭐⭐ (goroutines) | ⭐⭐⭐⭐ (event loop) |
| **Memory Usage** | ⭐⭐⭐ (higher than Go) | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **Development Speed** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Data Processing** | ⭐⭐⭐⭐⭐ (Pandas, NumPy) | ⭐⭐⭐ | ⭐⭐⭐ |
| **Deployment** | ⭐⭐⭐ (requires interpreter) | ⭐⭐⭐⭐⭐ (single binary) | ⭐⭐⭐⭐ |

**⚠️ Potential Issues with Python:**

1. **Global Interpreter Lock (GIL)**
   - Python's GIL prevents true multi-threading for CPU-bound tasks
   - **Impact**: Can't fully utilize multiple CPU cores for parallel metric calculations
   - **Workaround**: Use multiprocessing (more memory overhead) or async I/O (for I/O-bound tasks only)

2. **Performance for Large Datasets**
   - 2-5x slower than Go for CPU-intensive calculations
   - **Impact**: Metric calculations for 100K+ loans may take 10-25 minutes instead of 5 minutes
   - **Workaround**: Use NumPy/Pandas (C-optimized), or Cython for critical paths

3. **Memory Footprint**
   - Python uses more memory than Go (typically 2-3x)
   - **Impact**: Higher infrastructure costs, especially for large datasets
   - **Workaround**: Optimize data structures, use generators, process in batches

4. **Deployment Complexity**
   - Requires Python runtime + dependencies
   - **Impact**: Larger Docker images, more complex deployment
   - **Workaround**: Use multi-stage Docker builds, Alpine Linux base images

5. **Type Safety**
   - Type hints are optional and not enforced at runtime
   - **Impact**: More runtime errors, harder to catch bugs early
   - **Workaround**: Use mypy for static type checking, Pydantic for runtime validation

**When Python Makes Sense:**
- ✅ Team already has strong Python expertise
- ✅ Dataset is small-to-medium (< 50K loans)
- ✅ Need to integrate with existing Python data pipelines
- ✅ Rapid prototyping is more important than raw performance
- ✅ Planning to add ML/AI features later (fraud detection, credit scoring)

**When Python Doesn't Make Sense:**
- ❌ Need maximum performance (100K+ loans, sub-second API responses)
- ❌ High concurrency requirements (1000+ requests/sec)
- ❌ Limited infrastructure budget (need to minimize server costs)
- ❌ Team has Go expertise

**Sample Python (FastAPI) Structure:**
```
analytics-service/
├── app/
│   ├── models/           # SQLAlchemy models
│   ├── schemas/          # Pydantic schemas
│   ├── metrics/          # Metric calculation logic
│   ├── routers/          # API routes
│   ├── services/         # Business logic
│   ├── repositories/     # Database access
│   ├── core/             # Config, dependencies
│   └── utils/            # Utilities
├── alembic/              # Database migrations
├── tests/                # Unit tests
└── requirements.txt      # Dependencies
```

**Sample Python Code (Metric Calculation):**
```python
from typing import List, Dict
from datetime import date
import pandas as pd
from sqlalchemy.orm import Session

async def calculate_officer_metrics(
    db: Session,
    officer_id: str,
    calculation_date: date
) -> Dict:
    """Calculate all metrics for an officer."""

    # Fetch loans using pandas for efficient calculation
    query = """
        SELECT loan_id, loan_amount, disbursement_date,
               current_dpd, fimr_tagged, total_outstanding
        FROM loans
        WHERE officer_id = :officer_id
          AND disbursement_date <= :calc_date
    """

    df = pd.read_sql(query, db.bind, params={
        'officer_id': officer_id,
        'calc_date': calculation_date
    })

    # Calculate metrics using pandas
    total_disbursed = len(df)
    total_amount = df['loan_amount'].sum()
    fimr_count = df['fimr_tagged'].sum()
    fimr_rate = (fimr_count / total_disbursed * 100) if total_disbursed > 0 else 0

    par15 = df[df['current_dpd'] >= 15]['total_outstanding'].sum()
    par15_rate = (par15 / total_amount * 100) if total_amount > 0 else 0

    return {
        'officer_id': officer_id,
        'calculation_date': calculation_date,
        'total_disbursed': total_disbursed,
        'total_amount': float(total_amount),
        'fimr_rate': round(fimr_rate, 2),
        'par15_rate': round(par15_rate, 2)
    }
```

**Verdict on Python:**
- ✅ **Good for**: Prototyping, small-medium datasets, teams with Python expertise
- ⚠️ **Acceptable for**: Medium datasets (50K-100K loans) with optimization
- ❌ **Not recommended for**: Large datasets (100K+ loans), high-performance requirements

---

### 6.3 Alternative #2: **Node.js (TypeScript)**

**Why Node.js?**
- ✅ **Rapid Development**: Faster prototyping
- ✅ **JavaScript Ecosystem**: Reuse frontend calculation logic
- ✅ **Team Familiarity**: If team already knows JavaScript
- ✅ **Rich Libraries**: Extensive npm ecosystem
- ✅ **Async I/O**: Excellent for I/O-bound operations

**Performance Considerations:**
- 5-10x slower than Go for CPU-intensive calculations
- Good for I/O-bound operations (database queries, API calls)
- Single-threaded (can use worker threads, but more complex)

**Use Cases:**
- Rapid prototyping
- Teams with strong JavaScript background
- Integration with existing Node.js infrastructure
- I/O-heavy workloads

**Sample Node.js Structure:**
```
analytics-service/
├── src/
│   ├── models/       # TypeORM models
│   ├── metrics/      # Metric calculations
│   ├── controllers/  # Route controllers
│   ├── services/     # Business logic
│   ├── repositories/ # Database access
│   └── utils/        # Utilities
├── migrations/       # Database migrations
└── tests/            # Unit tests
```

---

### 6.4 Technology Stack Comparison Summary

| Aspect | Go | Python | Node.js |
|--------|----|----|---------|
| **Performance** | ⭐⭐⭐⭐⭐ (Fastest) | ⭐⭐ (2-5x slower) | ⭐⭐⭐ (5-10x slower) |
| **Concurrency** | ⭐⭐⭐⭐⭐ (Goroutines) | ⭐⭐⭐ (Asyncio, GIL limits) | ⭐⭐⭐⭐ (Event loop) |
| **Memory** | ⭐⭐⭐⭐⭐ (Lowest) | ⭐⭐⭐ (2-3x more) | ⭐⭐⭐ (2-3x more) |
| **Development Speed** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ (Fastest) | ⭐⭐⭐⭐ |
| **Type Safety** | ⭐⭐⭐⭐⭐ (Compile-time) | ⭐⭐⭐ (Optional hints) | ⭐⭐⭐⭐ (TypeScript) |
| **Deployment** | ⭐⭐⭐⭐⭐ (Single binary) | ⭐⭐⭐ (Interpreter needed) | ⭐⭐⭐⭐ (Node runtime) |
| **Data Processing** | ⭐⭐⭐ | ⭐⭐⭐⭐⭐ (Pandas, NumPy) | ⭐⭐⭐ |
| **Ecosystem** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **Learning Curve** | ⭐⭐⭐ (Moderate) | ⭐⭐⭐⭐⭐ (Easy) | ⭐⭐⭐⭐ (Easy) |
| **Infrastructure Cost** | ⭐⭐⭐⭐⭐ (Lowest) | ⭐⭐⭐ (Higher) | ⭐⭐⭐ (Higher) |

**Recommendation by Use Case:**

| Scenario | Best Choice | Reason |
|----------|-------------|--------|
| **Production, 100K+ loans** | **Go** | Performance, concurrency, low cost |
| **Production, 50K-100K loans** | **Go or Python** | Either works with optimization |
| **Production, < 50K loans** | **Any** | Performance not critical |
| **Rapid Prototyping** | **Python or Node.js** | Faster development |
| **Team knows Python** | **Python** | Leverage existing skills |
| **Team knows JavaScript** | **Node.js** | Leverage existing skills |
| **Team knows Go** | **Go** | Best performance |
| **Future ML/AI features** | **Python** | Best ML ecosystem |
| **Budget-constrained** | **Go** | Lowest infrastructure cost |

**Final Recommendation:** **Go for production, Python for prototyping**

---

### 6.5 Database: **PostgreSQL 14+**

### 6.3 Database: **PostgreSQL 14+**

**Why PostgreSQL?**
- ✅ **ACID Compliance**: Data integrity
- ✅ **Advanced Indexing**: B-tree, GiST, GIN for performance
- ✅ **Materialized Views**: Pre-computed aggregations
- ✅ **Window Functions**: Complex analytics queries
- ✅ **JSON Support**: Flexible schema for audit logs
- ✅ **Partitioning**: Scale large tables (loans, repayments)

**Recommended Configuration:**
```sql
-- Enable parallel query execution
SET max_parallel_workers_per_gather = 4;

-- Increase work memory for complex queries
SET work_mem = '256MB';

-- Enable JIT compilation for faster queries
SET jit = on;
```

---

### 6.6 Caching: **Redis 7+**

**Why Redis?**
- ✅ **In-Memory Speed**: Sub-millisecond response times
- ✅ **Data Structures**: Hashes, Sets, Sorted Sets for complex caching
- ✅ **TTL Support**: Automatic cache expiration
- ✅ **Pub/Sub**: Real-time updates

**Cache Strategy:**
```javascript
// Cache officer metrics for 15 minutes
const cacheKey = `officer_metrics:${officerId}:${date}`;
const ttl = 900; // 15 minutes

// Try cache first
let metrics = await redis.get(cacheKey);
if (!metrics) {
    // Calculate from database
    metrics = await calculateOfficerMetrics(officerId, date);
    await redis.setex(cacheKey, ttl, JSON.stringify(metrics));
}
```

---

## 7. Data Synchronization Implementation

### 7.1 Batch Processing Job (Every 15-30 minutes)

**Go Implementation:**
```go
package worker

import (
    "context"
    "time"
)

type MetricsCalculator struct {
    db    *sql.DB
    redis *redis.Client
}

func (mc *MetricsCalculator) CalculateOfficerMetrics(ctx context.Context) error {
    // 1. Get all active officers
    officers, err := mc.db.GetActiveOfficers(ctx)
    if err != nil {
        return err
    }

    // 2. Calculate metrics in parallel using goroutines
    results := make(chan OfficerMetrics, len(officers))
    errors := make(chan error, len(officers))

    for _, officer := range officers {
        go func(o Officer) {
            metrics, err := mc.calculateForOfficer(ctx, o)
            if err != nil {
                errors <- err
                return
            }
            results <- metrics
        }(officer)
    }

    // 3. Collect results
    for i := 0; i < len(officers); i++ {
        select {
        case metrics := <-results:
            // Save to database
            mc.db.SaveOfficerMetrics(ctx, metrics)
            // Update cache
            mc.redis.Set(ctx, fmt.Sprintf("officer_metrics:%s:%s",
                metrics.OfficerID, time.Now().Format("2006-01-02")),
                metrics, 15*time.Minute)
        case err := <-errors:
            log.Error("Error calculating metrics", err)
        }
    }

    return nil
}

func (mc *MetricsCalculator) calculateForOfficer(ctx context.Context, officer Officer) (OfficerMetrics, error) {
    // Calculate all metrics
    fimr := mc.calculateFIMR(ctx, officer.ID)
    slippage := mc.calculateSlippage(ctx, officer.ID)
    roll := mc.calculateRoll(ctx, officer.ID)
    // ... other metrics

    return OfficerMetrics{
        OfficerID: officer.ID,
        FIMR: fimr,
        Slippage: slippage,
        Roll: roll,
        // ... other fields
    }, nil
}
```

---

### 7.2 Scheduled Jobs (Cron)

**Recommended Schedule:**
```bash
# Every 15 minutes - Calculate officer metrics
*/15 * * * * /app/worker calculate-officer-metrics

# Every 30 minutes - Calculate branch metrics
*/30 * * * * /app/worker calculate-branch-metrics

# Daily at 2 AM - Full refresh and cleanup
0 2 * * * /app/worker full-refresh

# 15th of every month at 12 PM - Capture PAR15 snapshot
0 12 15 * * /app/worker capture-par15-snapshot

# Daily at 3 AM - Archive old data
0 3 * * * /app/worker archive-old-data
```

---

## 8. Performance Optimization Strategies

### 8.1 Database Optimization

#### **Table Partitioning**
Partition large tables by date for better query performance.

```sql
-- Partition loans table by disbursement_date (monthly)
CREATE TABLE loans_partitioned (
    LIKE loans INCLUDING ALL
) PARTITION BY RANGE (disbursement_date);

-- Create partitions for each month
CREATE TABLE loans_2024_01 PARTITION OF loans_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

CREATE TABLE loans_2024_02 PARTITION OF loans_partitioned
    FOR VALUES FROM ('2024-02-01') TO ('2024-03-01');
-- ... continue for each month
```

#### **Materialized Views**
Pre-compute complex aggregations.

```sql
-- Materialized view for officer metrics
CREATE MATERIALIZED VIEW mv_officer_metrics_current AS
SELECT
    o.officer_id,
    o.officer_name,
    o.region,
    o.branch,
    COUNT(l.loan_id) AS active_loans,
    SUM(l.total_outstanding) AS portfolio_total,
    SUM(CASE WHEN l.current_dpd > 15 THEN l.total_outstanding ELSE 0 END) AS overdue_15d,
    -- ... other aggregations
FROM officers o
LEFT JOIN loans l ON o.officer_id = l.officer_id
WHERE l.status = 'Active'
GROUP BY o.officer_id, o.officer_name, o.region, o.branch;

-- Refresh every 15 minutes
CREATE INDEX ON mv_officer_metrics_current(officer_id);
REFRESH MATERIALIZED VIEW CONCURRENTLY mv_officer_metrics_current;
```

#### **Query Optimization**
```sql
-- Use EXPLAIN ANALYZE to identify slow queries
EXPLAIN ANALYZE
SELECT officer_id, COUNT(*)
FROM loans
WHERE current_dpd BETWEEN 1 AND 6
GROUP BY officer_id;

-- Add covering indexes
CREATE INDEX idx_loans_dpd_officer_covering
ON loans(current_dpd, officer_id)
INCLUDE (total_outstanding, status);
```

---

### 8.2 Caching Strategy

#### **Multi-Layer Cache**
```
Request → Redis Cache → PostgreSQL → Response
           ↓ (miss)      ↓ (miss)
         Calculate    Store in DB
           ↓
      Store in Redis
```

#### **Cache Invalidation**
```javascript
// Invalidate cache when data changes
async function updateLoanStatus(loanId, newStatus) {
    // 1. Update database
    await db.updateLoan(loanId, { status: newStatus });

    // 2. Get affected officer
    const loan = await db.getLoan(loanId);

    // 3. Invalidate officer cache
    await redis.del(`officer_metrics:${loan.officerId}:*`);

    // 4. Invalidate branch cache
    await redis.del(`branch_metrics:${loan.branch}:*`);

    // 5. Trigger recalculation (async)
    await queue.add('recalculate-officer-metrics', {
        officerId: loan.officerId
    });
}
```

---

### 8.3 API Performance

#### **Pagination**
Always paginate large result sets.

```javascript
// Cursor-based pagination for better performance
GET /api/v1/loans/fimr-drilldown?cursor=LN12345&limit=50

// Response includes next cursor
{
  "data": [...],
  "meta": {
    "next_cursor": "LN12395",
    "has_more": true
  }
}
```

#### **Field Selection**
Allow clients to request only needed fields.

```javascript
GET /api/v1/metrics/officers?fields=officer_id,officer_name,risk_score,ayr
```

#### **Compression**
Enable gzip compression for API responses.

```go
// Go middleware
func GzipMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
            gz := gzip.NewWriter(w)
            defer gz.Close()
            w.Header().Set("Content-Encoding", "gzip")
            next.ServeHTTP(&gzipResponseWriter{Writer: gz, ResponseWriter: w}, r)
        } else {
            next.ServeHTTP(w, r)
        }
    })
}
```

---

## 9. Data Migration from Main Backend

### 9.1 Initial Data Load

**Step 1: Export from Main Backend**
```sql
-- Export loans data
COPY (
    SELECT
        loan_id, customer_id, customer_name, customer_phone,
        officer_id, officer_name, officer_phone,
        region, branch, state,
        loan_amount, disbursement_date, maturity_date,
        -- ... all other fields
    FROM main_loans
    WHERE status IN ('Active', 'Closed')
) TO '/tmp/loans_export.csv' WITH CSV HEADER;

-- Export repayments data
COPY (
    SELECT * FROM main_repayments
) TO '/tmp/repayments_export.csv' WITH CSV HEADER;
```

**Step 2: Import to Analytics Database**
```sql
-- Import loans
COPY loans FROM '/tmp/loans_export.csv' WITH CSV HEADER;

-- Import repayments
COPY repayments FROM '/tmp/repayments_export.csv' WITH CSV HEADER;
```

**Step 3: Calculate Initial Metrics**
```bash
# Run initial calculation
./worker full-refresh --date=2024-10-17
```

---

### 9.2 Ongoing Synchronization

**Option 1: Event-Driven (Recommended)**
```
Main Backend → Event Bus (Kafka/RabbitMQ) → Analytics Service
                    ↓
              Event Types:
              - loan.created
              - loan.updated
              - repayment.created
              - repayment.updated
```

**Option 2: CDC (Change Data Capture)**
```
Main Database → Debezium → Kafka → Analytics Service
                    ↓
              Captures all changes
              in real-time
```

**Option 3: Scheduled ETL**
```bash
# Every 15 minutes
*/15 * * * * /app/etl sync-from-main-backend
```

---

## 10. Deployment Architecture

### 10.1 Infrastructure

```
┌─────────────────────────────────────────────────────────┐
│                     Load Balancer                        │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│              API Servers (3+ instances)                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  API 1   │  │  API 2   │  │  API 3   │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│                    Redis Cluster                         │
│              (Cache + Session Store)                     │
└─────────────────────────────────────────────────────────┘
                          ↓
┌─────────────────────────────────────────────────────────┐
│              PostgreSQL (Primary + Replicas)             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │ Primary  │→ │ Replica1 │  │ Replica2 │              │
│  └──────────┘  └──────────┘  └──────────┘              │
└─────────────────────────────────────────────────────────┘
                          ↑
┌─────────────────────────────────────────────────────────┐
│              Background Workers (2+ instances)           │
│  ┌──────────┐  ┌──────────┐                             │
│  │ Worker 1 │  │ Worker 2 │                             │
│  └──────────┘  └──────────┘                             │
└─────────────────────────────────────────────────────────┘
                          ↑
┌─────────────────────────────────────────────────────────┐
│                   Message Queue                          │
│              (RabbitMQ / Kafka)                          │
└─────────────────────────────────────────────────────────┘
```

---

### 10.2 Docker Compose (Development)

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_DB: analytics
      POSTGRES_USER: analytics_user
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data

  api:
    build: .
    environment:
      DATABASE_URL: postgres://analytics_user:${DB_PASSWORD}@postgres:5432/analytics
      REDIS_URL: redis://redis:6379
      PORT: 8080
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis

  worker:
    build: .
    command: ./worker
    environment:
      DATABASE_URL: postgres://analytics_user:${DB_PASSWORD}@postgres:5432/analytics
      REDIS_URL: redis://redis:6379
    depends_on:
      - postgres
      - redis

volumes:
  postgres_data:
  redis_data:
```

---

### 10.3 Kubernetes (Production)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: analytics-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: analytics-api
  template:
    metadata:
      labels:
        app: analytics-api
    spec:
      containers:
      - name: api
        image: analytics-api:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: analytics-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: analytics-secrets
              key: redis-url
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: analytics-api
spec:
  selector:
    app: analytics-api
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

---

## 11. Monitoring and Observability

### 11.1 Metrics to Track

**Application Metrics:**
- API response times (p50, p95, p99)
- Request rate (requests/second)
- Error rate (%)
- Cache hit rate (%)
- Background job duration
- Database query performance

**Business Metrics:**
- Total loans processed
- Total officers tracked
- Metric calculation success rate
- Data freshness (time since last calculation)

**Infrastructure Metrics:**
- CPU usage
- Memory usage
- Database connections
- Redis memory usage
- Disk I/O

---

### 11.2 Logging

**Structured Logging (JSON)**
```json
{
  "timestamp": "2024-10-17T10:30:00Z",
  "level": "info",
  "service": "analytics-api",
  "endpoint": "/api/v1/metrics/officers",
  "method": "GET",
  "status": 200,
  "duration_ms": 45,
  "user_id": "USR123",
  "request_id": "req-abc-123"
}
```

---

### 11.3 Alerting

**Critical Alerts:**
- Metric calculation job failed
- Database connection pool exhausted
- API error rate > 5%
- Cache hit rate < 70%
- Data staleness > 1 hour

---

## 12. Security Considerations

### 12.1 Authentication & Authorization

**JWT-based Authentication:**
```javascript
// Middleware to verify JWT token
async function authenticate(req, res, next) {
    const token = req.headers.authorization?.split(' ')[1];
    if (!token) {
        return res.status(401).json({ error: 'Unauthorized' });
    }

    try {
        const decoded = jwt.verify(token, process.env.JWT_SECRET);
        req.user = decoded;
        next();
    } catch (error) {
        return res.status(401).json({ error: 'Invalid token' });
    }
}
```

**Role-Based Access Control (RBAC):**
```javascript
// Roles: admin, auditor, viewer
function authorize(roles) {
    return (req, res, next) => {
        if (!roles.includes(req.user.role)) {
            return res.status(403).json({ error: 'Forbidden' });
        }
        next();
    };
}

// Usage
app.put('/api/v1/audit/officers/:id/status',
    authenticate,
    authorize(['admin', 'auditor']),
    updateAuditStatus
);
```

---

### 12.2 Data Protection

**Encryption at Rest:**
- PostgreSQL: Enable transparent data encryption (TDE)
- Redis: Enable encryption for sensitive cached data

**Encryption in Transit:**
- Use TLS/SSL for all API endpoints
- Use SSL for database connections

**PII Protection:**
- Hash customer phone numbers in logs
- Mask sensitive data in API responses (for non-admin users)

---

## 13. Summary & Recommendations

### 13.1 Core Architecture

✅ **Separate Microservice**: Analytics service separate from main backend
✅ **Hybrid Sync**: Batch processing (15-30 min) + event-driven updates
✅ **Pre-Aggregation**: Store calculated metrics in `officer_metrics_daily` and `branch_metrics_daily`
✅ **Multi-Layer Cache**: Redis cache + PostgreSQL materialized views

---

### 13.2 Technology Stack

| Component | Recommendation | Alternative |
|-----------|---------------|-------------|
| **Backend Language** | **Go (Golang)** ⭐ | Node.js (TypeScript) |
| **Database** | **PostgreSQL 14+** | - |
| **Cache** | **Redis 7+** | - |
| **Message Queue** | Kafka / RabbitMQ | - |
| **Deployment** | Kubernetes | Docker Swarm |
| **Monitoring** | Prometheus + Grafana | Datadog |

---

### 13.3 Database Tables Summary

**Core Tables (4):**
1. `loans` - All loan data
2. `repayments` - All payment transactions
3. `officers` - Loan officer master data
4. `customers` - Customer master data

**Derived Tables (2):**
5. `officer_metrics_daily` - Pre-calculated officer metrics
6. `branch_metrics_daily` - Pre-calculated branch metrics

**Supporting Tables (6):**
7. `loan_schedule` - Payment schedule for each loan
8. `dpd_transitions` - DPD bucket transitions
9. `par15_snapshots` - PAR15 snapshots (mid-month)
10. `audit_tracking` - Audit assignments and status
11. `team_members` - Team members for assignments
12. `metric_calculation_log` - Calculation job logs

**Total: 12 tables**

---

### 13.4 API Endpoints Summary

**Officer Metrics (2):**
- `GET /api/v1/metrics/officers`
- `GET /api/v1/metrics/officers/:officer_id`

**Loan Drilldowns (2):**
- `GET /api/v1/loans/fimr-drilldown`
- `GET /api/v1/loans/early-indicators-drilldown`

**Branch Metrics (1):**
- `GET /api/v1/metrics/branches`

**Audit Management (4):**
- `GET /api/v1/audit/team-members`
- `PUT /api/v1/audit/officers/:officer_id/assignee`
- `PUT /api/v1/audit/officers/:officer_id/status`
- `POST /api/v1/audit/officers/:officer_id/actions`

**Filters (1):**
- `GET /api/v1/filters/options`

**Total: 10 endpoints**

---

### 13.5 Implementation Phases

**Phase 1: Foundation (Weeks 1-2)**
- Set up PostgreSQL database
- Create core tables (loans, repayments, officers, customers)
- Implement basic API endpoints
- Set up Redis cache

**Phase 2: Metrics Calculation (Weeks 3-4)**
- Implement metric calculation logic
- Create derived tables (officer_metrics_daily, branch_metrics_daily)
- Build background workers
- Set up scheduled jobs

**Phase 3: Dashboard Integration (Week 5)**
- Implement all API endpoints
- Connect frontend to backend
- Test all dashboard features
- Performance optimization

**Phase 4: Audit Features (Week 6)**
- Implement audit tracking
- Add team member management
- Build action recording
- Test audit workflows

**Phase 5: Production Deployment (Week 7-8)**
- Set up production infrastructure
- Configure monitoring and alerting
- Load testing and optimization
- Go live!

---

### 13.6 Performance Targets

| Metric | Target | Notes |
|--------|--------|-------|
| **API Response Time (p95)** | < 200ms | For pre-aggregated data |
| **API Response Time (p95)** | < 1s | For drilldown queries |
| **Cache Hit Rate** | > 80% | For officer/branch metrics |
| **Metric Calculation Time** | < 5 min | For all officers |
| **Data Freshness** | < 30 min | Time since last calculation |
| **Concurrent Users** | 100+ | Simultaneous dashboard users |

---

## 14. Next Steps

1. **Review & Approve**: Review this architecture with your team
2. **Prototype**: Build a small prototype with Go + PostgreSQL
3. **Data Mapping**: Map your existing database schema to proposed schema
4. **API Contract**: Finalize API endpoint specifications
5. **Development**: Start Phase 1 implementation
6. **Testing**: Set up automated testing (unit + integration)
7. **Deployment**: Deploy to staging environment
8. **Go Live**: Production deployment with monitoring

---

**Questions? Need clarification on any section?** 🚀

