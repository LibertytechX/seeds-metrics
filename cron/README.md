# Seeds Metrics Cron Jobs

This directory contains cron job configurations for automated data synchronization and maintenance tasks.

## Installation

To install the cron jobs on the production server:

```bash
# SSH to production server
ssh root@143.198.146.44

# Navigate to the repository
cd /home/seeds-metrics-backend

# Pull latest changes
git pull origin main

# Install the crontab
crontab cron/seeds-metrics-crontab

# Verify installation
crontab -l
```

## Cron Jobs Overview

### Job 1: Incremental Repayments Sync
- **Schedule**: Every 6 hours (00:00, 06:00, 12:00, 18:00 UTC)
- **Duration**: ~6 minutes
- **Log**: `/var/log/seeds-metrics-repayments-sync.log`
- **Purpose**: Sync new repayments from Django to Seeds Metrics

### Job 2: Recalculate Computed Fields
- **Schedule**: Every 6 hours (00:10, 06:10, 12:10, 18:10 UTC)
- **Duration**: < 1 minute
- **Log**: `/var/log/seeds-metrics-recalculate.log`
- **Purpose**: Recalculate all loan metrics after repayments sync
- **Dependency**: Runs 10 minutes after Job 1

### Job 3: Update Loans with Vertical Lead Data
- **Schedule**: Daily at 03:00 UTC
- **Duration**: < 1 minute
- **Log**: `/var/log/seeds-metrics-vertical-update.log`
- **Purpose**: Propagate officer hierarchy changes to loans

### Job 4: Full Data Sync
- **Schedule**: Weekly on Sunday at 01:00 UTC
- **Duration**: ~63 minutes
- **Log**: `/var/log/seeds-metrics-full-sync.log`
- **Purpose**: Full sync of officers, customers, loans, and repayments

## Monitoring

Check cron job logs:

```bash
# View recent repayments sync logs
tail -f /var/log/seeds-metrics-repayments-sync.log

# View recent recalculation logs
tail -f /var/log/seeds-metrics-recalculate.log

# View recent vertical lead update logs
tail -f /var/log/seeds-metrics-vertical-update.log

# View recent full sync logs
tail -f /var/log/seeds-metrics-full-sync.log

# Check all logs for errors
grep -i error /var/log/seeds-metrics-*.log
```

## Troubleshooting

### Check if cron jobs are running
```bash
# List current crontab
crontab -l

# Check cron service status
systemctl status cron
```

### Manually trigger a job
```bash
# Incremental repayments sync
cd /home/seeds-metrics-backend/backend && source .env && /usr/local/go/bin/go run scripts/sync_repayments_incremental.go

# Recalculate computed fields
curl -X POST https://metrics.seedsandpennies.com/api/v1/loans/recalculate-fields

# Full data sync
cd /home/seeds-metrics-backend/backend && /usr/local/go/bin/go run scripts/sync_from_django.go
```

## Log Rotation

To prevent log files from growing too large, consider setting up log rotation:

```bash
# Create logrotate configuration
sudo nano /etc/logrotate.d/seeds-metrics

# Add the following content:
/var/log/seeds-metrics-*.log {
    weekly
    rotate 4
    compress
    missingok
    notifempty
}
```

## Maintenance

### Updating Cron Jobs

1. Edit `cron/seeds-metrics-crontab` in the repository
2. Commit and push changes to git
3. SSH to production server
4. Pull latest changes: `git pull origin main`
5. Reinstall crontab: `crontab cron/seeds-metrics-crontab`
6. Verify: `crontab -l`

### Disabling Cron Jobs

```bash
# Remove all cron jobs
crontab -r

# Or comment out specific jobs in the crontab file
```

## Notes

- All times are in UTC (production server timezone)
- Jobs are staggered to avoid database contention
- Each job logs to a separate file for easy troubleshooting
- Environment variables are sourced from `/home/seeds-metrics-backend/backend/.env`

