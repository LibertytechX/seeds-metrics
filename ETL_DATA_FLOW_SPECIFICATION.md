# ETL Data Flow Specification

## 🎯 Critical Design Decision

**The analytics service computes all derived fields from repayments data.**

This design decision **reduces compute load on the main business server** by offloading all calculations to the analytics database. The main backend only sends raw transactional data.

---

## 📊 Data Sources and Flow

### Main Business Backend → Analytics Database

```
Main Business Backend (Source of Truth)
    ↓
ETL Worker (Every 15-30 minutes)
    ↓
Analytics Database (PostgreSQL)
    ↓
Computed Fields Trigger (Automatic)
    ↓
Updated Loan Records with Calculated Fields
```

---

## 📋 Table-by-Table Data Source Specification

### 1. **`loans` Table**

#### **FROM ETL SOURCE** (Main Backend sends these)

| Field | Type | Description | Source |
|-------|------|-------------|--------|
| `loan_id` | VARCHAR(50) | Primary key | Main DB |
| `customer_id` | VARCHAR(50) | Customer reference | Main DB |
| `customer_name` | VARCHAR(255) | Customer name | Main DB |
| `customer_phone` | VARCHAR(20) | Customer phone | Main DB |
| `officer_id` | VARCHAR(50) | Loan officer ID | Main DB |
| `officer_name` | VARCHAR(255) | Officer name | Main DB |
| `officer_phone` | VARCHAR(20) | Officer phone | Main DB |
| `region` | VARCHAR(100) | Geographic region | Main DB |
| `branch` | VARCHAR(100) | Branch name | Main DB |
| `state` | VARCHAR(100) | State/province | Main DB |
| `loan_amount` | DECIMAL(15,2) | Principal amount | Main DB |
| `disbursement_date` | DATE | Disbursement date | Main DB |
| `maturity_date` | DATE | Maturity date | Main DB |
| `loan_term_days` | INTEGER | Loan term in days | Main DB |
| `interest_rate` | DECIMAL(5,4) | Interest rate | Main DB |
| `fee_amount` | DECIMAL(15,2) | Fee amount | Main DB |
| `channel` | VARCHAR(50) | Distribution channel | Main DB |
| `channel_partner` | VARCHAR(100) | Partner name | Main DB |
| `status` | VARCHAR(50) | Loan status | Main DB |
| `closed_date` | DATE | Closure date | Main DB |

**Total: 20 fields from ETL source**

---

#### **[COMPUTED]** (Analytics service calculates these)

| Field | Type | Calculation Source | Trigger |
|-------|------|-------------------|---------|
| `current_dpd` | INTEGER | `loan_schedule` (unpaid installments) | After repayment/schedule update |
| `max_dpd_ever` | INTEGER | `MAX(current_dpd)` over time | After repayment/schedule update |
| `first_payment_missed` | BOOLEAN | `first_payment_date > first_due_date` | After repayment insert |
| `first_payment_due_date` | DATE | `MIN(due_date)` from `loan_schedule` | After schedule insert |
| `first_payment_received_date` | DATE | `MIN(payment_date)` from `repayments` | After repayment insert |
| `principal_outstanding` | DECIMAL(15,2) | `loan_amount - SUM(principal_paid)` | After repayment insert |
| `interest_outstanding` | DECIMAL(15,2) | `total_interest - SUM(interest_paid)` | After repayment insert |
| `fees_outstanding` | DECIMAL(15,2) | `fee_amount - SUM(fees_paid)` | After repayment insert |
| `total_outstanding` | DECIMAL(15,2) | Sum of all outstanding | After repayment insert |
| `total_principal_paid` | DECIMAL(15,2) | `SUM(principal_paid)` from `repayments` | After repayment insert |
| `total_interest_paid` | DECIMAL(15,2) | `SUM(interest_paid)` from `repayments` | After repayment insert |
| `total_fees_paid` | DECIMAL(15,2) | `SUM(fees_paid)` from `repayments` | After repayment insert |
| `fimr_tagged` | BOOLEAN | `first_payment_missed = TRUE` | After repayment insert |
| `early_indicator_tagged` | BOOLEAN | `current_dpd BETWEEN 1 AND 6` | After repayment/schedule update |

**Total: 14 computed fields**

---

### 2. **`repayments` Table**

#### **FROM ETL SOURCE** (Main Backend sends ALL fields)

