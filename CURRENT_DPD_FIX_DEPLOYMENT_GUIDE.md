# Current DPD Fix - Deployment Guide

## Overview

Migration 027 has been created to fix a critical bug in the `current_dpd` (Days Past Due) calculation where the field becomes NULL when `first_payment_due_date` is NULL.

## The Bug

**Location**: `backend/migrations/025_fix_days_since_due_calculation.sql` - Trigger function `update_loan_computed_fields()`

**Problem**: 
```sql
v_current_dpd := CASE
    WHEN v_last_payment_date IS NOT NULL THEN
        GREATEST(0, (CURRENT_DATE - v_last_payment_date)::INTEGER)
    ELSE
        GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)  -- BUG: Returns NULL if v_first_due_date is NULL
END;
```

When `v_first_due_date` is NULL (which happens when both `loan_schedule` and `first_payment_due_date` are NULL), the ELSE clause evaluates to NULL, resulting in `current_dpd = NULL`.

**Impact**: HIGH
- New loans without due dates will have NULL `current_dpd`
- All dependent queries and calculations will fail
- Risk scoring and portfolio analysis will be inaccurate

## The Fix

**Location**: `backend/migrations/027_fix_current_dpd_null_calculation.sql`

**Solution**:
```sql
v_current_dpd := CASE
    WHEN v_first_due_date IS NULL THEN 0  -- No due date = not overdue
    WHEN v_last_payment_date IS NOT NULL THEN
        GREATEST(0, (CURRENT_DATE - v_last_payment_date)::INTEGER)
    ELSE
        GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)
END;
```

Added NULL check to default to 0 when no due date is available.

## Deployment Steps

### Step 1: Apply the Migration

```bash
# SSH to production server
ssh root@143.198.146.44

# Apply the migration
psql 'postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require' \
  -f /home/seeds-metrics-backend/backend/migrations/027_fix_current_dpd_null_calculation.sql
```

### Step 2: Verify the Fix

```bash
# Check for NULL current_dpd values
psql 'postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require' << EOF
SELECT 
    COUNT(*) as total_loans,
    COUNT(CASE WHEN current_dpd IS NULL THEN 1 END) as null_dpd_count,
    COUNT(CASE WHEN current_dpd IS NOT NULL THEN 1 END) as valid_dpd_count
FROM loans;
EOF
```

Expected output: `null_dpd_count = 0`

### Step 3: Verify Distribution

```bash
psql 'postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require' << EOF
SELECT 
    CASE 
        WHEN current_dpd IS NULL THEN 'NULL'
        WHEN current_dpd = 0 THEN '0 (Current)'
        WHEN current_dpd BETWEEN 1 AND 6 THEN '1-6 (Early Indicator)'
        WHEN current_dpd BETWEEN 7 AND 15 THEN '7-15 (Overdue)'
        WHEN current_dpd > 15 THEN '>15 (Severely Overdue)'
    END as dpd_category,
    COUNT(*) as count
FROM loans
GROUP BY dpd_category
ORDER BY count DESC;
EOF
```

### Step 4: Test with New Repayment

Create a test repayment to ensure the trigger works correctly:

```bash
# The trigger will automatically recalculate current_dpd for the affected loan
# Verify the calculation is correct
psql 'postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require' << EOF
SELECT 
    loan_id,
    customer_name,
    current_dpd,
    days_since_due,
    first_payment_due_date,
    first_payment_received_date
FROM loans
WHERE current_dpd IS NOT NULL
ORDER BY current_dpd DESC
LIMIT 20;
EOF
```

## Files Modified

- **backend/migrations/027_fix_current_dpd_null_calculation.sql** - New migration file with the fix

## Testing Recommendations

1. **Unit Test**: Verify trigger function handles NULL `first_payment_due_date`
2. **Integration Test**: Create a loan without `first_payment_due_date` and verify `current_dpd = 0`
3. **Regression Test**: Verify existing loans still have correct `current_dpd` values
4. **API Test**: Verify dashboard queries return correct DPD values

## Rollback Plan

If issues occur, the previous trigger function can be restored from Migration 025:

```bash
psql 'postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require' \
  -f /home/seeds-metrics-backend/backend/migrations/025_fix_days_since_due_calculation.sql
```

## Current Status

- ✅ Migration file created: `backend/migrations/027_fix_current_dpd_null_calculation.sql`
- ✅ Investigation completed: Root cause identified
- ⏳ Deployment: Ready for production deployment
- ⏳ Verification: Pending post-deployment verification

## Next Steps

1. Deploy migration to production
2. Verify all loans have non-NULL `current_dpd` values
3. Monitor dashboard queries for any issues
4. Update documentation with deployment confirmation

