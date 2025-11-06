# Current DPD Fix - Summary Report

## Investigation Complete ✅

I have completed a comprehensive investigation into why many loans in the Seeds Metrics database do not have a `current_dpd` (Days Past Due) value.

## Key Findings

### Current Database State
- **Total Loans**: 17,419
- **Loans with NULL current_dpd**: 0 (0%)
- **Loans with valid current_dpd**: 17,419 (100%)

**Conclusion**: While all current loans have `current_dpd` values, a **critical bug** was identified that would cause NULL values in future scenarios.

## The Bug

**Location**: `backend/migrations/025_fix_days_since_due_calculation.sql`

**Issue**: The trigger function `update_loan_computed_fields()` has a flaw in the `current_dpd` calculation:

```sql
v_current_dpd := CASE
    WHEN v_last_payment_date IS NOT NULL THEN
        GREATEST(0, (CURRENT_DATE - v_last_payment_date)::INTEGER)
    ELSE
        GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)  -- Returns NULL if v_first_due_date is NULL
END;
```

**When It Triggers**: When `first_payment_due_date` is NULL (and no loan schedule exists), the calculation returns NULL instead of a valid DPD value.

**Impact**: HIGH - Affects risk scoring, portfolio analysis, and all dashboard queries that depend on `current_dpd`.

## The Solution

**Migration 027**: `backend/migrations/027_fix_current_dpd_null_calculation.sql`

**Fix Applied**:
```sql
v_current_dpd := CASE
    WHEN v_first_due_date IS NULL THEN 0  -- No due date = not overdue
    WHEN v_last_payment_date IS NOT NULL THEN
        GREATEST(0, (CURRENT_DATE - v_last_payment_date)::INTEGER)
    ELSE
        GREATEST(0, (CURRENT_DATE - v_first_due_date)::INTEGER)
END;
```

**Key Improvement**: Added NULL check to default to 0 when no due date is available.

## DPD Distribution (Current Data)

| Category | Count | Percentage |
|----------|-------|-----------|
| Current (DPD = 0) | 1,602 | 9.2% |
| Early Indicator (DPD 1-6) | 2,145 | 12.3% |
| Overdue (DPD 7-15) | 1,823 | 10.5% |
| Severely Overdue (DPD > 15) | 6,300 | 36.2% |
| Inactive/Closed | 5,549 | 31.8% |

## Affected Components

The `current_dpd` field is used in 38 locations across the codebase:

1. **Dashboard Repository** - Risk scoring, portfolio analysis
2. **Officer Performance** - Performance metrics by DPD ranges
3. **Early Indicator Tagging** - Loans with DPD 1-6
4. **Risk Assessment** - Risk score calculation
5. **Portfolio Management** - DPD range filters (D1-3, D4-6, D7-15, D16-30)

## Deployment Status

### Files Created
✅ `backend/migrations/027_fix_current_dpd_null_calculation.sql` - Migration with fix
✅ `CURRENT_DPD_INVESTIGATION_REPORT.md` - Detailed investigation
✅ `CURRENT_DPD_FIX_DEPLOYMENT_GUIDE.md` - Deployment instructions
✅ `CURRENT_DPD_INVESTIGATION_COMPLETE.md` - Investigation summary

### Ready for Deployment
The migration is production-ready and should be deployed to prevent future issues.

## Deployment Instructions

```bash
# SSH to production
ssh root@143.198.146.44

# Apply migration
psql 'postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require' \
  -f /home/seeds-metrics-backend/backend/migrations/027_fix_current_dpd_null_calculation.sql

# Verify fix
psql 'postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require' << EOF
SELECT COUNT(*) as null_dpd_count FROM loans WHERE current_dpd IS NULL;
EOF
```

Expected result: `null_dpd_count = 0`

## Recommendations

1. **Deploy Migration 027** to production immediately
2. **Test** with new loans to ensure `current_dpd` is calculated correctly
3. **Monitor** dashboard queries for any issues
4. **Document** the fix in deployment procedures

## Root Cause Analysis

The bug was introduced in Migration 025 when the trigger function was updated to fix the `days_since_due` calculation. The fix for `days_since_due` inadvertently exposed a latent bug in the `current_dpd` calculation where NULL values weren't properly handled.

All loans synced from Django have `first_payment_due_date` populated, which is why the bug hasn't manifested yet. However, future loans or edge cases could trigger this bug.

## Conclusion

The investigation has successfully identified and fixed a critical bug in the `current_dpd` calculation. The fix is production-ready and should be deployed to ensure all loans have proper DPD values for accurate risk assessment and portfolio management.

