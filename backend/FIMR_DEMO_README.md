# FIMR Demo Test Data

## Overview

This document describes the test data created to demonstrate FIMR (First Installment Miss Rate) tracking and other analytics metrics in the system.

## Test Script

**File:** `backend/test-fimr-simple.sh`

**Usage:**
```bash
bash backend/test-fimr-simple.sh
```

## Test Data Created

### Customers

1. **Inyang Kpongette** (CUST2024100001)
   - Phone: +234-801-234-5678
   - State: Lagos
   - KYC Status: Verified

2. **Shamsideen Allamu** (CUST2024100002)
   - Phone: +234-803-456-7890
   - State: Lagos
   - KYC Status: Verified

### Loans

#### Loan 1 - Inyang Kpongette (FIMR Case)

**Loan Details:**
- Loan ID: LN2024100001
- Customer: Inyang Kpongette
- Officer: Sarah Johnson (OFF2024012)
- Amount: ₦1,000,000
- Disbursement Date: 60 days ago from today
- Maturity Date: 30 days from today
- Duration: 90 days
- Interest Rate: 10% per month
- Total Interest: ₦300,000
- Processing Fee: ₦50,000
- Daily Repayment: ₦14,999.99

**Repayment Pattern:**
- **MISSED** first 10 days (Days 1-10)
- **PAID** days 11-20
- Total payments made: 10 out of 90

**Expected Metrics:**
- `first_payment_missed`: TRUE
- `fimr_tagged`: TRUE
- `current_dpd`: ~59 days
- Should appear in FIMR loans endpoint

---

#### Loan 2 - Shamsideen Allamu (Good Payer, NO FIMR)

**Loan Details:**
- Loan ID: LN2024100002
- Customer: Shamsideen Allamu
- Officer: Sarah Johnson (OFF2024012)
- Amount: ₦2,000,000
- Disbursement Date: 35 days ago from today
- Maturity Date: 55 days from today
- Duration: 90 days
- Interest Rate: 10% per month
- Total Interest: ₦600,000
- Processing Fee: ₦100,000
- Daily Repayment: ₦29,999.99

**Repayment Pattern:**
- **PAID ON TIME** from day 1
- **PAID** days 1-20
- Total payments made: 20 out of 90

**Expected Metrics:**
- `first_payment_missed`: FALSE
- `fimr_tagged`: FALSE
- `current_dpd`: ~14 days (for unpaid future installments)
- Should NOT appear in FIMR loans endpoint
- Should appear in Early Indicators endpoint (DPD 1-30)

---

## Verification

### 1. FIMR Loans Endpoint

```bash
curl -s http://localhost:8080/api/v1/fimr/loans | jq .
```

**Expected Result:**
- Total: 1 loan
- Loan: LN2024100001 (Inyang Kpongette)

### 2. Early Indicators Endpoint

```bash
curl -s http://localhost:8080/api/v1/early-indicators/loans | jq .
```

**Expected Result:**
- Total: 1 loan
- Loan: LN2024100002 (Shamsideen Allamu)
- DPD: ~14 days

### 3. Officers Metrics

```bash
curl -s http://localhost:8080/api/v1/officers | jq '.data.officers[] | select(.officer_id == "OFF2024012")'
```

**Expected Result:**
- Officer: Sarah Johnson
- Updated metrics reflecting both loans
- FIMR rate calculated based on 1 FIMR loan out of 2 total loans

### 4. Portfolio Metrics

```bash
curl -s http://localhost:8080/api/v1/metrics/portfolio | jq .
```

**Expected Result:**
- Total Loans: 3 (including the original test loan)
- Total Portfolio: ~₦2,864,444
- Updated DQI, AYR, and other metrics

### 5. Database State

```bash
docker exec -it analytics-postgres psql -U analytics_user -d analytics_db -c "
SELECT 
    loan_id, 
    customer_name, 
    first_payment_missed,
    fimr_tagged,
    current_dpd,
    (SELECT MIN(payment_date) FROM repayments WHERE repayments.loan_id = loans.loan_id) as first_payment_date,
    (SELECT MIN(due_date) FROM loan_schedule WHERE loan_schedule.loan_id = loans.loan_id) as first_due_date
FROM loans 
WHERE loan_id IN ('LN2024100001', 'LN2024100002')
ORDER BY loan_id;
"
```

