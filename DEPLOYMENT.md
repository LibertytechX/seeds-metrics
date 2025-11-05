# Seeds Metrics - Deployment Guide

This document provides comprehensive deployment instructions for the Seeds Metrics application, covering both backend (Go API) and frontend (React/Vite) components.

---

## 📋 Table of Contents

1. [Infrastructure Overview](#infrastructure-overview)
2. [Prerequisites](#prerequisites)
3. [Backend Deployment](#backend-deployment)
4. [Frontend Deployment](#frontend-deployment)
5. [Database Migrations](#database-migrations)
6. [Troubleshooting](#troubleshooting)
7. [Rollback Procedures](#rollback-procedures)

---

## 🏗️ Infrastructure Overview

### Production Environment

| Component | Details |
|-----------|---------|
| **Server** | DigitalOcean Droplet |
| **IP Address** | `143.198.146.44` |
| **Domain** | `metrics.seedsandpennies.com` |
| **OS** | Ubuntu Linux |
| **User** | `root` |

### Application Stack

| Layer | Technology | Port/Location |
|-------|-----------|---------------|
| **Frontend** | React (Vite) | `/home/seeds-metrics-frontend/` |
| **Web Server** | NGINX | Port 80/443 (SSL) |
| **Backend API** | Go (Gin framework) | Port 8080 (localhost) |
| **Service Manager** | systemd | `seeds-metrics-api.service` |
| **Database** | PostgreSQL (DigitalOcean Managed) | Port 25060 |

### Database Configuration

**SeedsMetrics Database (Analytics)**
- **Host**: `private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com`
- **Port**: `25060`
- **Database**: `seedsmetrics`
- **User**: `metricsuser`
- **Purpose**: Analytics database (read-write)

**Django Database (Source of Truth)**
- **Host**: `164.90.155.2`
- **Port**: `5432`
- **Database**: `liberty_db`
- **User**: `liberty_user`
- **Purpose**: Source database (read-only)

### Repository

- **GitHub**: `git@github.com:LibertytechX/seeds-metrics.git`
- **Branch**: `main`

---

## ✅ Prerequisites

### Local Development Machine

1. **Git** - For version control
2. **Go 1.21+** - For backend development
3. **Node.js 18+** - For frontend development
4. **SSH Access** - SSH key configured for `root@143.198.146.44`

### Production Server

1. **Go Compiler** - Installed at `/usr/local/go/bin/go`
2. **NGINX** - Web server with SSL certificates
3. **systemd** - Service manager for backend API
4. **Git** - For pulling code updates

---

## 🔧 Backend Deployment

### Deployment Strategy

**⚠️ IMPORTANT**: Do NOT upload compiled binaries. Always build on the server.

The backend binary is ~28MB and network uploads are unreliable. Instead:
1. Commit code changes to Git
2. Push to GitHub
3. Pull changes on production server
4. Build binary directly on server

### Step-by-Step Backend Deployment

#### 1. Commit and Push Changes

```bash
# From local development machine
cd /path/to/seeds-metrics

# Check status
git status

# Add changes
git add backend/

# Commit with descriptive message
git commit -m "Description of changes"

# Push to GitHub
git push origin main
```

#### 2. SSH to Production Server

```bash
ssh root@143.198.146.44
```

#### 3. Navigate to Backend Directory

```bash
cd /home/seeds-metrics-backend/backend
```

#### 4. Pull Latest Changes

```bash
# Pull from GitHub
git pull origin main

# If there are local changes, stash them first
git stash
git pull origin main
```

#### 5. Build the Backend Binary

```bash
# Build using Go compiler
/usr/local/go/bin/go build -o seeds-metrics-api ./cmd/api

# Verify build succeeded
ls -lh seeds-metrics-api
```

#### 6. Backup Old Binary (Optional but Recommended)

```bash
# Create timestamped backup
mv api api.old.$(date +%Y%m%d_%H%M%S)
```

#### 7. Deploy New Binary

```bash
# Move new binary to production location
mv seeds-metrics-api api

# Make executable
chmod +x api
```

#### 8. Restart Backend Service

```bash
# Restart the systemd service
sudo systemctl restart seeds-metrics-api

# Wait a few seconds for startup
sleep 3

# Check service status
sudo systemctl status seeds-metrics-api --no-pager
```

#### 9. Verify Deployment

```bash
# Check logs
sudo journalctl -u seeds-metrics-api -n 50 --no-pager

# Test API endpoint
curl -s http://localhost:8080/api/v1/health | jq '.'

# Test from external domain
curl -s https://metrics.seedsandpennies.com/api/v1/health | jq '.'
```

### One-Command Backend Deployment

```bash
# From local machine (after committing and pushing)
ssh root@143.198.146.44 "cd /home/seeds-metrics-backend/backend && \
  git pull origin main && \
  /usr/local/go/bin/go build -o seeds-metrics-api ./cmd/api && \
  mv api api.old.\$(date +%Y%m%d_%H%M%S) && \
  mv seeds-metrics-api api && \
  chmod +x api && \
  sudo systemctl restart seeds-metrics-api && \
  sleep 3 && \
  sudo systemctl status seeds-metrics-api --no-pager"
```

---

## 🎨 Frontend Deployment

### Step-by-Step Frontend Deployment

#### 1. Build Frontend Locally

```bash
# From local development machine
cd /path/to/seeds-metrics/metrics-dashboard

# Install dependencies (if needed)
npm install

# Build production bundle
npm run build
```

This creates optimized files in the `dist/` directory.

#### 2. Deploy to Production Server

```bash
# Use rsync to upload built files
rsync -avz --delete dist/ root@143.198.146.44:/home/seeds-metrics-frontend/
```

**Flags explained:**
- `-a` - Archive mode (preserves permissions, timestamps)
- `-v` - Verbose output
- `-z` - Compress during transfer
- `--delete` - Remove files on server that don't exist locally

#### 3. Verify Deployment

```bash
# Check deployed files
ssh root@143.198.146.44 "ls -lh /home/seeds-metrics-frontend/"

# Test the website
curl -I https://metrics.seedsandpennies.com
```

#### 4. Clear Browser Cache

After deployment, users may need to hard refresh their browsers:
- **Chrome/Firefox**: `Ctrl + Shift + R` (Windows/Linux) or `Cmd + Shift + R` (Mac)
- **Safari**: `Cmd + Option + R`

### Frontend-Only Deployment Script

```bash
# From local machine
cd /path/to/seeds-metrics/metrics-dashboard && \
  npm run build && \
  rsync -avz --delete dist/ root@143.198.146.44:/home/seeds-metrics-frontend/
```

---

## 🗄️ Database Migrations

### Migration Files Location

```
backend/migrations/
├── 001_initial_schema.sql
├── 002_add_indexes.sql
├── 017_add_user_type_to_officers.sql
└── ...
```

### Applying Migrations

#### 1. Upload Migration File

```bash
# From local machine
scp backend/migrations/017_add_user_type_to_officers.sql \
  root@143.198.146.44:/home/seeds-metrics-backend/backend/migrations/
```

#### 2. Connect to Database

```bash
# SSH to server
ssh root@143.198.146.44

# Connect to PostgreSQL
psql "postgresql://metricsuser:PASSWORD@private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require"
```

#### 3. Run Migration

```sql
-- Check current schema
\dt

-- Run migration file
\i /home/seeds-metrics-backend/backend/migrations/017_add_user_type_to_officers.sql

-- Verify changes
\d officers
```

#### 4. Test Migration

```sql
-- Check if column exists
SELECT column_name, data_type 
FROM information_schema.columns 
WHERE table_name = 'officers' AND column_name = 'user_type';

-- Check data
SELECT user_type, COUNT(*) 
FROM officers 
WHERE user_type IS NOT NULL 
GROUP BY user_type;
```

### Running Data Sync Scripts

After schema changes, sync data from Django database:

```bash
# SSH to server
ssh root@143.198.146.44

# Navigate to backend directory
cd /home/seeds-metrics-backend/backend

# Run sync script
/usr/local/go/bin/go run scripts/sync_from_django.go
```

---

## 🔍 Troubleshooting

### Backend Issues

#### Service Won't Start

```bash
# Check service status
sudo systemctl status seeds-metrics-api

# View recent logs
sudo journalctl -u seeds-metrics-api -n 100 --no-pager

# Check if port is in use
sudo lsof -i :8080

# Test binary directly
cd /home/seeds-metrics-backend/backend
./api
```

#### Database Connection Errors

```bash
# Test database connectivity
psql "postgresql://metricsuser:PASSWORD@private-generaldb-do-user-9489371-0.k.db.ondigitalocean.com:25060/seedsmetrics?sslmode=require" -c "SELECT 1;"

# Check environment variables
cat /home/seeds-metrics-backend/backend/.env
```

#### Build Failures

```bash
# Check Go version
/usr/local/go/bin/go version

# Clean build cache
/usr/local/go/bin/go clean -cache

# Rebuild with verbose output
/usr/local/go/bin/go build -v -o seeds-metrics-api ./cmd/api
```

### Frontend Issues

#### Blank Page After Deployment

1. Check browser console for errors
2. Verify API URL in environment config
3. Check NGINX configuration
4. Clear browser cache

#### API Calls Failing

```bash
# Check NGINX configuration
sudo nginx -t

# View NGINX error logs
sudo tail -f /var/log/nginx/error.log

# Check API proxy settings
sudo cat /etc/nginx/sites-available/metrics.seedsandpennies.com
```

---

## ⏮️ Rollback Procedures

### Backend Rollback

```bash
# SSH to server
ssh root@143.198.146.44
cd /home/seeds-metrics-backend/backend

# List backup binaries
ls -lh api.old.*

# Restore previous version
cp api.old.20250105_073000 api

# Restart service
sudo systemctl restart seeds-metrics-api
```

### Frontend Rollback

```bash
# From local machine, checkout previous commit
git log --oneline -10
git checkout <previous-commit-hash>

# Rebuild and redeploy
cd metrics-dashboard
npm run build
rsync -avz --delete dist/ root@143.198.146.44:/home/seeds-metrics-frontend/

# Return to latest
git checkout main
```

### Database Rollback

```sql
-- Connect to database
psql "postgresql://metricsuser:PASSWORD@..."

-- Drop column (example)
ALTER TABLE officers DROP COLUMN IF EXISTS user_type;

-- Restore from backup (if available)
-- Contact database administrator
```

---

## 📞 Support

For deployment issues or questions:
- Check application logs: `sudo journalctl -u seeds-metrics-api -f`
- Review NGINX logs: `sudo tail -f /var/log/nginx/error.log`
- Verify service status: `sudo systemctl status seeds-metrics-api`

---

**Last Updated**: 2025-11-05

