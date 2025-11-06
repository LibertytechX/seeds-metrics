# Holiday Table Sync Documentation

## Overview

The holiday table has been successfully synced from the Django database (`loans_holiday` table in the `savings` database) to the Seeds Metrics database. This document describes the sync process, schema, and deployment workflow.

## Schema

### Holiday Table Structure

The `holiday` table in Seeds Metrics mirrors the `loans_holiday` table from Django:

| Column | Type | Nullable | Description |
|--------|------|----------|-------------|
| id | BIGINT | NO | Primary key from Django |
| date | DATE | YES | Holiday date |
| name | VARCHAR(255) | YES | Holiday name (e.g., Good Friday, Easter Monday) |
| created_at | TIMESTAMP WITH TIME ZONE | NO | Record creation timestamp |
| updated_at | TIMESTAMP WITH TIME ZONE | NO | Record update timestamp |
| agent_id | BIGINT | YES | Agent ID if agent-specific holiday |
| branch_id | BIGINT | YES | Branch ID if branch-specific holiday |
| created_by_id | BIGINT | YES | User ID who created the record |
| type | VARCHAR(255) | NO | Holiday type: 'company' or agent-specific |
| salary_waver | BOOLEAN | NO | Whether salary is waived on this holiday |

### Indexes

The following indexes are created for optimal query performance:

- `idx_holiday_date` - On `date` column for date range queries
- `idx_holiday_type` - On `type` column for filtering by holiday type
- `idx_holiday_agent_id` - On `agent_id` column for agent-specific queries
- `idx_holiday_branch_id` - On `branch_id` column for branch-specific queries
- `idx_holiday_created_at` - On `created_at` column for temporal queries

## Data Statistics

**Sync Date**: 2025-11-06 00:07:27 UTC

| Metric | Value |
|--------|-------|
| Total Holidays | 1,015 |
| Company Holidays | 107 |
| Agent-Specific Holidays | 908 |
| Salary Waver Holidays | 30 |
| Date Range | 2024-03-29 to 2025-10-06 |

## Migration

### Migration File

**File**: `backend/migrations/026_create_holiday_table.sql`

The migration creates the holiday table with all necessary columns, constraints, and indexes. It is idempotent and can be safely re-run.

### Applying the Migration

```bash
# On production server
psql "postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require" < backend/migrations/026_create_holiday_table.sql
```

## Sync Script

### Script File

**File**: `backend/scripts/sync_holidays.go`

The sync script performs a full sync of all holidays from Django to Seeds Metrics. It:

1. Connects to both Django and Seeds Metrics databases
2. Fetches all holidays from Django's `loans_holiday` table
3. Truncates the existing `holiday` table in Seeds Metrics
4. Inserts all holidays from Django
5. Verifies the sync by comparing record counts

### Building the Script

```bash
cd /home/seeds-metrics-backend/backend
/usr/local/go/bin/go build -o bin/sync_holidays ./scripts/sync_holidays.go
```

### Running the Script

```bash
cd /home/seeds-metrics-backend/backend

# Set environment variables
export DB_HOST=generaldb-do-user-9489371-0.k.db.ondigitalocean.com
export DB_PORT=25060
export DB_USER=seedsuser
export DB_PASSWORD=@seedsuser2020
export DB_NAME=seedsmetrics
export DB_SSLMODE=require
export DJANGO_DB_HOST=164.90.155.2
export DJANGO_DB_PORT=5432
export DJANGO_DB_USER=metricsuser
export DJANGO_DB_PASSWORD='EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg'
export DJANGO_DB_NAME=savings
export DJANGO_DB_SSLMODE=require

# Run the sync
timeout 300 ./bin/sync_holidays
```

### Expected Output

```
🚀 Starting holiday sync from Django to SeedsMetrics...
✅ Connected to SeedsMetrics database
✅ Connected to Django database

📊 Syncing Holidays...
📥 Fetching holidays from Django database...
📊 Found 1015 holidays in Django database
🗑️  Clearing existing holidays in SeedsMetrics...
📤 Inserting holidays into SeedsMetrics...

✅ Sync completed in 1.146632827s
📊 Successfully inserted: 1015 holidays
✅ Verification: 1015 holidays now in SeedsMetrics database

✅ Holiday sync completed successfully!
```

## Deployment Workflow

### Initial Deployment

1. **Apply Migration**
   ```bash
   psql "postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require" < backend/migrations/026_create_holiday_table.sql
   ```

2. **Build Sync Script**
   ```bash
   cd /home/seeds-metrics-backend/backend
   /usr/local/go/bin/go build -o bin/sync_holidays ./scripts/sync_holidays.go
   ```

3. **Run Initial Sync**
   ```bash
   cd /home/seeds-metrics-backend/backend
   timeout 300 ./bin/sync_holidays
   ```

4. **Verify Data**
   ```bash
   psql "postgresql://seedsuser:%40seedsuser2020@generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require" -c "SELECT COUNT(*) FROM holiday;"
   ```

### Periodic Syncs

The holiday table should be synced periodically (e.g., weekly or monthly) to capture any new holidays added in Django:

```bash
# Add to crontab for weekly sync (every Sunday at 2 AM)
0 2 * * 0 cd /home/seeds-metrics-backend/backend && timeout 300 ./bin/sync_holidays >> /var/log/holiday-sync.log 2>&1
```

## Verification Queries

### Count Holidays by Type

```sql
SELECT type, COUNT(*) as count
FROM holiday
GROUP BY type
ORDER BY count DESC;
```

### Find Holidays in Date Range

```sql
SELECT date, name, type, salary_waver
FROM holiday
WHERE date BETWEEN '2025-01-01' AND '2025-12-31'
ORDER BY date;
```

### Find Agent-Specific Holidays

```sql
SELECT date, name, agent_id, branch_id
FROM holiday
WHERE agent_id IS NOT NULL
ORDER BY date;
```

### Find Salary Waver Holidays

```sql
SELECT date, name, type
FROM holiday
WHERE salary_waver = true
ORDER BY date;
```

## Sync Status

✅ **COMPLETED AND DEPLOYED**

- Migration 026 applied successfully
- Sync script built and tested
- All 1,015 holidays synced from Django to Seeds Metrics
- Data integrity verified
- Ready for production use

## Notes

- The sync is a full sync (truncate and reload) rather than incremental, as holidays are relatively static
- The script can be safely re-run multiple times without data loss
- All timestamps are preserved from the Django database
- Agent-specific and branch-specific holidays are fully supported

