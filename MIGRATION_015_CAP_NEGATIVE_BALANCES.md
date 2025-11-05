# Migration 015: Cap Negative Outstanding Balances

## Overview

This migration addresses the issue of negative portfolio totals caused by over-payment scenarios. It updates the loan outstanding balance calculation logic to cap negative values at zero.

## Problem Statement

### Issue
Officer `adeyinka232803@gmail.com` has a negative portfolio total showing in the Agent Performance table.

### Root Cause
When `total_principal_paid > loan_amount` (over-payment scenario), the calculation:
```sql
principal_outstanding = loan_amount - total_principal_paid
```
Results in a negative value, which contributes negatively to the officer's portfolio total.

### Impact
- Negative portfolio totals in officer metrics
- Incorrect dashboard displays
- Misleading performance indicators
- Potential issues with portfolio-level calculations

## Solution

### Business Logic Decision
Treat over-payments as fully paid (0 outstanding) rather than showing negative balances. This aligns with the business requirement that:
- A loan cannot have less than 0 outstanding
- Over-payments should show as 0 outstanding, not negative
- Portfolio totals should never be negative due to over-payments

### Technical Implementation

#### Before (Old Logic)
```sql
principal_outstanding = loan_amount - total_principal_paid
interest_outstanding = (loan_amount * interest_rate * loan_term_days / 365) - total_interest_paid
fees_outstanding = fee_amount - total_fees_paid
total_outstanding = principal_outstanding + interest_outstanding + fees_outstanding
```

**Problem**: If any component is over-paid, it becomes negative.

#### After (New Logic)
```sql
principal_outstanding = GREATEST(0, loan_amount - total_principal_paid)
interest_outstanding = GREATEST(0, (loan_amount * interest_rate * loan_term_days / 365) - total_interest_paid)
fees_outstanding = GREATEST(0, COALESCE(fee_amount, 0) - total_fees_paid)
total_outstanding = GREATEST(0, principal_outstanding + interest_outstanding + fees_outstanding)
actual_outstanding = GREATEST(0, actual_outstanding)
```

**Solution**: `GREATEST(0, value)` ensures the value is never negative.

## Changes Made

### 1. Updated Trigger Function
**File**: `backend/migrations/015_cap_negative_outstanding_balances.sql`

**Changes**:
- Added variables for capped outstanding balances
- Calculate each component with `GREATEST(0, ...)` to cap at zero
- Updated the UPDATE statement to use capped values

### 2. Recalculated Existing Loans
The migration includes a one-time UPDATE to fix all existing loans with negative balances:

```sql
UPDATE loans
SET
    principal_outstanding = GREATEST(0, loan_amount - total_principal_paid),
    interest_outstanding = GREATEST(0, (loan_amount * interest_rate * loan_term_days / 365) - total_interest_paid),
    fees_outstanding = GREATEST(0, COALESCE(fee_amount, 0) - total_fees_paid),
    total_outstanding = GREATEST(0, ...),
    actual_outstanding = GREATEST(0, actual_outstanding)
WHERE 
    principal_outstanding < 0 
    OR interest_outstanding < 0 
    OR fees_outstanding < 0 
    OR total_outstanding < 0
    OR actual_outstanding < 0;
```

### 3. Verification Queries
The migration includes verification queries to:
- Check for remaining negative balances (should be 0)
- Verify officer `adeyinka232803@gmail.com` portfolio total
- Summarize over-paid loans

## How to Apply

### Prerequisites
1. Database access credentials
2. PostgreSQL client (`psql`) installed
3. Backup of current database state

### Steps

#### Option 1: Using the Shell Script (Recommended)

```bash
# Navigate to migrations directory
cd backend/migrations

# Make script executable
chmod +x apply_cap_negative_balances.sh

# Set database password
export DB_PASSWORD="your_password"

# Run the script
./apply_cap_negative_balances.sh
```

The script will:
1. Create a backup of loans with negative balances
2. Show current state (before fix)
3. Apply the migration
4. Show results (after fix)
5. Verify no negative balances remain

#### Option 2: Manual Application

```bash
# Connect to database
psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
     -p 25060 \
     -U metricsuser \
     -d seedsmetrics

# Run migration
\i backend/migrations/015_cap_negative_outstanding_balances.sql
```

## Verification

### 1. Check Officer Portfolio Total

```sql
SELECT 
    COUNT(*) as total_loans,
    SUM(principal_outstanding) as portfolio_total,
    SUM(CASE WHEN principal_outstanding < 0 THEN 1 ELSE 0 END) as negative_loans
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';
```

**Expected Result**:
- `portfolio_total` should be >= 0 (not negative)
- `negative_loans` should be 0

### 2. Check for Any Negative Balances

```sql
SELECT COUNT(*) as negative_balance_count
FROM loans
WHERE principal_outstanding < 0 
   OR interest_outstanding < 0 
   OR fees_outstanding < 0 
   OR total_outstanding < 0
   OR actual_outstanding < 0;
```

