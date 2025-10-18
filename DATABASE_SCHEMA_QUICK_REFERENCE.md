# Database Schema Quick Reference

## Overview
This document provides a quick reference for all database tables required for the Metrics Dashboard.

---

## Core Tables (Source of Truth)

### 1. `loans`
**Purpose**: Primary source for all loan-level data

**⚠️ CRITICAL:** Fields marked `[COMPUTED]` are calculated from repayments, NOT from ETL source.
This reduces compute load on the main business server.

**Key Fields (FROM ETL):**
- `loan_id` (PK)
- `customer_id`, `customer_name`, `customer_phone`
- `officer_id`, `officer_name`, `officer_phone`
- `region`, `branch`, `state`
- `loan_amount`, `disbursement_date`, `maturity_date`
- `status`, `channel`, `interest_rate`, `fee_amount`

**Key Fields [COMPUTED]:**
- `current_dpd`, `max_dpd_ever` (from loan_schedule)
- `first_payment_missed`, `first_payment_due_date`, `first_payment_received_date` (from repayments)
- `principal_outstanding`, `interest_outstanding`, `fees_outstanding`, `total_outstanding` (from repayments)
- `total_principal_paid`, `total_interest_paid`, `total_fees_paid` (from repayments)
- `fimr_tagged`, `early_indicator_tagged` (from DPD status)

**Indexes:**
- `officer_id`, `branch`, `region`, `disbursement_date`
- `current_dpd`, `fimr_tagged`, `early_indicator_tagged`

**Total:** 20 fields from ETL + 14 computed = 34 fields

---

### 2. `repayments`
**Purpose**: Tracks all payment transactions

**Key Fields:**
- `repayment_id` (PK)
- `loan_id` (FK)
- `payment_date`, `payment_amount`
- `principal_paid`, `interest_paid`, `fees_paid`
- `is_backdated`, `is_reversed`
- `waiver_amount`, `waiver_type`

**Indexes:**
- `loan_id`, `payment_date`
- `is_backdated`, `is_reversed`

---

### 3. `officers`
**Purpose**: Loan officer master data

**Key Fields:**
- `officer_id` (PK)
- `officer_name`, `officer_phone`, `officer_email`
- `region`, `branch`
- `employment_status`, `primary_channel`

**Indexes:**
- `branch`, `region`, `employment_status`

---

### 4. `customers`
**Purpose**: Customer master data

**Key Fields:**
- `customer_id` (PK)
- `customer_name`, `customer_phone`, `customer_email`
- `state`, `lga`, `address`
- `kyc_status`

**Indexes:**
- `customer_phone`, `state`

---

## Derived/Aggregated Tables (Performance)

### 5. `officer_metrics_daily`
**Purpose**: Pre-calculated daily metrics per officer

**Key Fields:**
- `officer_id`, `calculation_date`
- Portfolio: `total_portfolio`, `active_loans`, `overdue_15d_amount`
- FIMR: `first_miss_count`, `disbursed_count`, `fimr`
- Slippage: `dpd_1to6_balance`, `amount_due_7d`, `d06_slippage`
- Roll: `moved_to_7to30`, `prev_dpd_1to6_balance`, `roll`
- FRR: `fees_collected`, `fees_due`, `frr`
- AYR: `interest_collected`, `par15_mid_month`, `ayr`
- DQI: `risk_score_norm`, `on_time_rate`, `channel_purity`, `dqi`
- Risk: `risk_score`, `risk_band`

**Indexes:**
- `officer_id`, `calculation_date`
- Composite: `(officer_id, calculation_date)`

---

### 6. `branch_metrics_daily`
**Purpose**: Pre-calculated daily metrics per branch

**Key Fields:**
- `branch`, `region`, `calculation_date`
- `portfolio_total`, `active_loans`, `total_officers`
- `overdue_15d`, `par15_ratio`
- `ayr`, `dqi`, `fimr`

**Indexes:**
- `branch`, `region`, `calculation_date`

---

## Supporting Tables

### 7. `loan_schedule`
**Purpose**: Payment schedule for each loan (required for D0-6 Slippage)

**Key Fields:**
- `loan_id` (FK), `installment_number`
- `due_date`, `total_due`, `amount_paid`
- `payment_status` ('Pending', 'Paid', 'Partial', 'Overdue')

**Indexes:**
- `loan_id`, `due_date`, `payment_status`

---

### 8. `dpd_transitions`
**Purpose**: Track DPD bucket transitions (required for Roll calculation)

**Key Fields:**
- `loan_id`, `officer_id`, `transition_date`
- `from_dpd_bucket`, `to_dpd_bucket`
- `outstanding_balance`

**Indexes:**
- `officer_id`, `transition_date`
- `(from_dpd_bucket, to_dpd_bucket)`

---

### 9. `par15_snapshots`
**Purpose**: PAR15 snapshots at mid-month (required for AYR)

**Key Fields:**
- `officer_id`, `snapshot_date`
- `par15_exposure`, `par15_count`

**Indexes:**
- `officer_id`, `snapshot_date`

---

### 10. `audit_tracking`
**Purpose**: Audit assignments and status for officers

**Key Fields:**
- `officer_id`, `assignee_id`, `assignee_name`
- `audit_status` ('In Progress', 'Assigned', 'Resolved')
- `last_audit_date`, `audit_notes`
- `action_type`, `action_date`

