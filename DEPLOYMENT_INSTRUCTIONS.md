# 🚀 Production Deployment Instructions

## Quick Deployment

The easiest way to deploy is using the automated deployment script:

```bash
./deploy.sh
```

This script will:
1. ✅ Check SSH connection to production server
2. ✅ Build backend binary for Linux
3. ✅ Build frontend for production
4. ✅ Copy files to production server
5. ✅ Restart backend service
6. ✅ Verify deployment

---

## Manual Deployment (Step-by-Step)

If you prefer to deploy manually or the script doesn't work, follow these steps:

### Prerequisites

- SSH access to production server: `metrics.seedsandpennies.com`
- SSH key configured for passwordless login
- Git repository pushed to GitHub

---

### Step 1: SSH into Production Server

```bash
ssh root@metrics.seedsandpennies.com
```

---

### Step 2: Pull Latest Code

```bash
cd /var/www/seeds-metrics
git pull origin main
```

---

### Step 3: Deploy Backend

```bash
# Navigate to backend directory
cd /var/www/seeds-metrics/backend

# Build the Go binary
go build -o bin/api ./cmd/api

# Stop existing backend process
pkill -f "bin/api" || true

# Start new backend process
nohup ./bin/api > logs/api.log 2>&1 &

# Verify backend is running
curl http://localhost:8080/api/v1/metrics/portfolio
```

---

### Step 4: Deploy Frontend

```bash
# Navigate to frontend directory
cd /var/www/seeds-metrics/metrics-dashboard

# Install dependencies (if package.json changed)
npm install

# Build for production
npm run build

# The dist folder is automatically served by Nginx
# No additional steps needed
```

---

### Step 5: Run Database Migrations (if needed)

```bash
cd /var/www/seeds-metrics/backend/migrations

# Make migration script executable
chmod +x apply_all_fixes.sh

# Run migrations
./apply_all_fixes.sh
```

---

### Step 6: Verify Deployment

```bash
# Check backend API
curl https://metrics.seedsandpennies.com/api/v1/metrics/portfolio

# Check frontend
curl https://metrics.seedsandpennies.com/

# Check Swagger docs
curl https://metrics.seedsandpennies.com/swagger/index.html
```

---

## What Was Deployed

### Backend Changes:
- ✅ New portfolio metrics calculations
- ✅ Active vs Inactive loans metrics
- ✅ ROT (Risk of Termination) analysis
- ✅ Portfolio delinquency risk metrics
- ✅ Portfolio repayment behavior metrics
- ✅ New repository method: `GetPortfolioLoanMetrics()`
- ✅ Updated metrics service for at-risk officers

### Frontend Changes:
- ✅ Removed "Average DQI" card from Portfolio Metrics
- ✅ Added 4 new metric cards:
  - Active vs Inactive Loans (with filtering buttons)
  - ROT Analysis (with Early/Late ROT filters)
  - Portfolio Delinquency Risk (with at-risk officers filter)
  - Portfolio Repayment Behavior (informational)
- ✅ Fixed filtering buttons to properly update AllLoans component
- ✅ Added useEffect to watch for filter prop changes
- ✅ Added key prop to force component remount
- ✅ Implemented client-side filtering for active/inactive and ROT loans
- ✅ Added CSS styles for interactive filter buttons

---

## Troubleshooting

### Backend Not Starting

```bash
# Check logs
tail -f /var/www/seeds-metrics/backend/logs/api.log

# Check if port is in use
lsof -i :8080

# Kill existing process
pkill -f "bin/api"

# Restart
cd /var/www/seeds-metrics/backend
nohup ./bin/api > logs/api.log 2>&1 &
```

### Frontend Not Updating

```bash
# Clear browser cache
# Or use Ctrl+Shift+R (hard refresh)

# Check Nginx configuration
nginx -t

# Restart Nginx
systemctl restart nginx

# Check Nginx logs
tail -f /var/log/nginx/error.log
```

### Database Connection Issues

```bash
# Check database connection
psql -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 -U doadmin -d seedsmetrics

# Verify environment variables
cat /var/www/seeds-metrics/backend/.env
```

---

## Rollback Procedure

If something goes wrong, you can rollback to the previous version:

```bash
# SSH into server
ssh root@metrics.seedsandpennies.com

# Navigate to project directory
cd /var/www/seeds-metrics

# Checkout previous commit
git log --oneline -5  # Find the previous commit hash
git checkout <previous-commit-hash>

# Rebuild backend
cd backend
go build -o bin/api ./cmd/api
pkill -f "bin/api"
nohup ./bin/api > logs/api.log 2>&1 &

# Rebuild frontend
cd ../metrics-dashboard
npm run build

# Verify
curl https://metrics.seedsandpennies.com/api/v1/metrics/portfolio
```

---

## Production URLs

- **Frontend**: https://metrics.seedsandpennies.com
- **Backend API**: https://metrics.seedsandpennies.com/api/v1
- **Swagger Docs**: https://metrics.seedsandpennies.com/swagger/index.html
- **Portfolio Metrics**: https://metrics.seedsandpennies.com/api/v1/metrics/portfolio
- **Officers**: https://metrics.seedsandpennies.com/api/v1/officers
- **Loans**: https://metrics.seedsandpennies.com/api/v1/loans

---

## Post-Deployment Checklist

- [ ] Backend API responding (check `/api/v1/metrics/portfolio`)
- [ ] Frontend loading correctly
- [ ] New Portfolio Metrics cards visible
- [ ] "Average DQI" card removed
- [ ] Filtering buttons working (Active/Inactive, ROT)
- [ ] All Loans view filtering correctly
- [ ] No console errors in browser
- [ ] Swagger documentation accessible
- [ ] Database migrations applied (if any)

---

## Support

If you encounter issues during deployment:

1. Check the logs:
   - Backend: `/var/www/seeds-metrics/backend/logs/api.log`
   - Nginx: `/var/log/nginx/error.log`

2. Verify services are running:
   ```bash
   pgrep -f "bin/api"  # Backend process
   systemctl status nginx  # Nginx status
   ```

3. Test API endpoints:
   ```bash
   curl https://metrics.seedsandpennies.com/api/v1/metrics/portfolio | jq '.'
   ```

---

**Deployment completed successfully! 🎉**

