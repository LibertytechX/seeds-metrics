# Deployment Checklist - Loan Metrics Validation

Use this checklist before every deployment that affects loan calculations.

---

## Pre-Deployment Checklist

### 1. Code Changes
- [ ] Review all migration files for formula changes
- [ ] Check trigger function modifications
- [ ] Verify sync script updates
- [ ] Review any changes to loan calculation logic

### 2. Local Testing
- [ ] Run migrations locally
- [ ] Test trigger function with sample data
- [ ] Verify sync scripts work correctly
- [ ] Check for SQL syntax errors

### 3. Run Validation Test Suite
```bash
./backend/tests/run_metrics_validation.sh
```

- [ ] Test suite runs without errors
- [ ] Review test results
- [ ] Pass rate is 100% (or 95%+ with known issues documented)
- [ ] All critical metrics pass (DPD, actual_outstanding, totals)

### 4. Review Failed Tests
If any tests fail:

- [ ] Identify which metrics failed
- [ ] Determine root cause (formula bug, sync issue, etc.)
- [ ] Fix the issue
- [ ] Re-run test suite
- [ ] Verify all tests now pass

### 5. Documentation
- [ ] Update migration notes with any formula changes
- [ ] Document any known issues
- [ ] Update CHANGELOG if applicable
- [ ] Add comments to complex calculations

---

## Deployment Steps

### 1. Backup Database
```bash
ssh root@143.198.146.44
pg_dump -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 -U seedsuser -d seedsmetrics > backup_$(date +%Y%m%d_%H%M%S).sql
```

- [ ] Database backup created
- [ ] Backup file verified (not empty)
- [ ] Backup stored securely

### 2. Deploy Code
```bash
git add .
git commit -m "Your commit message"
git push origin main
```

- [ ] Code pushed to repository
- [ ] CI/CD pipeline triggered (if applicable)
- [ ] No merge conflicts

### 3. Pull on Production Server
```bash
ssh root@143.198.146.44
cd /home/seeds-metrics-backend
git pull origin main
```

- [ ] Code pulled successfully
- [ ] No file conflicts
- [ ] All files updated

### 4. Run Migrations (if applicable)
```bash
# Run new migration files
psql -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 -U seedsuser -d seedsmetrics \
  -f backend/migrations/XXX_your_migration.sql
```

- [ ] Migration executed successfully
- [ ] No SQL errors
- [ ] Trigger function updated (if applicable)

### 5. Run Sync Scripts (if applicable)
```bash
# Example: Update first payment due dates
psql -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 -U seedsuser -d seedsmetrics \
  -f backend/scripts/update_first_payment_due_date.sql
```

- [ ] Sync script executed successfully
- [ ] Verify number of rows updated
- [ ] Check for any errors

### 6. Recalculate Loan Fields (if needed)
```bash
psql -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 -U seedsuser -d seedsmetrics \
  -c "SELECT recalculate_all_loan_fields();"
```

- [ ] Recalculation completed
- [ ] Check number of loans updated
- [ ] Verify no errors

### 7. Restart Backend Service (if needed)
```bash
sudo systemctl restart seeds-metrics-backend
sudo systemctl status seeds-metrics-backend
```

- [ ] Service restarted successfully
- [ ] Service is running
- [ ] No errors in logs

---

## Post-Deployment Validation

### 1. Run Test Suite on Production
```bash
ssh root@143.198.146.44
cd /home/seeds-metrics-backend
python3 backend/tests/test_loan_metrics_validation.py
```

- [ ] Test suite runs successfully
- [ ] Pass rate is 100% (or expected rate)
- [ ] No unexpected failures
- [ ] All critical metrics pass

### 2. Spot Check Specific Loans
Pick 2-3 loans and manually verify:

```bash
# Check loan metrics
psql -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 -U seedsuser -d seedsmetrics \
  -c "SELECT * FROM loans WHERE loan_id = 'LOAN_ID';"
```

- [ ] Loan amounts match Django
- [ ] DPD calculations are correct
- [ ] Outstanding balances are accurate
- [ ] Repayment totals match repayments table

### 3. Test API Endpoints
```bash
# Test FIMR loans endpoint
curl -s "https://metrics.seedsandpennies.com/api/v1/fimr/loans?limit=5" | jq '.'

# Test specific loan endpoint
curl -s "https://metrics.seedsandpennies.com/api/v1/loans/LOAN_ID" | jq '.'
```

- [ ] API returns 200 OK
- [ ] No 500 errors
- [ ] Data looks correct
- [ ] Response time is acceptable

### 4. Check Logs
```bash
# Check backend logs
sudo journalctl -u seeds-metrics-backend -n 100 --no-pager

# Check for errors
sudo journalctl -u seeds-metrics-backend -p err -n 50 --no-pager
```

- [ ] No critical errors
- [ ] No database connection issues
- [ ] No calculation errors

### 5. Monitor for 24 Hours
- [ ] Set reminder to check again in 24 hours
- [ ] Monitor error logs
- [ ] Check for user reports of issues
- [ ] Verify metrics are updating correctly

---

## Rollback Plan (If Issues Found)

### 1. Immediate Actions
- [ ] Stop deployment
- [ ] Document the issue
- [ ] Notify stakeholders

### 2. Restore Database (if needed)
```bash
# Restore from backup
psql -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 -U seedsuser -d seedsmetrics < backup_TIMESTAMP.sql
```

- [ ] Backup restored
- [ ] Verify data integrity
- [ ] Run test suite to confirm

### 3. Revert Code (if needed)
```bash
git revert HEAD
git push origin main

# On production
cd /home/seeds-metrics-backend
git pull origin main
sudo systemctl restart seeds-metrics-backend
```

- [ ] Code reverted
- [ ] Service restarted
- [ ] Test suite passes

### 4. Post-Rollback Validation
- [ ] Run test suite
- [ ] Verify all tests pass
- [ ] Check API endpoints
- [ ] Monitor logs

---

## Sign-Off

**Deployment Date:** _______________  
**Deployed By:** _______________  
**Test Suite Pass Rate:** _______________  
**Issues Found:** _______________  
**Resolution:** _______________  

**Approved for Production:** [ ] YES [ ] NO

**Signature:** _______________

---

## Notes

Use this space to document any issues, workarounds, or observations:

```
[Your notes here]
```

