# Loan Metrics Reference Guide

## Complete List of Loan Metrics

This document provides a comprehensive reference for all 56 fields in the `loans` table.

---

## 1. Basic Loan Information (14 fields)

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `loan_id` | VARCHAR | Django | Unique loan identifier |
| `customer_id` | VARCHAR | Django | Customer identifier |
| `customer_name` | VARCHAR | Django | Customer full name |
| `customer_phone` | VARCHAR | Django | Customer phone number |
| `officer_id` | VARCHAR | Django | Loan officer identifier |
| `officer_name` | VARCHAR | Django | Loan officer name |
| `officer_phone` | VARCHAR | Django | Loan officer phone |
| `region` | VARCHAR | Django | Geographic region |
| `branch` | VARCHAR | Django | Branch location |
| `state` | VARCHAR | Django | State/province |
| `channel` | VARCHAR | Django | Disbursement channel |
| `channel_partner` | VARCHAR | Django | Channel partner name |
| `wave` | VARCHAR | Django | Loan wave/batch |
| `loan_type` | VARCHAR | Django | Type of loan product |

---

## 2. Loan Terms (6 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `loan_amount` | DECIMAL(15,2) | Django | Principal amount disbursed | From Django |
| `disbursement_date` | DATE | Django | Date loan was disbursed | From Django |
| `maturity_date` | DATE | Django | Expected loan end date | From Django |
| `loan_term_days` | INTEGER | Django | Total loan duration in days | From Django |
| `interest_rate` | DECIMAL(5,4) | Django | Interest rate (e.g., 0.2800 = 28%) | From Django |
| `fee_amount` | DECIMAL(15,2) | Django | Total fees charged | From Django |

---

## 3. Loan Status (4 fields)

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `status` | VARCHAR | Django | Current loan status (ACTIVE, CLOSED, CANCELLED, etc.) |
| `closed_date` | DATE | Django | Date loan was closed |
| `verification_status` | VARCHAR | Django | Verification status |
| `performance_status` | VARCHAR | Computed | Performance classification |

---

## 4. Repayment Totals (4 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `total_principal_paid` | DECIMAL(15,2) | Computed | Total principal repaid | `SUM(repayments.principal_paid)` |
| `total_interest_paid` | DECIMAL(15,2) | Computed | Total interest repaid | `SUM(repayments.interest_paid)` |
| `total_fees_paid` | DECIMAL(15,2) | Computed | Total fees repaid | `SUM(repayments.fees_paid)` |
| `total_repayments` | DECIMAL(15,2) | Computed | Total amount repaid | `SUM(repayments.payment_amount)` |

---

## 5. Outstanding Balances (5 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `principal_outstanding` | DECIMAL(15,2) | Computed | Remaining principal | `loan_amount - total_principal_paid` |
| `interest_outstanding` | DECIMAL(15,2) | Computed | Remaining interest | `(loan_amount × interest_rate) - total_interest_paid` |
| `fees_outstanding` | DECIMAL(15,2) | Computed | Remaining fees | `fee_amount - total_fees_paid` |
| `total_outstanding` | DECIMAL(15,2) | Computed | Total remaining balance | `principal_outstanding + interest_outstanding + fees_outstanding` |
| `actual_outstanding` | DECIMAL(15,2) | Computed | Time-based overdue amount | `(daily_repayment_amount × repayment_days_due_today) - total_repayments` |

**Key Difference:**
- `total_outstanding` = What is left to pay (total remaining balance)
- `actual_outstanding` = What should have been paid by now minus what was paid (overdue amount)

---

## 6. Payment Dates (3 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `first_payment_due_date` | DATE | Django/Computed | When first payment was due | Synced from Django `start_date`, fallback: `disbursement_date + 30 days` |
| `first_payment_received_date` | DATE | Computed | When first payment was received | `MIN(repayments.payment_date)` |
| `days_since_last_repayment` | INTEGER | Computed | Days since last payment | `CURRENT_DATE - MAX(repayments.payment_date)` |

---

## 7. DPD (Days Past Due) Metrics (3 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `current_dpd` | INTEGER | Computed | Current days past due | `MAX(0, repayment_days_due_today - repayment_days_paid)` if `actual_outstanding > 0`, else `0` |
| `max_dpd_ever` | INTEGER | Computed | Highest DPD ever recorded | `MAX(repayments.dpd_at_payment, current_dpd)` |
| `days_since_due` | INTEGER | Computed | Days since first payment was due | `CURRENT_DATE - first_payment_due_date` |

---

## 8. Business Days Calculations (5 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `daily_repayment_amount` | DECIMAL(15,2) | Computed | Expected daily repayment | `repayment_amount / loan_term_days` |
| `real_loan_tenure_days` | INTEGER | Computed | Business days from disbursement to maturity | `count_business_days(disbursement_date, maturity_date)` |
| `repayment_days_paid` | DECIMAL(15,2) | Computed | Equivalent days paid | `total_repayments / daily_repayment_amount` |
| `repayment_days_due_today` | INTEGER | Computed | Business days due as of today | `count_business_days(first_payment_due_date, MIN(CURRENT_DATE, maturity_date))` |
| `business_days_since_disbursement` | INTEGER | Computed | Business days since disbursement | `count_business_days(disbursement_date, CURRENT_DATE)` |

