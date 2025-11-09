# Repayments Data Source and AYR Calculation Documentation

## Executive Summary

This document explains the complete data pipeline for repayment data used in the AYR (Annualized Yield Rate) calculation, including the source database, table schemas, sync process, and the critical finding about how fees and interest are calculated.

---

## 🔴 CRITICAL FINDING: Interest and Fees Are NOT Tracked in Source Data

**The Django source database does NOT track interest_paid and fees_paid separately.**

Instead, the Seeds Metrics platform uses **PROPORTIONAL ALLOCATION** to estimate these values from the total repayment amount based on the loan's original terms.

### Current State of Repayments Table

| Metric | Value |
|--------|-------|
| **Total Repayments** | 464,966 |
| **Unique Loans with Repayments** | 17,288 |
| **Reversed Repayments** | 0 |
| **Valid Repayments** | 464,966 |
| **Total Payment Amount** | ₦3,620,977,450.68 |
| **Total Principal Paid** | ₦3,620,977,450.68 |
| **Total Interest Paid** | ₦0.00 |
| **Total Fees Paid** | ₦0.00 |

**Observation**: 
- ✅ `payment_amount` is populated for all 464,966 repayments
- ✅ `principal_paid` is populated (equals `payment_amount`)
- ❌ `interest_paid` is **ALWAYS 0** (not tracked)
- ❌ `fees_paid` is **ALWAYS 0** (not tracked)

---

## 1. Database Tables

### 1.1 Source Database (Django)

**Database**: `savings` (PostgreSQL at 164.90.155.2:5432)  
**Table**: `loans_ajoloanrepayment`  
**Purpose**: Stores all payment transactions from borrowers

**Schema** (24 columns):

| Column Name | Data Type | Description |
|-------------|-----------|-------------|
| `id` | bigint | Primary key (repayment ID) |
| `ajo_loan_id` | bigint | Foreign key to loans table |
| `paid_date` | timestamp with time zone | Payment timestamp |
| `repayment_amount` | double precision | **Total payment amount** |
| `health_insurance` | double precision | Health insurance fee (proxy for fees) |
| `repayment_type` | varchar | Payment method (CASH, TRANSFER, AGENT_DEBIT, etc.) |
| `loan_amount` | double precision | Loan amount at time of payment |
| `repayment_ref` | uuid | Payment reference UUID |
| `agent_id` | bigint | Agent processing the payment |
| `borrower_id` | bigint | Borrower making the payment |
| `marked_as` | varchar | Status flag (BACKDATED, REVERSAL, etc.) |
| `applied_to_loan` | boolean | Whether applied to loan balance |
| `comment` | varchar | Notes/comments |
| `settlement_status` | boolean | Settlement status |
| `created_at` | timestamp | Record creation timestamp |
| `updated_at` | timestamp | Last modification timestamp |
| ... | ... | (18 more columns) |

**Key Observations**:
- ❌ **NO `interest_paid` column** in Django source
- ❌ **NO `fees_paid` column** in Django source
- ✅ Only `repayment_amount` (total payment) is tracked
- ✅ `health_insurance` is tracked (used as proxy for fees in some cases)

**Sample Data**:
```
id     | ajo_loan_id | paid_date                      | repayment_amount | health_insurance | repayment_type
-------|-------------|--------------------------------|------------------|------------------|---------------
479289 | 17533       | 2025-11-06 21:02:03.482787+00 | 4600             | 46.52            | AGENT_DEBIT
479288 | 12959       | 2025-11-06 21:01:44.599359+00 | 100              | 0                | AGENT_DEBIT
479287 | 14492       | 2025-11-06 21:01:43.964685+00 | 9000             | 0                | AGENT_DEBIT
```

---

### 1.2 Target Database (Seeds Metrics)

**Database**: `seedsmetrics` (PostgreSQL at private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060)  
**Table**: `repayments`  
**Purpose**: Stores synced repayment data with additional analytics fields

**Schema** (21 columns):