**Expected Result:**
```
   loan_id    |   customer_name   | first_payment_missed | fimr_tagged | current_dpd | first_payment_date | first_due_date 
--------------+-------------------+----------------------+-------------+-------------+--------------------+----------------
 LN2024100001 | Inyang Kpongette  | t                    | t           |          59 | 2025-08-30         | 2025-08-21
 LN2024100002 | Shamsideen Allamu | f                    | f           |          14 | 2025-09-15         | 2025-09-15
```

---

## How FIMR Tracking Works

### Database Trigger Logic

The `update_loan_metrics()` function is automatically triggered after each repayment is posted. It:

1. **Calculates first payment date**: Gets the earliest payment date from the repayments table
2. **Gets first due date**: Gets the earliest due date from the loan_schedule table
3. **Determines FIMR status**: 
   - `first_payment_missed = TRUE` if first_payment_date > first_due_date OR first_payment_date IS NULL
   - `fimr_tagged = TRUE` if first_payment_missed is TRUE

### Loan Schedule

Each loan has a schedule of 90 daily installments created in the `loan_schedule` table:
- Installment 1 due date = Disbursement date + 1 day
- Installment 2 due date = Disbursement date + 2 days
- ... and so on

### Payment Matching

When a repayment is posted:
1. The trigger calculates which installments should be marked as "Paid"
2. Updates the loan's computed fields (DPD, outstanding balances, etc.)
3. Tags the loan as FIMR if the first payment was late

---

## Cleanup

To remove the test data:

```bash
docker exec -i analytics-postgres psql -U analytics_user -d analytics_db <<EOF
DELETE FROM loan_schedule WHERE loan_id IN ('LN2024100001', 'LN2024100002');
DELETE FROM repayments WHERE loan_id IN ('LN2024100001', 'LN2024100002');
DELETE FROM loans WHERE loan_id IN ('LN2024100001', 'LN2024100002');
DELETE FROM customers WHERE customer_id IN ('CUST2024100001', 'CUST2024100002');
EOF
```

---

## Integration with Frontend

The frontend dashboard at http://localhost:5173/ can now display:

1. **FIMR Drilldown**: Shows loans that missed their first payment
2. **Early Indicators**: Shows loans in early delinquency (DPD 1-30)
3. **Agent Performance**: Shows officer metrics including FIMR rate
4. **Portfolio Metrics**: Shows overall portfolio health including FIMR

To connect the frontend to the backend API, update the API base URL in the frontend configuration.

---

## Notes

- The test script uses `CURRENT_DATE` for date calculations, so the exact dates will vary based on when you run the script
- The script is idempotent - you can run it multiple times and it will update existing records
- All monetary values are in Nigerian Naira (₦)
- The daily repayment amounts are calculated as: (Principal + Interest + Fees) / 90 days

---

## Troubleshooting

### FIMR loans not showing up

1. Check if loan_schedule entries exist:
   ```bash
   docker exec -it analytics-postgres psql -U analytics_user -d analytics_db -c "SELECT COUNT(*) FROM loan_schedule WHERE loan_id = 'LN2024100001';"
   ```

2. Check if repayments exist:
   ```bash
   docker exec -it analytics-postgres psql -U analytics_user -d analytics_db -c "SELECT COUNT(*) FROM repayments WHERE loan_id = 'LN2024100001';"
   ```

3. Manually trigger the metrics calculation:
   ```bash
   docker exec -i analytics-postgres psql -U analytics_user -d analytics_db -c "SELECT update_loan_metrics('LN2024100001'::VARCHAR);"
   ```

### Database connection issues

1. Check if Docker containers are running:
   ```bash
   cd backend && docker-compose ps
   ```

2. Restart the backend:
   ```bash
   cd backend && docker-compose restart
   ```

---

**Created:** 2025-10-19  
**Last Updated:** 2025-10-19