**Expected Result**: `negative_balance_count` should be 0

### 3. Check Dashboard

1. Navigate to the Agent Performance table
2. Find officer `adeyinka232803@gmail.com`
3. Verify `Portfolio Total` is no longer negative
4. Verify other metrics are still correct

### 4. Check Over-Paid Loans

```sql
SELECT 
    loan_id,
    customer_name,
    loan_amount,
    total_principal_paid,
    principal_outstanding,
    (total_principal_paid - loan_amount) as overpayment
FROM loans
WHERE total_principal_paid > loan_amount
LIMIT 10;
```

**Expected Result**: 
- `principal_outstanding` should be 0 for all over-paid loans
- `overpayment` shows how much was over-paid

## Impact Analysis

### Affected Components

1. **Database Trigger**: `update_loan_computed_fields()`
   - Now caps all outstanding balances at 0
   - Runs automatically on every repayment

2. **Loan Records**: All loans with negative balances
   - Updated to show 0 instead of negative values
   - Historical payment data unchanged

3. **Officer Metrics**: Portfolio total calculation
   - Now correctly sums only non-negative values
   - Fixes negative portfolio totals

4. **Dashboard**: Agent Performance table
   - Will show correct portfolio totals
   - No code changes needed (uses database values)

### Data Integrity

**What Changed**:
- Outstanding balance values for over-paid loans
- Portfolio total calculations

**What Didn't Change**:
- Loan amounts
- Repayment records
- Payment history
- Total amounts paid
- Any other loan data

**Reversibility**:
- The migration can be reversed by removing `GREATEST(0, ...)` wrappers
- Backup file contains original negative values
- However, reversal is not recommended as it reintroduces the bug

## Business Implications

### Positive Impacts
1. **Accurate Metrics**: Portfolio totals now correctly reflect outstanding amounts
2. **Better Visibility**: Officers with over-paid loans show 0 outstanding, not negative
3. **Correct Reporting**: Dashboard and reports show accurate data
4. **Prevents Confusion**: Negative balances were confusing and misleading

### Considerations
1. **Over-Payment Tracking**: Over-paid amounts are still tracked in `total_principal_paid`
2. **Refund Process**: If refunds are needed, they should be processed separately
3. **Audit Trail**: Backup file shows which loans were over-paid
4. **Future Prevention**: Consider adding validation to prevent over-payments

## Monitoring

### Post-Migration Checks

1. **Daily Check** (first week):
   ```sql
   SELECT COUNT(*) FROM loans WHERE principal_outstanding < 0;
   ```
   Should always return 0.

2. **Weekly Check**:
   ```sql
   SELECT 
       COUNT(*) as overpaid_loans,
       SUM(total_principal_paid - loan_amount) as total_overpayment
   FROM loans
   WHERE total_principal_paid > loan_amount;
   ```
   Monitor for increasing over-payments.

3. **Officer Portfolio Check**:
   ```sql
   SELECT 
       officer_id,
       SUM(principal_outstanding) as portfolio_total
   FROM loans
   GROUP BY officer_id
   HAVING SUM(principal_outstanding) < 0;
   ```
   Should return 0 rows.

## Rollback Plan

If issues are discovered:

1. **Stop New Repayments**: Temporarily disable repayment entry
2. **Restore Trigger**: Revert to previous trigger version
3. **Restore Data**: Use backup to restore original values
4. **Investigate**: Determine root cause of issues
5. **Re-apply**: Fix issues and re-apply migration

**Rollback Script**:
```sql
-- Restore previous trigger (from migration 012)
\i backend/migrations/012_update_fimr_4day_grace_period.sql

-- Restore data from backup
-- (Manual process using backup CSV file)
```

## Future Enhancements

### Recommended Improvements

1. **Validation**: Add constraint to prevent over-payments
   ```sql
   ALTER TABLE repayments ADD CONSTRAINT check_no_overpayment
   CHECK (principal_paid <= (
       SELECT loan_amount - COALESCE(total_principal_paid, 0)
       FROM loans WHERE loan_id = repayments.loan_id
   ));
   ```

2. **Alerts**: Add monitoring for over-payment scenarios
3. **Reporting**: Create report of over-paid loans for review
4. **Process**: Define business process for handling over-payments

## Support

### Common Issues

**Q: Why are some loans showing 0 outstanding when they were over-paid?**
A: This is expected behavior. Over-paid loans now show 0 outstanding instead of negative values.

**Q: How can I see which loans were over-paid?**
A: Query loans where `total_principal_paid > loan_amount`

**Q: Will this affect historical reports?**
A: No. Historical payment data is unchanged. Only the calculated outstanding balances are updated.

**Q: What if we need to refund an over-payment?**
A: Process refunds separately. The over-payment amount is `total_principal_paid - loan_amount`.

## Conclusion

This migration successfully addresses the negative portfolio total issue by implementing a business-logic-driven solution that caps outstanding balances at zero. The change is backward-compatible, maintains data integrity, and improves the accuracy of portfolio metrics across the system.

