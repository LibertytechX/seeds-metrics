# Setup External Database - Run These Commands

## ✅ Configuration Complete - Just Run These 3 Steps

---

## Step 1: Create Database

```bash
psql -h localhost -p 5433 -U postgres -d postgres
```

Then paste this command:

```sql
CREATE DATABASE seedsmetrics OWNER postgres;
\q
```

---

## Step 2: Apply Database Schema

```bash
cd /Users/manager/Documents/Liberty/seeds-metrics
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -f backend/migrations/001_initial_schema.sql
```

---

## Step 3: Start the Application

```bash
cd backend
docker-compose down -v
docker-compose up -d --build
```

---

## ✅ Verify Everything Works

```bash
# Test API health
curl http://localhost:8080/health

# Should return:
# {
#   "status": "healthy",
#   "services": {
#     "database": {
#       "status": "healthy"
#     }
#   }
# }
```

---

## 🎯 Load Test Data (Optional)

```bash
bash backend/test-fimr-simple.sh
```

---

## 📋 Database Configuration

- **Host:** `localhost` (from Mac) / `host.docker.internal` (from Docker)
- **Port:** `5433`
- **Database:** `seedsmetrics`
- **User:** `postgres`
- **Password:** `19sedimat54`

---

## 🔧 Quick Commands

```bash
# Connect to database
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics

# View tables
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -c "\dt"

# View API logs
cd backend && docker-compose logs -f api

# Restart API
cd backend && docker-compose restart api
```

---

**That's it! Your backend is now configured to use the external PostgreSQL database `seedsmetrics` on port 5433.** 🚀