**Note:** Business days exclude weekends (Saturday and Sunday).

---

## 9. Loan Age and Timing (2 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `loan_age` | INTEGER | Computed | Calendar days since disbursement | `CURRENT_DATE - disbursement_date` |
| `repayment_amount` | DECIMAL(15,2) | Computed | Total amount to be repaid | `loan_amount × (1 + interest_rate) + fee_amount` |

---

## 10. Risk Indicators (3 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `fimr_tagged` | BOOLEAN | Computed | First Installment Missed Repayment | `TRUE` if no payment on/before `first_payment_due_date` AND due date has passed |
| `early_indicator_tagged` | BOOLEAN | Computed | Early warning indicator | `current_dpd BETWEEN 1 AND 6` |
| `first_payment_missed` | BOOLEAN | Computed | Whether first payment was late | `first_payment_received_date > first_payment_due_date` OR `first_payment_received_date IS NULL` |

---

## 11. Health Metrics (3 fields)

| Field | Type | Source | Description | Formula |
|-------|------|--------|-------------|---------|
| `repayment_delay_rate` | DECIMAL(5,2) | Computed | Repayment frequency score (can be negative) | `(1 - (((days_since_last_repayment + current_dpd) / 2) / loan_age) / 0.25) × 100` |
| `repayment_health` | DECIMAL(5,2) | Computed | Overall repayment health score (0-100) | Complex formula with DPD penalties and progress penalties |
| `timeliness_score` | DECIMAL(5,2) | Computed | Payment timeliness score | Based on `days_since_last_repayment / loan_age` ratio |

**Repayment Delay Rate Interpretation:**
- `100`: Excellent repayment frequency
- `0-100`: Good to fair repayment behavior
- `< 0`: Poor repayment behavior (delays exceed 25% of loan age)

---

## 12. Organizational Fields (2 fields)

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `vertical_lead_name` | VARCHAR | Django | Vertical lead name |
| `vertical_lead_email` | VARCHAR | Django | Vertical lead email |

---

## 13. Metadata (2 fields)

| Field | Type | Source | Description |
|-------|------|--------|-------------|
| `created_at` | TIMESTAMP | System | When record was created |
| `updated_at` | TIMESTAMP | System | When record was last updated |

---

## Total: 56 Fields

### Breakdown by Source:
- **Django (Source of Truth)**: 24 fields
- **Computed (Trigger Function)**: 30 fields
- **System Metadata**: 2 fields

---

## Critical Formulas

### 1. Actual Outstanding (Overdue Amount)
```sql
actual_outstanding = GREATEST(0,
    (daily_repayment_amount × repayment_days_due_today) - total_repayments
)
```

### 2. Current DPD
```sql
current_dpd = CASE
    WHEN actual_outstanding <= 0 THEN 0
    ELSE GREATEST(0, repayment_days_due_today - repayment_days_paid::INTEGER)
END
```

### 3. Repayment Delay Rate
```sql
repayment_delay_rate = CASE
    WHEN loan_age > 0 AND days_since_last_repayment IS NOT NULL THEN
        (1.0 - ((((days_since_last_repayment + current_dpd) / 2.0) / loan_age) / 0.25)) × 100
    WHEN loan_age > 0 AND days_since_last_repayment IS NULL THEN
        (1.0 - ((current_dpd / loan_age) / 0.25)) × 100
    WHEN loan_age = 0 THEN 0
    ELSE NULL
END
```

### 4. FIMR Tagged
```sql
fimr_tagged = CASE
    WHEN first_payment_due_date IS NULL THEN TRUE
    WHEN payment_on_due_date_exists THEN FALSE
    WHEN first_payment_date IS NULL AND first_payment_due_date >= CURRENT_DATE THEN FALSE
    ELSE TRUE
END
```

---

## Data Flow

```
Django Database (savings)
    ↓
    ├─ loans_ajoloan → loans (basic fields)
    ├─ loans_ajorepayment → repayments
    └─ Sync Scripts (update_first_payment_due_date.sql)
    ↓
SeedsMetrics Database (seedsmetrics)
    ↓
Trigger Function (update_loan_computed_fields)
    ↓
Computed Metrics Updated
```

---

## Test Coverage

The validation test suite (`test_loan_metrics_validation.py`) validates:

✓ All 6 basic loan fields  
✓ All 4 repayment totals  
✓ All 4 outstanding balances  
✓ All 5 business days calculations  
✓ All 3 DPD metrics  
✓ Actual outstanding calculation  
✓ Repayment delay rate formula  
✓ All 3 risk indicators (FIMR, early indicator, first payment missed)  

**Total: ~24 validations per loan**