| Field | Type | Description | Source |
|-------|------|-------------|--------|
| `repayment_id` | VARCHAR(50) | Primary key | Main DB |
| `loan_id` | VARCHAR(50) | Loan reference | Main DB |
| `payment_date` | DATE | Payment date | Main DB |
| `payment_amount` | DECIMAL(15,2) | Total payment | Main DB |
| `principal_paid` | DECIMAL(15,2) | Principal portion | Main DB |
| `interest_paid` | DECIMAL(15,2) | Interest portion | Main DB |
| `fees_paid` | DECIMAL(15,2) | Fees portion | Main DB |
| `penalty_paid` | DECIMAL(15,2) | Penalty portion | Main DB |
| `payment_method` | VARCHAR(50) | Payment method | Main DB |
| `payment_reference` | VARCHAR(100) | Payment reference | Main DB |
| `payment_channel` | VARCHAR(50) | Payment channel | Main DB |
| `dpd_at_payment` | INTEGER | DPD at payment time | Main DB |
| `is_backdated` | BOOLEAN | Backdated flag | Main DB |
| `is_reversed` | BOOLEAN | Reversal flag | Main DB |
| `reversal_date` | DATE | Reversal date | Main DB |
| `reversal_reason` | TEXT | Reversal reason | Main DB |
| `waiver_amount` | DECIMAL(15,2) | Waiver amount | Main DB |
| `waiver_type` | VARCHAR(50) | Waiver type | Main DB |
| `waiver_approved_by` | VARCHAR(100) | Approver | Main DB |

**Total: 19 fields - ALL from ETL source**

**No computed fields** - This is the source data for computing loan fields.

---

### 3. **`loan_schedule` Table**

#### **FROM ETL SOURCE** (Main Backend sends these)

| Field | Type | Description | Source |
|-------|------|-------------|--------|
| `loan_id` | VARCHAR(50) | Loan reference | Main DB |
| `installment_number` | INTEGER | Installment number | Main DB |
| `due_date` | DATE | Due date | Main DB |
| `principal_due` | DECIMAL(15,2) | Principal due | Main DB |
| `interest_due` | DECIMAL(15,2) | Interest due | Main DB |
| `fee_due` | DECIMAL(15,2) | Fee due | Main DB |
| `total_due` | DECIMAL(15,2) | Total due | Main DB |

**Total: 7 fields from ETL source**

---

#### **[COMPUTED]** (Analytics service calculates these)

| Field | Type | Calculation Source | Trigger |
|-------|------|-------------------|---------|
| `payment_status` | VARCHAR(50) | Based on `amount_paid` vs `total_due` | After repayment insert |
| `amount_paid` | DECIMAL(15,2) | `SUM(repayments)` for this installment | After repayment insert |
| `payment_date` | DATE | Date when fully paid | After repayment insert |

**Total: 3 computed fields**

---

### 4. **`officers` Table**

#### **FROM ETL SOURCE** (Main Backend sends ALL fields)

All fields come from the main backend. No computed fields.

---

### 5. **`customers` Table**

#### **FROM ETL SOURCE** (Main Backend sends ALL fields)

All fields come from the main backend. No computed fields.

---

## 🔄 Computation Triggers

### Trigger 1: `update_loan_computed_fields()`

**Fires when:**
- New repayment inserted
- Repayment updated
- Loan schedule updated

**Updates:**
- All 14 computed fields in `loans` table

**Logic:**
```sql
1. SUM all repayments (excluding reversed) → total_*_paid
2. Calculate outstanding = original - paid
3. Get first payment date from repayments
4. Get first due date from loan_schedule
5. Compare dates → first_payment_missed
6. Calculate current_dpd from unpaid installments
7. Update max_dpd_ever
8. Set fimr_tagged and early_indicator_tagged
```

---

### Trigger 2: `update_schedule_payment_status()`

**Fires when:**
- New repayment inserted
- Repayment updated

**Updates:**
- `payment_status`, `amount_paid`, `payment_date` in `loan_schedule`

**Logic:**
```sql
1. SUM repayments for this installment
2. Compare to total_due
3. Set status: 'Paid', 'Partial', 'Pending', 'Overdue'
```

---

## 📥 ETL Data Format

### Expected JSON Format from Main Backend

#### Loans ETL Payload
```json
{
  "loans": [
    {
      "loan_id": "LN001",
      "customer_id": "CUST001",
      "customer_name": "John Doe",
      "customer_phone": "+234-803-123-4567",
      "officer_id": "OFF001",
      "officer_name": "Sarah Johnson",
      "officer_phone": "+234-803-987-6543",
      "region": "South West",
      "branch": "Lagos Main",
      "state": "Lagos",
      "loan_amount": 500000.00,
      "disbursement_date": "2024-01-15",
      "maturity_date": "2024-07-15",
      "loan_term_days": 180,
      "interest_rate": 0.15,
      "fee_amount": 25000.00,
      "channel": "Direct",
      "channel_partner": null,
      "status": "Active",
      "closed_date": null
    }
  ]
}
```

**Note:** No DPD, outstanding balances, or payment totals included!

---