**Indexes:**
- `officer_id`, `assignee_id`, `audit_status`

---

### 11. `team_members`
**Purpose**: Team members who can be assigned audits

**Key Fields:**
- `member_id` (PK)
- `member_name`, `member_email`, `role`
- `is_active`

**Indexes:**
- `is_active`, `role`

---

### 12. `metric_calculation_log`
**Purpose**: Tracks metric calculation jobs (monitoring)

**Key Fields:**
- `calculation_type`, `calculation_date`
- `start_time`, `end_time`, `duration_seconds`
- `records_processed`, `status`

**Indexes:**
- `calculation_type`, `calculation_date`, `status`

---

## Metric Calculation Formulas

### FIMR (First-Installment Miss Rate)
```
FIMR = first_miss_count / disbursed_count
```
- `first_miss_count`: COUNT of loans where `first_payment_missed = TRUE`
- `disbursed_count`: Total loans disbursed in period

---

### D0-6 Slippage (Early Slippage)
```
D0-6 Slippage = dpd_1to6_balance / amount_due_7d
```
- `dpd_1to6_balance`: SUM of outstanding for loans with DPD 1-6
- `amount_due_7d`: SUM of payments due in next 7 days

---

### Roll (0-6 → 7-30)
```
Roll = moved_to_7to30 / prev_dpd_1to6_balance
```
- `moved_to_7to30`: Amount moved from DPD 1-6 to 7-30
- `prev_dpd_1to6_balance`: DPD 1-6 balance from previous period

---

### FRR (Fees Realization Rate)
```
FRR = fees_collected / fees_due
```
- `fees_collected`: SUM of fees_paid in period
- `fees_due`: SUM of fee_amount for active loans

---

### AYR (Adjusted Yield Ratio)
```
AYR = (interest_collected + fees_collected) / par15_mid_month
```
- Numerator: Interest + Fees collected in current month
- Denominator: PAR15 exposure at mid-month (15th)

---

### DQI (Delinquency Quality Index)
```
DQI = 100 * (0.4 * RQ + 0.35 * OTI + 0.25 * (1 - FIMR)) * CP
```
- `RQ`: Risk Quality (normalized risk score)
- `OTI`: On-Time Rate
- `FIMR`: First-Installment Miss Rate
- `CP`: Channel Purity (optional multiplier)

---

### Risk Score
```
Risk Score = 100 - penalties
```
Penalties:
- 20 * PORR
- 15 * FIMR
- 10 * Roll
- 10 * (waivers / amount_due_7d)
- 10 * (backdated / entries)
- 10 * (reversals / entries)
- 10 * (1 - FRR)
- 5 * (1 - channel_purity)
- 10 * (had_float_gap ? 1 : 0)

**Risk Bands:**
- Green: ≥ 80
- Watch: 60-79
- Amber: 40-59
- Red/Flag: < 40

---

## Data Relationships

```
customers (1) ──→ (N) loans (N) ──→ (1) officers
                      ↓
                  (1 to N)
                      ↓
                 repayments
                      ↓
                  (1 to N)
                      ↓
               loan_schedule

officers (1) ──→ (N) officer_metrics_daily
         (1) ──→ (1) audit_tracking

branch ──→ (N) officers
       ──→ (1) branch_metrics_daily
```

---

## Table Sizes (Estimated)

| Table | Estimated Rows | Growth Rate |
|-------|---------------|-------------|
| `loans` | 1M - 10M | 10K/day |
| `repayments` | 5M - 50M | 50K/day |
| `officers` | 100 - 1,000 | Slow |
| `customers` | 500K - 5M | 5K/day |
| `officer_metrics_daily` | 100K - 1M | 100/day |
| `branch_metrics_daily` | 10K - 100K | 25/day |
| `loan_schedule` | 10M - 100M | 100K/day |
| `dpd_transitions` | 1M - 10M | 10K/day |
| `par15_snapshots` | 10K - 100K | 100/month |
| `audit_tracking` | 1K - 10K | Slow |

---

## Partitioning Strategy

**Partition by Date:**
- `loans`: Partition by `disbursement_date` (monthly)
- `repayments`: Partition by `payment_date` (monthly)
- `loan_schedule`: Partition by `due_date` (monthly)

**Benefits:**
- Faster queries (scan only relevant partitions)
- Easier archival (drop old partitions)
- Better maintenance (vacuum/analyze per partition)

---

## Backup Strategy

**Daily Backups:**
- Full backup of all tables
- Retention: 30 days

**Point-in-Time Recovery:**
- WAL archiving enabled
- Recovery window: 7 days

**Critical Tables (Hourly Snapshots):**
- `officer_metrics_daily`
- `branch_metrics_daily`

---

## Migration Checklist

- [ ] Create all 12 tables
- [ ] Add all indexes
- [ ] Set up foreign key constraints
- [ ] Configure partitioning for large tables
- [ ] Import initial data from main backend
- [ ] Run initial metric calculations
- [ ] Verify data integrity
- [ ] Set up backup jobs
- [ ] Configure monitoring
- [ ] Test all API endpoints

---

**Total Tables: 12**
**Total Indexes: ~50**
**Estimated Database Size: 50GB - 500GB** (depending on loan volume)

