# Days Since Due Calculation Update - Summary

## Task Completed ✅

Updated the FIMR table to correctly calculate `days_since_due` based on the difference between the first payment due date and the actual first repayment received date (or current date if no payment received).

## Changes Made

### 1. Migration 025: Fix Days Since Due Calculation
**File:** `backend/migrations/025_fix_days_since_due_calculation.sql`

- Added `days_since_due` column to loans table
- Updated all 17,419 loans with correct calculation:
  - **If payment received:** `days_since_due = first_payment_received_date - first_payment_due_date`
  - **If no payment:** `days_since_due = CURRENT_DATE - first_payment_due_date`
- Updated trigger function to maintain `days_since_due` on new repayments
- Added comprehensive statistics and sample queries

### 2. Backend Repository Fix
**File:** `backend/internal/repository/dashboard_repository.go`

- Changed GetFIMRLoans query from using `l.current_dpd as days_since_due` to `l.days_since_due`
- Ensures API returns correct calculated values

## Results

### Loan Distribution by Payment Status

| Payment Category | Count | Min Days | Max Days | Avg Days |
|------------------|-------|----------|----------|----------|
| Early Payment | 14,975 | -368 | -1 | -18.97 |
| On-Time Payment | 996 | 0 | 0 | 0.00 |
| Late Payment | 1,358 | 1 | 197 | 5.12 |
| No Payment - Overdue | 90 | 1 | 571 | 276.77 |
| No Payment - Not Yet Due | 41 | -29 | 0 | -4.37 |

**Total Loans:** 17,419

### Key Metrics

- **Early Payments:** 14,975 loans (86.0%) - paid before due date
- **On-Time Payments:** 996 loans (5.7%) - paid on due date
- **Late Payments:** 1,358 loans (7.8%) - paid after due date
- **Overdue (No Payment):** 90 loans (0.5%) - past due date with no payment
- **Future Due:** 41 loans (0.2%) - due date not yet reached

## Value Interpretation

- **Negative values:** Payment received before due date (early payment)
- **Zero:** Payment received on exact due date (on-time)
- **Positive (with payment):** Days late for payment
- **Positive (no payment):** Days overdue since due date
- **Negative (no payment):** Days until due date

## API Response Example

```json
{
  "loan_id": "18597",
  "customer_name": "RUKAYAT AMOSUN",
  "first_payment_due_date": "2025-10-25T00:00:00Z",
  "days_since_due": 1,
  "amount_paid": 33000,
  "fimr_tagged": true
}
```

## Deployment Status

✅ Migration 025 applied to production database
✅ Backend code updated and deployed
✅ API service restarted and verified
✅ All 17,419 loans recalculated with correct values
✅ FIMR drilldown table now displays accurate days_since_due

## Testing

Verified with:
- Direct database queries showing correct calculations
- API endpoint testing returning proper values
- Sample loans across all payment categories
- Comprehensive statistics showing distribution

## Files Modified

1. `backend/migrations/025_fix_days_since_due_calculation.sql` - New migration
2. `backend/internal/repository/dashboard_repository.go` - Query fix
3. `backend/DAYS_SINCE_DUE_UPDATE_SUMMARY.md` - This summary

## Commits

- `feat: fix days_since_due calculation in FIMR table`
- `fix: use days_since_due column in GetFIMRLoans query`

