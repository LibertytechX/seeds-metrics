# Loan Metrics Validation Test Suite

## Overview

This test suite validates that all computed loan metrics in the SeedsMetrics database are correctly calculated and match the source data from the Django database.

## Purpose

After every new implementation or migration that affects loan calculations, run this test suite to ensure:

1. **Data Sync Integrity**: Basic loan fields match between Django and SeedsMetrics databases
2. **Repayment Totals**: All repayment aggregations are correct
3. **Outstanding Balances**: Principal, interest, and fees outstanding are calculated correctly
4. **Business Days Calculations**: All business day counts are accurate
5. **DPD Calculations**: Days Past Due is calculated correctly
6. **Actual Outstanding**: Time-based overdue amounts are correct
7. **Repayment Delay Rate**: Repayment behavior metrics are accurate
8. **FIMR Tagging**: First Installment Missed Repayment flags are correct

## Test Loans

The suite tests 10 randomly selected loans:
- Loan IDs: `8`, `3475`, `11883`, `2118`, `7150`, `12129`, `107`, `3523`, `16192`, `18691`

These loans provide a representative sample across different:
- Loan ages
- Repayment statuses
- DPD levels
- Payment histories

## Metrics Validated

### Basic Fields (6 tests per loan)
- Loan Amount
- Disbursement Date
- Maturity Date
- Loan Term Days
- Interest Rate
- First Payment Due Date

### Repayment Totals (4 tests per loan)
- Total Principal Paid
- Total Interest Paid
- Total Fees Paid
- Total Repayments

### Outstanding Balances (4 tests per loan)
- Principal Outstanding
- Interest Outstanding
- Fees Outstanding
- Total Outstanding

### Business Days Calculations (4 tests per loan)
- Business Days Since Disbursement
- Loan Age
- Daily Repayment Amount
- Repayment Days Paid

### DPD and Risk Indicators (3 tests per loan)
- Current DPD
- Early Indicator Tagged
- Actual Outstanding

### Health Metrics (1 test per loan)
- Repayment Delay Rate

### FIMR Tagging (2 tests per loan)
- FIMR Tagged
- First Payment Missed

**Total: ~24 tests per loan × 10 loans = ~240 tests**

## Usage

### Prerequisites

1. Python 3.7+
2. `psycopg2` library installed:
   ```bash
   pip install psycopg2-binary
   ```

### Running the Tests

#### Option 1: Using the shell script (recommended)
```bash
./backend/tests/run_metrics_validation.sh
```

#### Option 2: Direct Python execution
```bash
python backend/tests/test_loan_metrics_validation.py
```

#### Option 3: From production server
```bash
ssh root@143.198.146.44
cd /home/seeds-metrics-backend
python3 backend/tests/test_loan_metrics_validation.py
```

### Environment Variables

The test suite uses these environment variables (with defaults):

**Django Database:**
- `DJANGO_DB_HOST` (default: `164.90.155.2`)
- `DJANGO_DB_PORT` (default: `5432`)
- `DJANGO_DB_NAME` (default: `savings`)
- `DJANGO_DB_USER` (default: `metricsuser`)
- `DJANGO_DB_PASSWORD` (default: hardcoded in script)

**SeedsMetrics Database:**
- `SEEDSMETRICS_DB_HOST` (default: `generaldb-do-user-9489371-0.k.db.ondigitalocean.com`)
- `SEEDSMETRICS_DB_PORT` (default: `25060`)
- `SEEDSMETRICS_DB_NAME` (default: `seedsmetrics`)
- `SEEDSMETRICS_DB_USER` (default: `seedsuser`)
- `SEEDSMETRICS_DB_PASSWORD` (default: hardcoded in script)

## Output

The test suite provides colored terminal output:

- ✓ **Green**: Test passed
- ✗ **Red**: Test failed
- ⚠ **Yellow**: Warning or skipped test

### Example Output

```
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

...

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

## When to Run

Run this test suite:

1. **After every migration** that affects loan calculations
2. **After deploying trigger function changes**
3. **After running sync scripts** (e.g., `update_first_payment_due_date.sql`)
4. **After recalculating loan fields** (e.g., `recalculate_all_loan_fields()`)
5. **Before releasing to production**
6. **When investigating data discrepancies**

## Interpreting Results

### All Tests Pass (100%)
✅ All metrics are correctly calculated. Safe to proceed.

### High Pass Rate (>95%)
⚠️ Most metrics are correct, but investigate failures. Common issues:
- Stale `repayment_delay_rate` (trigger function not updating it)
- Rounding differences (acceptable if within tolerance)

### Low Pass Rate (<95%)
🚨 Critical issues detected. Do NOT deploy. Common causes:
- Trigger function not running
- Sync script failed
- Formula bugs in migrations
- Data corruption

## Troubleshooting

### Test fails with "Loan not found"
- Loan may have been deleted or not synced yet
- Update `TEST_LOAN_IDS` in the script with valid loan IDs

### Connection errors
- Check database credentials
- Verify network access to databases
- Ensure SSL certificates are valid

### All repayment_delay_rate tests fail
- Trigger function likely not calculating this field
- Check if migration 037 removed the calculation
- Run the fix to add it back to the trigger

## Maintenance

To update test loans with a new random sample:

```bash
PGPASSWORD='@seedsuser2020' psql -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 -U seedsuser -d seedsmetrics \
  -c "SELECT loan_id FROM loans WHERE disbursement_date IS NOT NULL AND status != 'CANCELLED' ORDER BY RANDOM() LIMIT 10;"
```

Then update `TEST_LOAN_IDS` in `test_loan_metrics_validation.py`.

