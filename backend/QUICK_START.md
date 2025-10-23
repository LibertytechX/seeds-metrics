# Quick Start - External PostgreSQL Database

## 🚀 3-Step Setup

### Step 1: Create Database (Run in psql)

```bash
# Connect to PostgreSQL as superuser
psql -h localhost -p 5433 -U postgres -d postgres

# Run this command:
CREATE DATABASE seedsmetrics OWNER postgres;
\q
```

### Step 2: Apply Schema

```bash
cd /Users/manager/Documents/Liberty/seeds-metrics
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -f backend/migrations/001_initial_schema.sql
```

### Step 3: Start Application

```bash
cd backend
docker-compose down -v
docker-compose up -d --build
```

---

## ✅ Verify

```bash
# Test database
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -c "\dt"

# Test API
curl http://localhost:8080/health

# Load test data (optional)
bash backend/test-fimr-simple.sh
```

---

## 📋 Configuration Summary

| Setting | Value |
|---------|-------|
| **Database Host** | `localhost` (from Mac) / `host.docker.internal` (from Docker) |
| **Database Port** | `5433` |
| **Database Name** | `seedsmetrics` |
| **Database User** | `postgres` |
| **Database Password** | `19sedimat54` |
| **API URL** | `http://localhost:8080` |
| **Frontend URL** | `http://localhost:5173` |

---

## 🔧 Useful Commands

```bash
# Test database connection
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics

# View tables
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -c "\dt"

# Count loans
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -c "SELECT COUNT(*) FROM loans;"

# Restart API
cd backend && docker-compose restart api

# View API logs
cd backend && docker-compose logs -f api

# Stop everything
cd backend && docker-compose down

# Start everything
cd backend && docker-compose up -d
```

---

## 📚 Documentation

- **Setup Instructions**: `backend/SETUP_INSTRUCTIONS.md`
- **Technical Details**: `backend/EXTERNAL_DATABASE_SETUP.md`
- **Migration Summary**: `EXTERNAL_DATABASE_MIGRATION_SUMMARY.md`

---

## 🐛 Troubleshooting

**Can't connect to database?**
```bash
# Check PostgreSQL is running
lsof -i :5433

# Check user exists
psql -h localhost -p 5433 -U postgres -c "\du"
```

**API won't start?**
```bash
# Check logs
cd backend && docker-compose logs api

# Rebuild
cd backend && docker-compose up -d --build
```

**Need help?**
```bash
# Run diagnostic test
bash backend/test-external-db.sh
```