| Column Name | Data Type | Default | Description |
|-------------|-----------|---------|-------------|
| `repayment_id` | varchar(50) | - | Primary key (from Django `id`) |
| `loan_id` | varchar(50) | - | Foreign key to loans table |
| `payment_date` | date | - | Payment date (from Django `paid_date`) |
| `payment_amount` | numeric(15,2) | - | **Total payment amount** |
| `principal_paid` | numeric(15,2) | 0 | **Calculated/estimated** |
| `interest_paid` | numeric(15,2) | 0 | **NOT POPULATED** |
| `fees_paid` | numeric(15,2) | 0 | **NOT POPULATED** |
| `penalty_paid` | numeric(15,2) | 0 | Not tracked |
| `payment_method` | varchar(50) | - | Payment method |
| `payment_reference` | varchar(100) | - | Payment reference |
| `payment_channel` | varchar(50) | - | Payment channel |
| `dpd_at_payment` | integer | 0 | Days past due at payment |
| `is_backdated` | boolean | false | Whether payment is backdated |
| `is_reversed` | boolean | false | Whether payment is reversed |
| `reversal_date` | date | - | Reversal date |
| `reversal_reason` | text | - | Reversal reason |
| `waiver_amount` | numeric(15,2) | 0 | Waiver amount |
| `waiver_type` | varchar(50) | - | Waiver type |
| `waiver_approved_by` | varchar(100) | - | Waiver approver |
| `created_at` | timestamp | CURRENT_TIMESTAMP | Record creation |
| `updated_at` | timestamp | CURRENT_TIMESTAMP | Last update |

**Sample Data**:
```
repayment_id | loan_id | payment_date | payment_amount | principal_paid | interest_paid | fees_paid | payment_method | is_reversed
-------------|---------|--------------|----------------|----------------|---------------|-----------|----------------|------------
476506       | 17174   | 2025-11-05   | 5000.00        | 5000.00        | 0.00          | 0.00      | AGENT_DEBIT    | f
476745       | 14213   | 2025-11-05   | 30000.00       | 30000.00       | 0.00          | 0.00      | ESCROW_DEBIT   | f
476844       | 16680   | 2025-11-05   | 6200.00        | 6200.00        | 0.00          | 0.00      | AGENT_DEBIT    | f
```

---

## 2. Data Sync Process

### 2.1 Sync Script

**File**: `backend/scripts/sync_repayments_incremental.go`  
**Purpose**: Syncs repayment data from Django to Seeds Metrics

**Key Steps**:

1. **Fetch repayments from Django** (via `DjangoRepository.GetRepayments()`)
2. **Transform data** to Seeds Metrics schema
3. **Insert/update** in Seeds Metrics `repayments` table

### 2.2 Django Repository Query

<augment_code_snippet path="backend/internal/repository/django_repository.go" mode="EXCERPT">
````go
// GetRepayments retrieves repayments from Django database
func (r *DjangoRepository) GetRepayments(ctx context.Context, limit, offset int) ([]map[string]interface{}, error) {
    query := `
        SELECT
            r.id::VARCHAR(50) as repayment_id,
            r.ajo_loan_id::VARCHAR(50) as loan_id,
            r.paid_date as payment_date,
            r.repayment_amount as payment_amount,
            COALESCE(r.repayment_type, 'TRANSFER') as payment_method,
            r.created_at,
            r.updated_at
        FROM loans_ajoloanrepayment r
        WHERE r.paid_date IS NOT NULL
        ORDER BY r.paid_date DESC
        LIMIT $1 OFFSET $2
    `
    // ... (rest of the method)
}
````
</augment_code_snippet>

