# Loan Metrics Validation Test Suite - Summary

## Overview

A comprehensive test suite has been created to validate all computed loan metrics in the SeedsMetrics database against the Django source database.

---

## Files Created

### 1. **test_loan_metrics_validation.py** (644 lines)
Main Python test suite that validates all loan metrics.

**Features:**
- Connects to both Django and SeedsMetrics databases
- Tests 10 randomly selected loans
- Validates ~24 metrics per loan (~240 total tests)
- Colored terminal output (green ✓ for pass, red ✗ for fail)
- Detailed error reporting with expected vs actual values
- Exit code 0 for success, 1 for failures

**Test Categories:**
1. Basic Fields (6 tests/loan) - Loan amount, dates, terms
2. Repayment Totals (4 tests/loan) - Principal, interest, fees paid
3. Outstanding Balances (4 tests/loan) - What's left to pay
4. Business Days Calculations (4 tests/loan) - Business day counts
5. Actual Outstanding (1 test/loan) - Overdue amount
6. DPD Calculations (2 tests/loan) - Days past due
7. Repayment Delay Rate (1 test/loan) - Behavior score
8. FIMR Tagging (2 tests/loan) - First payment missed indicators

### 2. **run_metrics_validation.sh** (67 lines)
Shell script to easily run the test suite.

**Features:**
- Checks for Python 3 and psycopg2
- Auto-installs psycopg2 if missing
- Sets default environment variables
- Colored output
- Returns appropriate exit codes

**Usage:**
```bash
./backend/tests/run_metrics_validation.sh
```

### 3. **README_METRICS_VALIDATION.md** (234 lines)
Comprehensive documentation for the test suite.

**Contents:**
- Purpose and overview
- Test loans list
- Metrics validated (detailed breakdown)
- Usage instructions
- Environment variables
- Output examples
- When to run
- Interpreting results
- Troubleshooting guide
- Maintenance instructions

### 4. **METRICS_REFERENCE.md** (234 lines)
Complete reference guide for all 56 loan metrics.

**Contents:**
- All 56 fields organized by category
- Field types and sources
- Formulas for computed fields
- Critical formula explanations
- Data flow diagram
- Test coverage summary

**Categories:**
1. Basic Loan Information (14 fields)
2. Loan Terms (6 fields)
3. Loan Status (4 fields)
4. Repayment Totals (4 fields)
5. Outstanding Balances (5 fields)
6. Payment Dates (3 fields)
7. DPD Metrics (3 fields)
8. Business Days Calculations (5 fields)
9. Loan Age and Timing (2 fields)
10. Risk Indicators (3 fields)
11. Health Metrics (3 fields)
12. Organizational Fields (2 fields)
13. Metadata (2 fields)

### 5. **QUICK_START.md** (234 lines)
Quick reference guide for running tests.

**Contents:**
- TL;DR section
- What this tests
- When to run
- How to run (local and production)
- Understanding results (with examples)
- Known issues
- Test loan IDs
- Example full test run

---

## Test Loan IDs

10 randomly selected loans for testing:
- `8`, `3475`, `11883`, `2118`, `7150`, `12129`, `107`, `3523`, `16192`, `18691`

These provide diverse coverage across:
- Different loan ages
- Various repayment statuses
- Different DPD levels
- Multiple payment histories

---

## Current Known Issue

### Repayment Delay Rate Bug

**Finding:** Loan 19539 (and likely all loans) show `repayment_delay_rate = 100.00` even when they have high DPD.

**Root Cause:** Migration 037 accidentally removed the `repayment_delay_rate` calculation from the trigger function `update_loan_computed_fields()`.

**Evidence:**
- Trigger function doesn't include `repayment_delay_rate` in UPDATE statement
- All loans show 100.00 (stale data from before migration 037)
- Manual calculation shows loan 19539 should be -80.65, not 100.00

**Impact:**
- All `repayment_delay_rate` values are incorrect
- Values are frozen at pre-migration 037 state
- Affects repayment behavior analysis

**Expected Test Results:**
- When you run the test suite, expect ~10 failures (all `repayment_delay_rate` tests)
- Pass rate should be ~95.8% (230/240 tests passing)
- All other metrics should pass

**Fix Required:**
Add the `repayment_delay_rate` calculation back to the trigger function in migration 037 (use formula from migration 035).

---

## How to Use This Test Suite

### Step 1: Run the Tests

```bash
./backend/tests/run_metrics_validation.sh
```

### Step 2: Review Results

**If 100% pass:**
✅ All metrics are correct. Safe to deploy.

**If 95-99% pass (only repayment_delay_rate fails):**
⚠️ Known issue. Fix the trigger function, then re-run tests.

**If <95% pass:**
🚨 Critical issues. DO NOT DEPLOY. Investigate failures.

### Step 3: After Fixing Issues

1. Deploy the fix
2. Run `recalculate_all_loan_fields()` if needed
3. Re-run the test suite
4. Verify 100% pass rate

### Step 4: Integrate into Workflow

Add to your deployment checklist:
- [ ] Run migrations
- [ ] Run sync scripts
- [ ] **Run metrics validation test suite**
- [ ] Verify 100% pass rate
- [ ] Deploy to production

---

## Example Test Output

```
================================================================================
Validating Loan ID: 19539
================================================================================

Basic Fields Validation
✓ Loan Amount: 300000.00
✓ Disbursement Date: 2025-10-24
✓ Maturity Date: 2026-01-22
✓ Loan Term Days: 90
✓ Interest Rate: 0.2800
✓ First Payment Due Date: 2025-10-29

Repayment Totals Validation
✓ Total Principal Paid: 24200.00
✓ Total Interest Paid: 0.00
✓ Total Fees Paid: 0.00
✓ Total Repayments: 24200.00

Outstanding Balances Validation
✓ Principal Outstanding: 275800.00
✓ Interest Outstanding: 84000.00
✓ Fees Outstanding: 18000.00
✓ Total Outstanding: 377800.00

Business Days Calculations Validation
✓ Business Days Since Disbursement: 23
✓ Loan Age: 31
✓ Daily Repayment Amount: 4674.42
✓ Repayment Days Paid: 5.18

Actual Outstanding Validation
✓ Actual Outstanding: 64613.95

DPD Calculations Validation
✓ Current DPD: 14
✓ Early Indicator Tagged: False

Repayment Delay Rate Validation
✗ Repayment Delay Rate
  Expected: -80.65
  Actual:   100.00

FIMR Tagging Validation
✓ FIMR Tagged: True
✓ First Payment Missed: True
```

---

## Benefits

1. **Catch Bugs Early:** Detect calculation errors before they reach production
2. **Prevent Data Corruption:** Ensure sync scripts work correctly
3. **Validate Migrations:** Confirm trigger functions calculate correctly
4. **Document Metrics:** Comprehensive reference for all 56 fields
5. **Regression Testing:** Ensure new changes don't break existing calculations
6. **Confidence:** Deploy with confidence knowing metrics are correct

---

## Next Steps

1. **Run the test suite now** to establish baseline
2. **Fix the repayment_delay_rate bug** in the trigger function
3. **Re-run tests** to verify 100% pass rate
4. **Add to CI/CD** for automated testing
5. **Update test loans periodically** for better coverage

---

## Questions?

Refer to:
- **QUICK_START.md** - Quick reference
- **README_METRICS_VALIDATION.md** - Detailed documentation
- **METRICS_REFERENCE.md** - Complete metrics guide

