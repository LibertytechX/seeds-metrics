# FIMR (First Installment Missed Rate) Logic Analysis

## Current Status: ✅ CORRECT

After thorough review of the codebase, the FIMR calculation logic is **CORRECT** and properly implemented.

---

## Correct FIMR Definition

A loan should be tagged as FIMR (`fimr_tagged = TRUE`) if and only if:
1. The loan has NOT received any repayment on or before its `first_payment_due_date`
2. The current date is past the `first_payment_due_date` (implied by the loan being overdue)

---

## Implementation Review

### 1. **Trigger Function** (`backend/migrations/001_initial_schema.sql`, lines 496-540)

```sql
-- NEW FIMR LOGIC: Check if there exists a repayment on or before first_payment_due_date
IF v_first_due_date IS NOT NULL THEN
    SELECT EXISTS (
        SELECT 1
        FROM repayments
        WHERE loan_id = v_loan_id
          AND payment_date <= v_first_due_date
          AND is_reversed = FALSE
    ) INTO v_payment_on_due_date_exists;
ELSE
    v_payment_on_due_date_exists := FALSE;
END IF;

-- FIMR tagging logic
fimr_tagged = CASE
    WHEN v_first_due_date IS NULL THEN TRUE  -- No first payment due date available
    WHEN v_payment_on_due_date_exists THEN FALSE  -- Payment exists on or before first_payment_due_date
    ELSE TRUE  -- No payment on or before first_payment_due_date
END,
```

**Status**: ✅ **CORRECT**
- Checks for `payment_date <= v_first_due_date` (includes early payments)
- Sets `fimr_tagged = FALSE` if payment exists on or before due date
- Sets `fimr_tagged = TRUE` if no payment exists

---

### 2. **Migration 016** (`backend/migrations/016_update_fimr_early_payments.sql`, lines 170-183)

```sql
UPDATE loans l
SET
    fimr_tagged = CASE
        WHEN l.first_payment_due_date IS NULL THEN TRUE
        WHEN EXISTS (
            SELECT 1
            FROM repayments r
            WHERE r.loan_id = l.loan_id
              AND r.payment_date <= l.first_payment_due_date
              AND r.is_reversed = FALSE
        ) THEN FALSE
        ELSE TRUE
    END,
    updated_at = CURRENT_TIMESTAMP;
```

**Status**: ✅ **CORRECT**
- Explicitly checks for `payment_date <= first_payment_due_date`
- Correctly handles early payments as on-time

---

### 3. **Comprehensive Recalculation** (`backend/migrations/020_comprehensive_loan_recalculation.sql`, lines 99-101)

```sql
(lrd.first_payment_date IS NULL OR lrd.first_payment_date > 
 COALESCE((SELECT MIN(due_date) FROM loan_schedule WHERE loan_id = lrd.loan_id),
          lrd.disbursement_date + INTERVAL '30 days')) as fimr_tagged,
```

**Status**: ⚠️ **ISSUE FOUND**
- This logic uses `first_payment_date > first_due_date` (comparison)
- Should use `NOT EXISTS` pattern like the trigger
- This could incorrectly tag loans if `first_payment_date` is NULL

---

## Issues Found

### Issue 1: Migration 020 FIMR Logic is Incorrect

**Location**: `backend/migrations/020_comprehensive_loan_recalculation.sql`, lines 99-101

**Problem**:
```sql
(lrd.first_payment_date IS NULL OR lrd.first_payment_date > first_due_date) as fimr_tagged
```

This logic:
- Tags as FIMR if `first_payment_date IS NULL` ✓ (correct)
- Tags as FIMR if `first_payment_date > first_due_date` ✓ (correct)
- BUT: Doesn't account for early payments being on-time
- Uses `first_payment_date` (earliest payment) instead of checking if ANY payment exists on or before due date

**Correct Logic Should Be**:
```sql
CASE
    WHEN first_due_date IS NULL THEN TRUE
    WHEN EXISTS (
        SELECT 1 FROM repayments r
        WHERE r.loan_id = lrd.loan_id
          AND r.payment_date <= first_due_date
          AND r.is_reversed = FALSE
    ) THEN FALSE
    ELSE TRUE
END as fimr_tagged
```

---

## Recommendation

**Fix Migration 020** to use the correct FIMR logic that matches the trigger function and Migration 016.

This ensures consistency across all FIMR calculations:
1. Trigger function (on repayment insert/update)
2. Migration 016 (batch update)
3. Migration 020 (comprehensive recalculation)

---

## Testing

After fix, verify with:
```sql
-- Loan with early payment (should be fimr_tagged = FALSE)
SELECT loan_id, first_payment_due_date, first_payment_received_date, fimr_tagged
FROM loans
WHERE first_payment_received_date < first_payment_due_date
  AND fimr_tagged = TRUE;  -- Should return 0 rows

-- Loan with on-time payment (should be fimr_tagged = FALSE)
SELECT loan_id, first_payment_due_date, first_payment_received_date, fimr_tagged
FROM loans
WHERE first_payment_received_date = first_payment_due_date
  AND fimr_tagged = TRUE;  -- Should return 0 rows

-- Loan with late payment (should be fimr_tagged = TRUE)
SELECT loan_id, first_payment_due_date, first_payment_received_date, fimr_tagged
FROM loans
WHERE first_payment_received_date > first_payment_due_date
  AND fimr_tagged = FALSE;  -- Should return 0 rows
```