**Observations**:
- ✅ Fetches `repayment_amount` (total payment)
- ❌ Does NOT fetch `interest_paid` or `fees_paid` (they don't exist in Django)
- ❌ Does NOT fetch `health_insurance` (could be used as proxy for fees)

### 2.3 Sync Transformation

<augment_code_snippet path="backend/scripts/sync_repayments_incremental.go" mode="EXCERPT">
````go
input := &models.RepaymentInput{
    RepaymentID:   repaymentID,
    LoanID:        loanID,
    PaymentDate:   paymentDate,
    PaymentAmount: decimal.NewFromFloat(paymentAmount),
    PrincipalPaid: decimal.NewFromFloat(paymentAmount),  // ⚠️ Assumes all payment is principal
    InterestPaid:  decimal.Zero,                         // ❌ Always 0
    FeesPaid:      decimal.Zero,                         // ❌ Always 0
    PenaltyPaid:   decimal.Zero,                         // ❌ Always 0
    PaymentMethod: paymentMethod,
    DPDAtPayment:  0,
    IsBackdated:   false,
    IsReversed:    false,
    WaiverAmount:  decimal.Zero,
}
````
</augment_code_snippet>

**Critical Issue**: 
- The sync script sets `principal_paid = payment_amount`
- It sets `interest_paid = 0` and `fees_paid = 0`
- This means the `repayments` table does NOT contain the actual breakdown of payments

---

## 3. How AYR Calculation Works (Given This Limitation)

Since the `repayments` table does NOT have `interest_paid` and `fees_paid`, the AYR calculation uses **PROPORTIONAL ALLOCATION** based on the loan's original terms.

### 3.1 Proportional Allocation Formula

**Step 1**: Calculate total expected amount for each loan:
```
Total Expected = Loan Amount × (1 + Interest Rate) + Fee Amount
```

**Step 2**: Calculate the proportion of each component:
```
Interest Proportion = (Loan Amount × Interest Rate) / Total Expected
Fees Proportion = Fee Amount / Total Expected
```

**Step 3**: Allocate total repayments proportionally:
```
Interest Collected = Total Repayments × Interest Proportion
Fees Collected = Total Repayments × Fees Proportion
```

### 3.2 SQL Implementation

<augment_code_snippet path="backend/internal/repository/dashboard_repository.go" mode="EXCERPT">
````sql
WITH loan_repayments AS (
    SELECT
        l.loan_id,
        l.officer_id,
        l.loan_amount,
        l.interest_rate,
        l.fee_amount,
        SUM(r.payment_amount) as total_repayments  -- ⚠️ Uses payment_amount, NOT interest_paid/fees_paid
    FROM loans l
    LEFT JOIN repayments r ON l.loan_id = r.loan_id AND r.is_reversed = false
    GROUP BY l.loan_id, l.officer_id, l.loan_amount, l.interest_rate, l.fee_amount
)
SELECT
    -- Fees collected (proportional allocation)
    COALESCE(SUM(
        CASE
            WHEN lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount > 0 THEN
                lr.total_repayments * lr.fee_amount / (lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount)
            ELSE 0
        END
    ), 0) as fees_collected,
    
    -- Interest collected (proportional allocation)
    COALESCE(SUM(
        CASE
            WHEN lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount > 0 THEN
                lr.total_repayments * (lr.loan_amount * lr.interest_rate) / (lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount)
            ELSE 0
        END
    ), 0) as interest_collected
FROM loan_repayments lr
````
</augment_code_snippet>

### 3.3 Example Calculation

**Loan Details** (Loan ID 1095, Officer 1053):
- Loan Amount: ₦100,000
- Interest Rate: 30% (0.30)
- Fee Amount: ₦2,000
- Total Repayments: ₦130,000

**Step 1**: Calculate total expected:
```
Total Expected = ₦100,000 × (1 + 0.30) + ₦2,000
               = ₦100,000 × 1.30 + ₦2,000
               = ₦130,000 + ₦2,000
               = ₦132,000
```

**Step 2**: Calculate proportions:
```
Interest Proportion = (₦100,000 × 0.30) / ₦132,000 = ₦30,000 / ₦132,000 = 0.2273
Fees Proportion = ₦2,000 / ₦132,000 = 0.0152
```

**Step 3**: Allocate repayments:
```
Interest Collected = ₦130,000 × 0.2273 = ₦29,545.45
Fees Collected = ₦130,000 × 0.0152 = ₦1,969.70
```

**Verification**:
```sql
-- Actual query result for Officer 1053
fees_collected: ₦181,482.34
interest_collected: ₦2,027,730.03
```

---

## 4. Data Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ DJANGO DATABASE (Source of Truth)                              │
│ Host: 164.90.155.2:5432                                         │
│ Database: savings                                               │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ Sync Script
                              │ (sync_repayments_incremental.go)
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ SEEDS METRICS DATABASE (Analytics)                             │
│ Host: private-generaldb-do-user-9489371-0.k.db.ondigitalocean  │
│ Database: seedsmetrics                                          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ SQL Query (GetOfficers)
                              │ Uses proportional allocation
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ RAW METRICS (RawMetrics struct)                                │
│ - fees_collected (calculated)                                   │
│ - interest_collected (calculated)                               │
│ - par15_mid_month (sum of principal_outstanding)                │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ MetricsService.CalculateOfficerMetrics()
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│ CALCULATED METRICS (CalculatedMetrics struct)                  │
│ - AYR = (interest_collected + fees_collected) / par15_mid_month│
└─────────────────────────────────────────────────────────────────┘
```

---

## 5. Key Findings

### ✅ What Works

1. **Repayment data is synced** from Django to Seeds Metrics
2. **Total payment amounts are accurate** (₦3.6B across 464,966 repayments)
3. **Proportional allocation provides reasonable estimates** for interest and fees
4. **AYR calculation is mathematically sound** given the available data

### ❌ Limitations

1. **Interest and fees are NOT tracked separately** in the source database
2. **Proportional allocation is an ESTIMATE**, not actual breakdown
3. **Assumes borrowers pay proportionally** (may not reflect reality)
4. **Cannot distinguish** between:
   - Borrower who paid 100% interest + 0% principal
   - Borrower who paid 50% interest + 50% principal
5. **health_insurance field in Django is NOT used** (could be used as proxy for fees)

### ⚠️ Implications for AYR

1. **AYR is based on ESTIMATED interest/fees**, not actual collections
2. **If borrowers pay principal first** (common in microfinance), AYR will be **OVERSTATED**
3. **If borrowers pay interest first**, AYR will be **UNDERSTATED**
4. **The calculation assumes** borrowers pay proportionally across all components

---

## 6. Recommendations

### Short-term (No Code Changes)

1. **Document the limitation** in user-facing reports
2. **Add disclaimer** that AYR is based on proportional allocation, not actual breakdown
3. **Validate the assumption** by sampling a few loans and checking if proportional allocation is reasonable

### Medium-term (Minor Code Changes)

1. **Use `health_insurance` field** from Django as proxy for fees collected
2. **Update sync script** to populate `fees_paid` from `health_insurance`
3. **Recalculate interest_paid** as `payment_amount - principal_paid - fees_paid`

### Long-term (Django Schema Changes)

1. **Add `interest_paid` and `fees_paid` columns** to Django `loans_ajoloanrepayment` table
2. **Update Django application** to track these values when processing payments
3. **Backfill historical data** using proportional allocation or manual reconciliation
4. **Update sync script** to use actual values instead of estimates

---

## 7. Summary

**Repayment Data Source**:
- ✅ Django database (`loans_ajoloanrepayment` table)
- ✅ 464,966 repayments totaling ₦3.6B
- ❌ Does NOT track interest_paid and fees_paid separately

**AYR Calculation Method**:
- Uses **proportional allocation** to estimate interest and fees from total repayments
- Based on loan's original terms (loan_amount, interest_rate, fee_amount)
- Assumes borrowers pay proportionally across all components

**Data Quality**:
- ✅ Total payment amounts are accurate
- ⚠️ Interest and fees are ESTIMATES, not actual values
- ⚠️ May not reflect actual payment behavior (e.g., principal-first vs interest-first)

**Next Steps**:
1. Validate proportional allocation assumption with sample data
2. Consider using `health_insurance` field as proxy for fees
3. Long-term: Update Django schema to track interest/fees separately