#### Repayments ETL Payload
```json
{
  "repayments": [
    {
      "repayment_id": "REP001",
      "loan_id": "LN001",
      "payment_date": "2024-02-01",
      "payment_amount": 100000.00,
      "principal_paid": 80000.00,
      "interest_paid": 15000.00,
      "fees_paid": 5000.00,
      "penalty_paid": 0.00,
      "payment_method": "Bank Transfer",
      "payment_reference": "TXN123456",
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
  ]
}
```

---

#### Loan Schedule ETL Payload
```json
{
  "loan_schedules": [
    {
      "loan_id": "LN001",
      "installment_number": 1,
      "due_date": "2024-02-01",
      "principal_due": 83333.33,
      "interest_due": 12500.00,
      "fee_due": 4166.67,
      "total_due": 100000.00
    }
  ]
}
```

**Note:** No payment_status, amount_paid, or payment_date included!

---

## 🔧 ETL Worker Implementation

### Pseudo-code for ETL Process

```javascript
async function syncFromMainBackend() {
    // 1. Fetch data from main backend
    const loans = await mainBackendAPI.getLoans({ since: lastSyncTime });
    const repayments = await mainBackendAPI.getRepayments({ since: lastSyncTime });
    const schedules = await mainBackendAPI.getLoanSchedules({ since: lastSyncTime });
    
    // 2. Insert/Update loans (only ETL fields)
    for (const loan of loans) {
        await analyticsDB.upsert('loans', {
            loan_id: loan.loan_id,
            customer_id: loan.customer_id,
            customer_name: loan.customer_name,
            // ... only the 20 ETL fields
            // DO NOT include computed fields
        });
    }
    
    // 3. Insert/Update loan schedules
    for (const schedule of schedules) {
        await analyticsDB.upsert('loan_schedule', schedule);
        // Trigger will fire and update loan.current_dpd
    }
    
    // 4. Insert/Update repayments
    for (const repayment of repayments) {
        await analyticsDB.upsert('repayments', repayment);
        // Trigger will fire and update all computed fields in loans table
    }
    
    // 5. Update last sync time
    await analyticsDB.updateSyncLog({
        sync_time: new Date(),
        loans_synced: loans.length,
        repayments_synced: repayments.length
    });
}
```

---

## ✅ Benefits of This Approach

### 1. **Reduced Load on Main Backend**
- Main backend only sends raw transactional data
- No complex aggregations or calculations
- Faster API responses

### 2. **Centralized Calculation Logic**
- All metric calculations in one place (analytics service)
- Easier to maintain and update formulas
- Consistent calculations across all metrics

### 3. **Real-time Updates**
- Triggers automatically update computed fields
- No manual recalculation needed
- Always up-to-date data

### 4. **Audit Trail**
- All raw data preserved in repayments table
- Can recalculate historical metrics
- Easy to debug discrepancies

### 5. **Performance**
- Computed fields stored in database (fast queries)
- No on-the-fly calculations for dashboards
- Indexed for optimal performance

---

## 📊 Data Flow Summary

```
Main Backend
    ↓ (20 fields)
loans table (ETL fields only)
    ↓
    ↓ (Trigger watches)
    ↓
repayments table (19 fields)
    ↓ (Trigger fires)
    ↓
loans table (14 computed fields updated)
    ↓
Dashboard API (reads complete loan data)
```

---

## 🚨 Important Notes

1. **Never send computed fields from main backend** - They will be overwritten by triggers
2. **Repayments are the source of truth** - All calculations derive from this table
3. **Triggers run automatically** - No manual intervention needed
4. **Batch updates supported** - Triggers handle bulk inserts efficiently
5. **Reversals handled** - `is_reversed = TRUE` excludes payments from calculations

---

## 🔍 Verification Queries

### Check if computed fields are correct:

```sql
-- Verify total_principal_paid
SELECT 
    l.loan_id,
    l.total_principal_paid as stored_value,
    COALESCE(SUM(r.principal_paid), 0) as calculated_value,
    l.total_principal_paid - COALESCE(SUM(r.principal_paid), 0) as difference
FROM loans l
LEFT JOIN repayments r ON l.loan_id = r.loan_id AND r.is_reversed = FALSE
GROUP BY l.loan_id, l.total_principal_paid
HAVING ABS(l.total_principal_paid - COALESCE(SUM(r.principal_paid), 0)) > 0.01;
```

---

**Total Fields Summary:**
- **Loans**: 20 from ETL + 14 computed = 34 total
- **Repayments**: 19 from ETL + 0 computed = 19 total
- **Loan Schedule**: 7 from ETL + 3 computed = 10 total

**Main Backend Sends**: 46 fields total  
**Analytics Service Computes**: 17 fields total  
**Total Data Points**: 63 fields

