# Days Since Due Calculation Fix - Loan ID 9623

## Issue Summary

**Loan ID 9623** (Customer: Rafiu Samod) was showing an incorrect `days_since_due` value of **193 days** in the API response, when the correct value should have been **153 days**.

### Loan Details
- **Loan ID**: 9623
- **Customer**: Rafiu Samod (Phone: 09126660778)
- **Disbursement Date**: 27 Mar 2025
- **First Payment Due Date**: 26 Apr 2025
- **First Payment Received Date**: 26 Sep 2025
- **Loan Amount**: ₦400,000

### Expected vs Actual
- **Expected `days_since_due`**: 153 days (26 Sep 2025 - 26 Apr 2025)
- **Incorrect API value**: 193 days
- **Correct API value after fix**: 153 days ✅

## Root Cause Analysis

The issue was **NOT** with the database calculation or the backend code logic. The problem was with the **binary deployment process**.

### What Happened

1. **Build Process**: The backend code was being built to `/home/seeds-metrics-backend/backend/bin/api`
2. **Service Configuration**: The systemd service was configured to run `/home/seeds-metrics-backend/backend/api`
3. **Mismatch**: The service was running an **old binary** from 20:21 (before the fix was deployed)
4. **New Binary**: The new binary was built at 23:03 but was in the wrong location (`bin/api` instead of `api`)

### Timeline

- **20:21**: Old binary at `/home/seeds-metrics-backend/backend/api` (running incorrect code)
- **23:01**: New binary built to `/home/seeds-metrics-backend/backend/bin/api` (correct code)
- **23:03**: New binary copied to `/home/seeds-metrics-backend/backend/api` and service restarted
- **23:03**: API now returns correct value (153 instead of 193)

## Investigation Steps

1. ✅ Verified database contains correct value: `days_since_due = 153`
2. ✅ Verified trigger function logic is correct
3. ✅ Verified backend query selects correct column
4. ✅ Verified scan logic maps to correct field
5. ✅ Discovered binary location mismatch
6. ✅ Deployed correct binary to production

## The Fix

**File**: `/home/seeds-metrics-backend/backend/api`

**Action**: Copied the newly built binary from `bin/api` to `api` and restarted the systemd service.

```bash
systemctl stop seeds-metrics-api
cp /home/seeds-metrics-backend/backend/bin/api /home/seeds-metrics-backend/backend/api
systemctl start seeds-metrics-api
```

## Verification

### Database Query Result
```sql
SELECT loan_id, customer_name, first_payment_due_date, 
       first_payment_received_date, days_since_due, current_dpd
FROM loans
WHERE loan_id = '9623';

-- Result:
-- loan_id | customer_name | first_payment_due_date | first_payment_received_date | days_since_due | current_dpd
-- 9623    | Rafiu Samod   | 2025-04-26             | 2025-09-26                  |            153 |         193
```

### API Response (After Fix)
```json
{
  "loan_id": "9623",
  "customer_name": "Rafiu Samod",
  "first_payment_due_date": "2025-04-26T00:00:00Z",
  "first_payment_received_date": "2025-09-26T00:00:00Z",
  "days_since_due": 153,
  "current_dpd": 193
}
```

### Additional Test Cases
| Loan ID | Customer Name | Due Date | Received Date | days_since_due | Status |
|---------|---------------|----------|---------------|----------------|--------|
| 9623 | Rafiu Samod | 2025-04-26 | 2025-09-26 | 153 | ✅ Correct |
| 18597 | RUKAYAT AMOSUN | 2025-10-25 | 2025-10-27 | 2 | ✅ Correct |
| 19704 | RISQIYAT TAOHEED | 2025-11-04 | (none) | 1 | ✅ Correct |

## Key Learnings

1. **Binary Deployment**: Always ensure the binary is deployed to the correct location that the service is configured to run
2. **Service Configuration**: Verify the systemd service ExecStart path matches where the binary is actually built
3. **Verification**: After deployment, verify the service is running the new binary (check timestamps)

## Recommendation

To prevent this issue in the future:
1. Update the build process to output directly to `/home/seeds-metrics-backend/backend/api`
2. Or update the systemd service to run from `/home/seeds-metrics-backend/backend/bin/api`
3. Add a deployment verification step to check binary timestamps

## Status

✅ **FIXED AND DEPLOYED TO PRODUCTION**

All FIMR loans now display the correct `days_since_due` value calculated as:
- **If payment received**: `first_payment_received_date - first_payment_due_date`
- **If no payment**: `CURRENT_DATE - first_payment_due_date`

