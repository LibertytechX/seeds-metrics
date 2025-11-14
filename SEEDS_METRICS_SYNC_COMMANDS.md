# Seeds Metrics System - Complete Data Synchronization & Maintenance Guide

**Last Updated:** 2025-11-09
**Production VM:** root@143.198.146.44
**Backend Location:** /home/seeds-metrics-backend/backend/

---

## 📋 Table of Contents

1. [Data Synchronization Commands](#1-data-synchronization-commands)
2. [Computed Fields & Metrics Recalculation](#2-computed-fields--metrics-recalculation)
3. [Data Enrichment & Mapping](#3-data-enrichment--mapping)
4. [Data Quality & Corrections](#4-data-quality--corrections)
5. [Complete System Refresh Sequence](#5-complete-system-refresh-sequence)
6. [Database Connection Details](#6-database-connection-details)

---

## 1. Data Synchronization Commands

### 1.1 Full Data Sync (All Entities)

**Command:**
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_from_django.go'
```

**What it does:**
- Syncs Officers from Django → Seeds Metrics
- Syncs Customers from Django → Seeds Metrics
- Syncs Loans from Django → Seeds Metrics
- Syncs Repayments from Django → Seeds Metrics

**Execution time:** 10-30 minutes (depending on data volume)
**Prerequisites:** None
**Run location:** Production VM
**Batch size:** 500 loans, 1000 repayments

**Output:**
```
✅ Connected to SeedsMetrics database
✅ Connected to Django database
📊 Syncing Officers...
✅ Officers sync complete: 1,234 successful, 0 errors
📊 Syncing Customers...
✅ Customers sync complete: 15,678 successful, 0 errors
📊 Syncing Loans...
✅ Loans sync complete: 17,929 successful, 0 errors
📊 Syncing Repayments...
✅ Repayments sync complete: 45,123 successful, 0 errors
✅ Data sync completed successfully!
```

---

### 1.2 Incremental Repayments Sync (New Repayments Only)

**Command:**
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_repayments_incremental.go'
```

**What it does:**
- Syncs only NEW repayments created/updated since last sync
- Uses `sync_tracking` table to track last sync timestamp
- Much faster than full sync for daily updates

**Execution time:** 1-5 minutes
**Prerequisites:** `sync_tracking` table must exist (created by migration 024)
**Run location:** Production VM
**Recommended frequency:** Daily or after bulk repayment imports

**Output:**
```
📅 Last sync: 2025-11-08 14:30:00
🔄 Syncing repayments created/updated after: 2025-11-08 14:30:00
✅ Incremental repayment sync complete: 234 new repayments synced
```

---

### 1.3 Sync Officer Emails Only

**Command:**
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && python3 ./scripts/sync_officer_emails.py'
```

**What it does:**
- Updates officer email addresses from Django to Seeds Metrics
- Useful when officer emails change in Django

**Execution time:** < 1 minute
**Prerequisites:** Python 3, psycopg2 library
**Run location:** Production VM

---

### 1.4 Sync Holidays (for Business Days Calculation)

**Command:**
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_holidays.go'
```

**What it does:**
- Syncs Nigerian public holidays from Django to Seeds Metrics
- Required for accurate `business_days_since_disbursement` calculation

**Execution time:** < 1 minute
**Prerequisites:** `holidays` table must exist (created by migration 026)
**Run location:** Production VM
**Recommended frequency:** Annually or when holidays change

---

## 2. Computed Fields & Metrics Recalculation

### 2.1 Recalculate All Loan Fields (Comprehensive)

**Method 1: Via API (Recommended)**
```bash
curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields
```

**Method 2: Via Database Function**
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c 'SELECT * FROM recalculate_all_loan_fields();'"
```

**What it recalculates:**
- ✅ `principal_outstanding` - Remaining principal balance
- ✅ `interest_outstanding` - Remaining interest balance
- ✅ `fees_outstanding` - Remaining fees balance
- ✅ `total_outstanding` - Sum of all outstanding balances
- ✅ `actual_outstanding` - Actual amount owed (capped at 0)
- ✅ `current_dpd` - Days Past Due (using new methodology)
- ✅ `max_dpd_ever` - Maximum DPD ever reached
- ✅ `fimr_tagged` - First Installment Missed Risk flag
- ✅ `early_indicator_tagged` - Early risk indicator (DPD 1-6)
- ✅ `repayment_delay_rate` - Percentage of on-time repayments
- ✅ `timeliness_score` - Repayment timeliness score (0-100)
- ✅ `repayment_health` - Overall repayment health score
- ✅ `days_since_last_repayment` - Days since last payment
- ✅ `total_repayments` - Sum of all repayments
- ✅ `daily_repayment_amount` - Expected daily repayment
- ✅ `repayment_days_paid` - Number of repayment days paid
- ✅ `repayment_days_due_today` - Number of repayment days due
- ✅ `business_days_since_disbursement` - Business days since loan start

**Execution time:** 5-15 seconds for ~18,000 loans
**Prerequisites:** None
**Run location:** Anywhere (API) or Production VM (SQL)
**Recommended frequency:** After bulk data imports or when metrics seem incorrect

**Output:**
```
 total_loans_processed | loans_updated | execution_time_ms
-----------------------+---------------+-------------------
                 17929 |         17929 |              8234
```

---

### 2.2 Recalculate via Frontend UI

**Steps:**
1. Navigate to https://metrics.seedsandpennies.com
2. Go to "All Loans" table
3. Click the **"Refresh Fields"** button (green button with refresh icon)
4. Wait for confirmation message

**What it does:**
- Calls the `/api/v1/loans/recalculate-fields` endpoint
- Shows success message with count of loans updated
- Automatically refreshes the table

**User-friendly:** ✅ Yes - No command line needed
**Execution time:** 5-15 seconds

---

## 3. Data Enrichment & Mapping

### 3.1 Assign Wave Classifications to Loans

**Method 1: Apply Migration (One-time setup)**
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -f /home/seeds-metrics-backend/backend/migrations/004_wave_assignment.sql"
```

**Method 2: Manual Update (Backfill existing loans)**
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
UPDATE loans l
SET wave = CASE
    WHEN (
        SELECT o.hire_date >= '2025-10-01'::DATE
        FROM officers o
        WHERE o.officer_id = l.officer_id
    ) OR l.disbursement_date >= '2025-10-20'::DATE
    THEN 'Wave 2'
    ELSE 'Wave 1'
END;
\""
```

**Wave Assignment Rules:**
- **Wave 2:** Officer hired >= 2025-10-01 OR Loan disbursed >= 2025-10-20
- **Wave 1:** All other loans

**Execution time:** < 5 seconds
**Prerequisites:** `wave` column must exist (migration 011)
**Run location:** Production VM
**Automatic:** ✅ Yes - Triggers auto-assign wave on new loans

---

### 3.2 Map Supervisors & Vertical Leads to Officers

**Command:**
```bash
./load_verticals_data.sh
```

**What it does:**
- Reads `verticals.tsv` file
- Maps supervisor_email, supervisor_name to officers
- Maps vertical_lead_email, vertical_lead_name to officers
- Maps region to officers based on branch

**Execution time:** 1-2 minutes
**Prerequisites:**
- `verticals.tsv` file must exist in current directory
- Migration 031 must be applied (adds supervisor/vertical lead columns)
- Officer mapping file at `/tmp/officer_mapping.txt`

**Run location:** Local machine (uses SSH to production)

---

### 3.3 Update Vertical Lead & Region in Loans Table

**Command:**
```bash
./update_loans_vertical_lead_and_region.sh
```

**What it does:**
- Copies `vertical_lead_name`, `vertical_lead_email`, `region` from officers table to loans table
- Ensures loans have the latest organizational hierarchy data

**SQL executed:**
```sql
UPDATE loans l
SET
    vertical_lead_name = o.vertical_lead_name,
    vertical_lead_email = o.vertical_lead_email,
    region = o.region
FROM officers o
WHERE l.officer_id = o.officer_id;
```

**Execution time:** < 5 seconds
**Prerequisites:** Officers must have vertical lead and region data populated
**Run location:** Local machine (uses SSH to production)
**Recommended frequency:** After updating officer hierarchy data

---

### 3.4 Update Regions from TSV File

**Command:**
```bash
./update_regions_from_tsv.sh
```

**What it does:**
- Reads `verticals.tsv` file
- Updates `region` column in officers table based on TSV data
- Maps Django officer emails to Seeds Metrics officer IDs

**Execution time:** 1-2 minutes
**Prerequisites:** `verticals.tsv` file must exist
**Run location:** Local machine (uses SSH to production)

---

## 4. Data Quality & Corrections

### 4.1 Fix first_payment_due_date from Django Loan Schedules

**Method 1: Using Go Script (Recommended)**
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_first_payment_due_date_only.go'
```

**Method 2: Using SQL Script (Requires dblink)**
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -f /home/seeds-metrics-backend/backend/scripts/update_first_payment_due_date.sql"
```

**What it does:**
- Fetches `start_date` from Django `loans_ajoloan` table
- Updates `first_payment_due_date` in Seeds Metrics loans table
- Fixes loans with incorrect 30-day fallback dates

**Execution time:** 2-5 minutes
**Prerequisites:** Network access to Django database
**Run location:** Production VM
**Use case:** Fix data inconsistencies like loan 19800 issue

**Output:**
```
📊 Syncing first_payment_due_date from Django...
✅ Updated 17,929 loans
📈 Summary:
   - Loans with first_payment_due_date: 17,924
   - Loans without first_payment_due_date: 5
```

---

### 4.2 Fix Repayment Amounts from Django

**Command:**
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && bash ./scripts/fix_repayment_amounts.sh'
```

**What it does:**
- Fetches correct `repayment_amount` from Django `loans_ajoloan` table
- Generates UPDATE statements for all loans
- Updates Seeds Metrics loans table with correct repayment amounts

**Execution time:** 5-10 minutes
**Prerequisites:** `.env` file with database credentials
**Run location:** Production VM

---

### 4.3 Validate and Correct Status Mappings

**Manual SQL Query:**
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
-- Find loans with status mismatches
SELECT
    l.loan_id,
    l.status as seeds_status,
    dj.status as django_status
FROM loans l
JOIN dblink(
    'host=164.90.155.2 port=5432 dbname=savings user=metricsuser password=EiRXo6IfeHQuM3wcbZ67\\\$LzwmVKCXhpUhWg sslmode=require',
    'SELECT id::VARCHAR(50), status FROM loans_ajoloan WHERE is_disbursed = TRUE'
) AS dj(loan_id VARCHAR, status VARCHAR)
ON l.loan_id = dj.loan_id
WHERE l.status != dj.status
LIMIT 20;
\""
```

**What it does:**
- Compares loan status between Django and Seeds Metrics
- Identifies mismatches for manual review
- Helps identify sync issues

**Execution time:** < 1 minute
**Prerequisites:** dblink extension installed
**Run location:** Production VM

---

### 4.4 Cap Negative Outstanding Balances

**Command:**
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && bash ./migrations/apply_cap_negative_balances.sh'
```

**What it does:**
- Applies migration 015 to cap negative outstanding balances at 0
- Ensures `actual_outstanding` is never negative (overpayments)
- Updates trigger function to prevent future negative balances

**Execution time:** < 1 minute
**Prerequisites:** None
**Run location:** Production VM
**One-time:** ✅ Yes - Only needed once

---

## 5. Complete System Refresh Sequence

**Use this sequence to bring the entire system fully up to date from scratch:**

### Step 1: Full Data Sync (30 minutes)
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_from_django.go'
```
**Syncs:** Officers, Customers, Loans, Repayments

---

### Step 2: Sync Holidays (1 minute)
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_holidays.go'
```
**Syncs:** Nigerian public holidays for business days calculation

---

### Step 3: Fix first_payment_due_date (5 minutes)
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_first_payment_due_date_only.go'
```
**Fixes:** Incorrect first payment due dates from Django schedules

---

### Step 4: Assign Wave Classifications (5 seconds)
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
UPDATE loans l
SET wave = CASE
    WHEN (SELECT o.hire_date >= '2025-10-01'::DATE FROM officers o WHERE o.officer_id = l.officer_id)
         OR l.disbursement_date >= '2025-10-20'::DATE
    THEN 'Wave 2'
    ELSE 'Wave 1'
END;
\""
```
**Assigns:** Wave 1 or Wave 2 to all loans

---

### Step 5: Load Supervisor & Vertical Lead Data (2 minutes)
```bash
./load_verticals_data.sh
```
**Updates:** Officers table with supervisor and vertical lead information

---

### Step 6: Update Loans with Vertical Lead & Region (5 seconds)
```bash
./update_loans_vertical_lead_and_region.sh
```
**Updates:** Loans table with vertical lead and region from officers

---

### Step 7: Recalculate All Computed Fields (15 seconds)
```bash
curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields
```
**Recalculates:** All 18 computed fields for all loans

---

### Step 8: Verify Data Quality (1 minute)
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
SELECT
    COUNT(*) as total_loans,
    COUNT(CASE WHEN wave IS NOT NULL THEN 1 END) as with_wave,
    COUNT(CASE WHEN vertical_lead_email IS NOT NULL THEN 1 END) as with_vertical_lead,
    COUNT(CASE WHEN region IS NOT NULL THEN 1 END) as with_region,
    COUNT(CASE WHEN first_payment_due_date IS NOT NULL THEN 1 END) as with_first_payment_due,
    ROUND(AVG(current_dpd), 2) as avg_dpd,
    ROUND(AVG(total_outstanding), 2) as avg_outstanding
FROM loans;
\""
```

**Expected Output:**
```
 total_loans | with_wave | with_vertical_lead | with_region | with_first_payment_due | avg_dpd | avg_outstanding
-------------+-----------+--------------------+-------------+------------------------+---------+-----------------
       17929 |     17929 |              17500 |       17800 |                  17924 |   12.45 |        45678.90
```

---

**Total Time:** ~40-50 minutes for complete system refresh

---

## 6. Database Connection Details

### Seeds Metrics Database (Production - Analytics)
```
Host: private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com
Port: 25060
Database: seedsmetrics
User: seedsuser
Password: @seedsuser2020
SSL Mode: require
```

**Connection String:**
```bash
PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics
```

---

### Django Database (Source of Truth)
```
Host: 164.90.155.2
Port: 5432
Database: savings
User: metricsuser
Password: EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg
SSL Mode: require
```

**Connection String:**
```bash
PGPASSWORD='EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg' psql -h 164.90.155.2 -p 5432 -U metricsuser -d savings
```

---

## 7. Maintenance Schedule Recommendations

### Daily Tasks
- ✅ **Incremental Repayments Sync** (5 minutes)
  ```bash
  ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_repayments_incremental.go'
  ```

### Weekly Tasks
- ✅ **Full Data Sync** (30 minutes) - Run on Sundays
  ```bash
  ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_from_django.go'
  ```

- ✅ **Recalculate All Fields** (15 seconds) - After full sync
  ```bash
  curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields
  ```

### Monthly Tasks
- ✅ **Update Vertical Leads & Regions** (5 minutes) - When hierarchy changes
  ```bash
  ./load_verticals_data.sh
  ./update_loans_vertical_lead_and_region.sh
  ```

### Quarterly Tasks
- ✅ **Sync Holidays** (1 minute) - Update holiday calendar
  ```bash
  ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_holidays.go'
  ```

### As-Needed Tasks
- ✅ **Fix first_payment_due_date** - When data inconsistencies are found
- ✅ **Fix Repayment Amounts** - When repayment amounts are incorrect
- ✅ **Validate Status Mappings** - When status mismatches are suspected

---

## 8. Troubleshooting

### Issue: Sync script fails with "connection refused"
**Solution:** Check database connectivity and credentials
```bash
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./cmd/api/main.go'
```

### Issue: Recalculation takes too long (> 30 seconds)
**Solution:** Check database performance and indexes
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c 'ANALYZE loans; ANALYZE repayments;'"
```

### Issue: Wave assignment not working for new loans
**Solution:** Verify trigger is installed
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
SELECT tgname, tgenabled FROM pg_trigger WHERE tgname LIKE '%wave%';
\""
```

### Issue: Vertical lead data not showing in loans
**Solution:** Run the update script
```bash
./update_loans_vertical_lead_and_region.sh
```

---

## 9. Quick Reference Commands

### Check Sync Status
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c 'SELECT * FROM sync_tracking ORDER BY last_sync_at DESC;'"
```

### Count Records by Entity
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
SELECT
    'Officers' as entity, COUNT(*) as count FROM officers
UNION ALL
SELECT 'Customers', COUNT(*) FROM customers
UNION ALL
SELECT 'Loans', COUNT(*) FROM loans
UNION ALL
SELECT 'Repayments', COUNT(*) FROM repayments;
\""
```

### Check Wave Distribution
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c 'SELECT wave, COUNT(*) as count FROM loans GROUP BY wave ORDER BY wave;'"
```

### Check Vertical Lead Coverage
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
SELECT
    COUNT(*) as total_loans,
    COUNT(vertical_lead_email) as with_vertical_lead,
    ROUND(100.0 * COUNT(vertical_lead_email) / COUNT(*), 2) as coverage_pct
FROM loans;
\""
```

### Check Data Quality Metrics
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
SELECT
    COUNT(*) as total_loans,
    COUNT(CASE WHEN current_dpd > 0 THEN 1 END) as loans_in_arrears,
    COUNT(CASE WHEN fimr_tagged = TRUE THEN 1 END) as fimr_loans,
    COUNT(CASE WHEN early_indicator_tagged = TRUE THEN 1 END) as early_indicator_loans,
    ROUND(AVG(timeliness_score), 2) as avg_timeliness_score,
    ROUND(AVG(repayment_health), 2) as avg_repayment_health,
    ROUND(SUM(actual_outstanding), 2) as total_portfolio_outstanding
FROM loans
WHERE status = 'Active';
\""
```

### Check Recent Repayments
```bash
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c \"
SELECT
    DATE(payment_date) as payment_date,
    COUNT(*) as repayment_count,
    ROUND(SUM(payment_amount), 2) as total_amount
FROM repayments
WHERE payment_date >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY DATE(payment_date)
ORDER BY payment_date DESC;
\""
```

---

## 10. API Endpoints for Automation

### Recalculate All Loan Fields
```bash
curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields \
  -H "Content-Type: application/json"
```

**Response:**
```json
{
  "status": "success",
  "message": "Successfully recalculated computed fields for all loans",
  "data": {
    "loans_updated": 17929
  }
}
```

### Get Portfolio Metrics
```bash
curl -X GET https://metrics.seedsandpennies.com/api/v1/metrics/portfolio
```

### Get All Loans (with filters)
```bash
curl -X GET "https://metrics.seedsandpennies.com/api/v1/loans?page=1&limit=100&wave=Wave%202&dpd_min=1&dpd_max=30"
```

### Health Check
```bash
curl -X GET https://metrics.seedsandpennies.com/api/v1/health
```

---

## 11. Cron Job Setup (Optional)

To automate daily/weekly syncs, add these to crontab on production VM:

```bash
# Edit crontab
ssh root@143.198.146.44 'crontab -e'
```

**Add these lines:**
```cron
# Daily incremental repayments sync at 2 AM
0 2 * * * cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_repayments_incremental.go >> /var/log/seeds-metrics-sync.log 2>&1

# Weekly full data sync on Sundays at 3 AM
0 3 * * 0 cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_from_django.go >> /var/log/seeds-metrics-sync.log 2>&1

# Recalculate fields after weekly sync (Sundays at 4 AM)
0 4 * * 0 curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields >> /var/log/seeds-metrics-recalc.log 2>&1
```

**View logs:**
```bash
ssh root@143.198.146.44 'tail -f /var/log/seeds-metrics-sync.log'
```

---

## 12. Emergency Recovery Procedures

### Scenario 1: Database Connection Lost
```bash
# 1. Check database status
ssh root@143.198.146.44 "PGPASSWORD='@seedsuser2020' psql -h private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U seedsuser -d seedsmetrics -c 'SELECT 1;'"

# 2. Restart API service
ssh root@143.198.146.44 'systemctl restart seeds-metrics-api.service'

# 3. Check service status
ssh root@143.198.146.44 'systemctl status seeds-metrics-api.service'
```

### Scenario 2: Data Corruption Detected
```bash
# 1. Stop API service
ssh root@143.198.146.44 'systemctl stop seeds-metrics-api.service'

# 2. Run full data sync
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_from_django.go'

# 3. Recalculate all fields
curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields

# 4. Restart API service
ssh root@143.198.146.44 'systemctl start seeds-metrics-api.service'
```

### Scenario 3: Metrics Showing Incorrect Values
```bash
# 1. Recalculate all computed fields
curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields

# 2. If still incorrect, sync first_payment_due_date
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run ./scripts/sync_first_payment_due_date_only.go'

# 3. Recalculate again
curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields
```

---

**End of Guide**

For questions or issues, contact the Seeds Metrics development team.
