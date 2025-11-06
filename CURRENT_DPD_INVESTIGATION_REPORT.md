# Current DPD Investigation Report

## Executive Summary

Investigation into the `current_dpd` (Days Past Due) field in the Seeds Metrics database has revealed a **critical bug in the trigger function** that calculates this field. While the initial query showed all loans have `current_dpd` values, the underlying issue is that the calculation logic is **fundamentally broken** when `first_payment_due_date` is NULL.

## Issue Identified

### Root Cause

The `update_loan_computed_fields()` trigger function (used in Migration 025) has a critical flaw in the `current_dpd` calculation:

```sql
v_current_dpd := CASE
    WHEN v_last_payment_date IS NOT NULL THEN
        GREATEST(0, (CURRENT_DATE - v_last_payment_date)::INTEGER)
    ELSE
        GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)  -- BUG: NULL if v_first_due_date is NULL
END;
```

**Problem**: When `v_first_due_date` is NULL (which happens when both `loan_schedule` and `first_payment_due_date` are NULL), the ELSE clause evaluates to NULL, resulting in `current_dpd = NULL`.

### Impact

1. **Loans without `first_payment_due_date`** will have NULL `current_dpd` values
2. **New loans** that haven't had their first payment due date calculated may have NULL values
3. **Loans without loan_schedule entries** may have NULL values
4. **Risk calculations** that depend on `current_dpd` will be inaccurate

### Current Data Status

- **Total Loans**: 17,419
- **Loans with current_dpd**: 17,419 (100%)
- **Loans with NULL current_dpd**: 0 (0%)

**Note**: The current database shows all loans have values, but this is likely because:
1. All loans have been synced from Django with `first_payment_due_date` populated
2. The trigger hasn't been tested with new loans that might not have this field

## Trigger Function Analysis

### Current Logic (Migration 025)

**Lines 210-216** of the trigger function:

```sql
-- Calculate current DPD
v_current_dpd := CASE
    WHEN v_last_payment_date IS NOT NULL THEN
        GREATEST(0, (CURRENT_DATE - v_last_payment_date)::INTEGER)
    ELSE
        GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)
END;
```

### Issues

1. **No NULL handling**: If `v_first_due_date` is NULL, the result is NULL
2. **Incomplete logic**: Doesn't account for loans without payment history or due dates
3. **Inconsistent with days_since_due**: The `days_since_due` calculation (lines 201-208) has better NULL handling

### Correct Logic Should Be

```sql
v_current_dpd := CASE
    WHEN v_first_due_date IS NULL THEN 0  -- No due date = not overdue
    WHEN v_last_payment_date IS NOT NULL THEN
        GREATEST(0, (CURRENT_DATE - v_last_payment_date)::INTEGER)
    ELSE
        GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)
END;
```

## Recommended Fix

### Migration 027: Fix current_dpd NULL Calculation

Create a new migration that:

1. **Updates the trigger function** to handle NULL `first_payment_due_date`
2. **Backfills any NULL values** in existing loans
3. **Adds validation** to ensure `current_dpd` is never NULL for active loans

### Key Changes

- Add NULL check for `v_first_due_date` before calculation
- Default to 0 when no due date is available
- Ensure consistency with `days_since_due` logic
- Add comments explaining the calculation

## Testing Strategy

1. **Verify current state**: Confirm all loans have non-NULL `current_dpd`
2. **Test with new loans**: Create test loans without `first_payment_due_date`
3. **Verify trigger fires**: Ensure trigger updates `current_dpd` on repayment insert
4. **Check edge cases**: Test with NULL disbursement_date, NULL first_payment_date, etc.

## Impact Assessment

### Affected Queries

The following queries/features depend on `current_dpd`:

1. **Dashboard metrics**: `avg_days_past_due`, `early_rot_count`, `late_rot_count`
2. **Loan filtering**: DPD range filters (D1-3, D4-6, D7-15, D16-30)
3. **Risk scoring**: Risk score calculation uses `current_dpd` heavily
4. **Portfolio analysis**: PAR (Portfolio at Risk) calculations
5. **Officer performance**: DPD-based metrics for officers

### Risk Level

**HIGH** - If new loans are created without `first_payment_due_date`, they will have NULL `current_dpd` values, breaking all dependent calculations.

## Recommendations

1. **Immediate**: Apply Migration 027 to fix the trigger function
2. **Short-term**: Add database constraints to ensure `current_dpd` is never NULL
3. **Medium-term**: Add monitoring to detect NULL `current_dpd` values
4. **Long-term**: Refactor DPD calculation to be more robust and testable

## Next Steps

1. Create Migration 027 with the fix
2. Test the fix in development
3. Deploy to production
4. Verify all loans have proper `current_dpd` values
5. Monitor for any issues

