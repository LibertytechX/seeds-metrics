# Final Setup - External PostgreSQL Database

## ✅ All Configuration Files Updated

The backend is now configured to use the **existing `postgres` user** with your external PostgreSQL database.

---

## 📋 Database Configuration

| Setting | Value |
|---------|-------|
| **Database User** | `postgres` |
| **Database Password** | `19sedimat54` |
| **Database Name** | `seedsmetrics` |
| **Database Host (from Mac)** | `localhost` |
| **Database Host (from Docker)** | `host.docker.internal` |
| **Database Port** | `5433` |

---

## 🚀 Setup Steps (2 Steps Only!)

Since the `postgres` user already exists, you only need to create the database and apply the schema.

### **Step 1: Create Database**

```bash
psql -h localhost -p 5433 -U postgres -d postgres
```

Then run:
```sql
CREATE DATABASE seedsmetrics OWNER postgres;
\q
```

### **Step 2: Apply Schema**

```bash
cd /Users/manager/Documents/Liberty/seeds-metrics
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -f backend/migrations/001_initial_schema.sql
```

### **Step 3: Start Application**

```bash
cd backend
docker-compose down -v
docker-compose up -d --build
```

### **Step 4: Verify**

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "services": {
    "database": {
      "status": "healthy"
    }
  }
}
```

---

## 📁 Files Updated

All configuration files have been updated to use `postgres` as the database user:

1. ✅ **`backend/.env`** - DB_USER=postgres
2. ✅ **`backend/docker-compose.yml`** - DB_USER: postgres
3. ✅ **`backend/setup-database.sql`** - Creates database with postgres owner (no user creation)
4. ✅ **`backend/SETUP_NOW.md`** - Updated all commands
5. ✅ **`backend/SETUP_COMMANDS.md`** - Updated all commands
6. ✅ **`backend/QUICK_START.md`** - Updated all commands

---

## 🔧 Quick Commands

```bash
# Connect to database
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics

# View tables
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -c "\dt"

# Count loans
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -c "SELECT COUNT(*) FROM loans;"

# View API logs
cd backend && docker-compose logs -f api

# Restart API
cd backend && docker-compose restart api
```

---

## 🎯 Load Test Data (Optional)

After the application is running, load test data:

```bash
bash backend/test-fimr-simple.sh
```

This will create:
- 2 customers (Inyang Kpongette, Shamsideen Allamu)
- 2 loans with different repayment patterns
- Loan schedules
- Repayment records

---

## ✅ Summary

**Configuration Complete!**

- ✅ Using existing `postgres` user (no need to create new user)
- ✅ Database name: `seedsmetrics`
- ✅ Password: `19sedimat54`
- ✅ Port: `5433`
- ✅ All configuration files updated

**Just run the 3 steps above to get started!** 🚀

