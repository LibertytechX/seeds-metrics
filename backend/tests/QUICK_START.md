# Quick Start Guide - Loan Metrics Validation

## TL;DR

Run this after every deployment or migration:

```bash
./backend/tests/run_metrics_validation.sh
```

If all tests pass ✅ → You're good to go!  
If tests fail ❌ → Investigate before deploying to production.

---

## What This Tests

This validates that **all computed loan metrics** in SeedsMetrics match the source data from Django.

**Test Coverage:**
- ✓ 10 randomly selected loans
- ✓ ~24 metrics per loan
- ✓ ~240 total validations

**Metrics Validated:**
1. Basic loan fields (amount, dates, terms)
2. Repayment totals (principal, interest, fees)
3. Outstanding balances (what's left to pay)
4. Business days calculations
5. DPD (Days Past Due)
6. Actual outstanding (overdue amount)
7. Repayment delay rate (behavior score)
8. FIMR tagging (first payment missed)

---

## When to Run

Run this test suite:

1. ✅ **After every migration** that touches loan calculations
2. ✅ **After deploying trigger function changes**
3. ✅ **After running sync scripts**
4. ✅ **After calling `recalculate_all_loan_fields()`**
5. ✅ **Before releasing to production**
6. ✅ **When investigating data issues**

---

## How to Run

### On Your Local Machine

```bash
# Make sure you have Python 3 and psycopg2
pip install psycopg2-binary

# Run the test suite
./backend/tests/run_metrics_validation.sh
```

### On Production Server

```bash
ssh root@143.198.146.44
cd /home/seeds-metrics-backend

# Install dependencies (first time only)
pip3 install psycopg2-binary

# Run tests
python3 backend/tests/test_loan_metrics_validation.py
```

---

## Understanding Results

### ✅ All Tests Pass (100%)

```
================================================================================
TEST SUMMARY
================================================================================
Total Tests: 240
Passed: 240
Failed: 0

✓ ALL TESTS PASSED!

Pass Rate: 100.0%
================================================================================
```

**Action:** Safe to deploy! All metrics are correct.

---

### ⚠️ High Pass Rate (95-99%)

```
================================================================================
TEST SUMMARY
================================================================================
Total Tests: 240
Passed: 235
Failed: 5

✗ SOME TESTS FAILED

Failed Tests:
  - Repayment Delay Rate (Loan 8)
  - Repayment Delay Rate (Loan 3475)
  - Repayment Delay Rate (Loan 11883)
  - Repayment Delay Rate (Loan 2118)
  - Repayment Delay Rate (Loan 7150)

Pass Rate: 97.9%
================================================================================
```

**Action:** Investigate failures. Common issues:
- `repayment_delay_rate` not being calculated (trigger function bug)
- Rounding differences (acceptable if within tolerance)

**If only `repayment_delay_rate` fails:** Known issue - trigger function doesn't calculate it. See fix below.

---

### 🚨 Low Pass Rate (<95%)

```
================================================================================
TEST SUMMARY
================================================================================
Total Tests: 240
Passed: 180
Failed: 60

✗ SOME TESTS FAILED

Pass Rate: 75.0%
================================================================================
```

**Action:** DO NOT DEPLOY! Critical issues detected.

**Common Causes:**
- Trigger function not running
- Sync script failed
- Formula bugs in migrations
- Data corruption

**Fix:** Review failed tests, check trigger function, verify sync scripts ran successfully.

---

## Known Issues

### Issue #1: Repayment Delay Rate Always 100.00

**Symptom:** All loans show `repayment_delay_rate = 100.00` even when they have high DPD.

**Cause:** Migration 037 accidentally removed the `repayment_delay_rate` calculation from the trigger function.

**Impact:** The field shows stale data from before migration 037.

**Fix:** Add the calculation back to the trigger function (see migration 035 for correct formula).

**Test:** After fixing, run this test suite to verify all `repayment_delay_rate` values are correct.

---

## Test Loan IDs

Current test loans (randomly selected):
- `8`, `3475`, `11883`, `2118`, `7150`, `12129`, `107`, `3523`, `16192`, `18691`

To generate new random test loans:

```bash
PGPASSWORD='@seedsuser2020' psql \
  -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 -U seedsuser -d seedsmetrics \
  -c "SELECT loan_id FROM loans WHERE disbursement_date IS NOT NULL AND status != 'CANCELLED' ORDER BY RANDOM() LIMIT 10;"
```

Then update `TEST_LOAN_IDS` in `test_loan_metrics_validation.py`.

---

## Files

- **`test_loan_metrics_validation.py`** - Main test suite (Python script)
- **`run_metrics_validation.sh`** - Shell script to run tests
- **`README_METRICS_VALIDATION.md`** - Detailed documentation
- **`METRICS_REFERENCE.md`** - Complete metrics reference (all 56 fields)
- **`QUICK_START.md`** - This file

---

## Example: Full Test Run

```bash
$ ./backend/tests/run_metrics_validation.sh

========================================
Loan Metrics Validation Test Runner
========================================

Running test suite...

================================================================================
Loan Metrics Validation Test Suite
================================================================================

Connecting to Django database...
✓ Connected to Django database

Connecting to SeedsMetrics database...
✓ Connected to SeedsMetrics database

Testing 10 loans: 8, 3475, 11883, 2118, 7150, 12129, 107, 3523, 16192, 18691

================================================================================
Validating Loan ID: 8
================================================================================

Basic Fields Validation
✓ Loan Amount: 300000.00
✓ Disbursement Date: 2024-04-15
✓ Maturity Date: 2024-10-15
✓ Loan Term Days: 183
✓ Interest Rate: 0.2800
✓ First Payment Due Date: 2024-04-20

Repayment Totals Validation
✓ Total Principal Paid: 150000.00
✓ Total Interest Paid: 42000.00
✓ Total Fees Paid: 9000.00
✓ Total Repayments: 201000.00

Outstanding Balances Validation
✓ Principal Outstanding: 150000.00
✓ Interest Outstanding: 42000.00
✓ Fees Outstanding: 9000.00
✓ Total Outstanding: 201000.00

Business Days Calculations Validation
✓ Business Days Since Disbursement: 156
✓ Loan Age: 224
✓ Daily Repayment Amount: 2104.92
✓ Repayment Days Paid: 95.48

Actual Outstanding Validation
✓ Actual Outstanding: 125432.10

DPD Calculations Validation
✓ Current DPD: 15
✓ Early Indicator Tagged: False

Repayment Delay Rate Validation
✗ Repayment Delay Rate
  Expected: -45.54
  Actual:   100.00

FIMR Tagging Validation
✓ FIMR Tagged: False
✓ First Payment Missed: False

[... 9 more loans ...]

================================================================================
TEST SUMMARY
================================================================================
Total Tests: 240
Passed: 230
Failed: 10

✗ SOME TESTS FAILED

Failed Tests:
  - Repayment Delay Rate (Loan 8)
  - Repayment Delay Rate (Loan 3475)
  - Repayment Delay Rate (Loan 11883)
  - Repayment Delay Rate (Loan 2118)
  - Repayment Delay Rate (Loan 7150)
  - Repayment Delay Rate (Loan 12129)
  - Repayment Delay Rate (Loan 107)
  - Repayment Delay Rate (Loan 3523)
  - Repayment Delay Rate (Loan 16192)
  - Repayment Delay Rate (Loan 18691)

Pass Rate: 95.8%
================================================================================

========================================
✗ Some tests failed
========================================
```

---

## Next Steps

1. **Run the test suite now** to see current state
2. **Fix any failures** before deploying
3. **Add to CI/CD pipeline** to run automatically
4. **Update test loans periodically** for better coverage

---

## Questions?

See detailed documentation:
- **README_METRICS_VALIDATION.md** - Full test suite documentation
- **METRICS_REFERENCE.md** - Complete metrics reference guide

