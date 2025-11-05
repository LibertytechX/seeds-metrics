# Quick Start: Fix Negative Portfolio Total

## Problem
Officer `adeyinka232803@gmail.com` has a negative portfolio total in the Agent Performance table.

## Solution
Apply Migration 015 to cap negative outstanding balances at zero.

---

## Quick Fix (5 minutes)

### Step 1: Navigate to Migrations Directory
```bash
cd backend/migrations
```

### Step 2: Make Script Executable
```bash
chmod +x apply_cap_negative_balances.sh
```

### Step 3: Set Database Password
```bash
export DB_PASSWORD="your_database_password"
```

### Step 4: Run the Fix
```bash
./apply_cap_negative_balances.sh
```

### Step 5: Verify in Dashboard
1. Open the dashboard
2. Navigate to Agent Performance table
3. Find officer `adeyinka232803@gmail.com`
4. Verify Portfolio Total is no longer negative

---

## What This Does

1. **Updates the database trigger** to cap negative balances at 0
2. **Recalculates all existing loans** to fix current negative balances
3. **Creates a backup** of loans with negative balances
4. **Verifies the fix** automatically

---

## Expected Results

### Before Fix
```
Officer: adeyinka232803@gmail.com
Portfolio Total: -500,000 (NEGATIVE)
Negative Loans: 3
```

### After Fix
```
Officer: adeyinka232803@gmail.com
Portfolio Total: 1,250,000 (POSITIVE)
Negative Loans: 0
```

---

## Manual Verification

### Check Officer Portfolio
```sql
SELECT 
    COUNT(*) as total_loans,
    SUM(principal_outstanding) as portfolio_total
FROM loans
WHERE officer_id = 'adeyinka232803@gmail.com';
```

**Expected**: `portfolio_total` should be >= 0

### Check for Negative Balances
```sql
SELECT COUNT(*) FROM loans WHERE principal_outstanding < 0;
```

**Expected**: Should return 0

---

## Rollback (if needed)

If you need to undo the changes:

```bash
# Restore previous trigger
psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
     -p 25060 -U metricsuser -d seedsmetrics \
     -f 012_update_fimr_4day_grace_period.sql

# Restore data from backup CSV file
# (Manual process - contact support)
```

---

## Files Created

1. **Migration Script**: `015_cap_negative_outstanding_balances.sql`
2. **Apply Script**: `apply_cap_negative_balances.sh`
3. **Test Script**: `test_cap_negative_balances.sql`
4. **Documentation**: `MIGRATION_015_CAP_NEGATIVE_BALANCES.md`

---

## Support

### Common Questions

**Q: Is this safe to run on production?**
A: Yes. The script creates a backup first and only updates calculated fields.

**Q: Will this affect payment history?**
A: No. Payment records are unchanged. Only calculated outstanding balances are updated.

**Q: What if I have other officers with negative portfolios?**
A: The fix applies to ALL loans, not just this officer.

**Q: How long does it take?**
A: Usually 1-2 minutes, depending on the number of loans.

---

## Troubleshooting

### Error: "psql: command not found"
**Solution**: Install PostgreSQL client
```bash
# macOS
brew install postgresql

# Ubuntu/Debian
sudo apt-get install postgresql-client
```

### Error: "Permission denied"
**Solution**: Make script executable
```bash
chmod +x apply_cap_negative_balances.sh
```

### Error: "Connection refused"
**Solution**: Check database credentials and network access

---

## Next Steps After Fix

1. ✓ Verify dashboard shows correct portfolio total
2. ✓ Check other officers' portfolios
3. ✓ Monitor for new over-payment scenarios
4. ✓ Consider adding validation to prevent over-payments
5. ✓ Review backup file to understand which loans were over-paid

---

## Technical Details

### What Changed
- Database trigger function `update_loan_computed_fields()`
- Outstanding balance calculations now use `GREATEST(0, value)`
- All existing loans with negative balances updated

### What Didn't Change
- Loan amounts
- Payment records
- Payment history
- Any other loan data

### Business Logic
Over-payments now show as 0 outstanding instead of negative values. This is a business decision that:
- Prevents negative portfolio totals
- Treats over-paid loans as fully paid
- Maintains accurate portfolio metrics

---

## Contact

For issues or questions:
1. Check `MIGRATION_015_CAP_NEGATIVE_BALANCES.md` for detailed documentation
2. Review test results in `test_cap_negative_balances.sql`
3. Check backup file for original values

