# External PostgreSQL Database - Setup Instructions

## 🎯 Quick Start Guide

Follow these steps to configure your backend to use the external PostgreSQL database on `localhost:5433`.

---

## Step 1: Create Database User and Database

You need to create the database user and database in your PostgreSQL instance. Connect to PostgreSQL as a superuser (usually `postgres`):

### Option A: Using psql command line

```bash
# Connect to PostgreSQL
psql -h localhost -p 5433 -U postgres -d postgres

# Then run these SQL commands:
CREATE USER analytics_user WITH PASSWORD '19sedimat54';
CREATE DATABASE analytics_db OWNER analytics_user;
GRANT ALL PRIVILEGES ON DATABASE analytics_db TO analytics_user;

# Exit psql
\q
```

### Option B: Using a PostgreSQL GUI (pgAdmin, DBeaver, etc.)

1. Connect to your PostgreSQL server on `localhost:5433`
2. Create a new user:
   - Username: `analytics_user`
   - Password: `19sedimat54`
3. Create a new database:
   - Database name: `analytics_db`
   - Owner: `analytics_user`

---

## Step 2: Apply Database Schema

Once the database is created, apply the schema:

```bash
cd /Users/manager/Documents/Liberty/seeds-metrics

# Apply the migration script
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U analytics_user -d analytics_db -f backend/migrations/001_initial_schema.sql
```

This will create all the necessary tables, triggers, and functions.

---

## Step 3: Verify Database Setup

Test the connection:

```bash
# Test connection
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U analytics_user -d analytics_db -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';"

# List tables
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U analytics_user -d analytics_db -c "\dt"
```

You should see tables like: `loans`, `repayments`, `customers`, `officers`, `loan_schedule`, etc.

---

## Step 4: Stop Old Docker Containers

Stop and remove the old containers (including the old PostgreSQL container):

```bash
cd backend
docker-compose down -v
```

This will:
- Stop all containers
- Remove containers
- Remove the old `postgres_data` volume (we don't need it anymore)

---

## Step 5: Start the Application

Start the application with the new configuration:

```bash
cd backend
docker-compose up -d --build
```

This will:
- Build the API container with the new configuration
- Start the Redis container
- Start the API container (which will connect to your external PostgreSQL)

---

## Step 6: Verify the Connection

### Check API Health

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "timestamp": "2025-10-19T...",
  "services": {
    "database": {
      "status": "healthy",
      "message": "Database connection is healthy"
    }
  }
}
```

### Check Docker Logs

```bash
cd backend
docker-compose logs api | tail -50
```

Look for messages like:
- "Database connection established"
- "Server started on :8080"

---

## Step 7: Load Test Data (Optional)

If you want to test with the FIMR demo data:

```bash
bash backend/test-fimr-simple.sh
```

This will create:
- 2 customers
- 2 loans (Inyang Kpongette and Shamsideen Allamu)
- Loan schedules
- Repayment records

---

## 🔧 Configuration Files

### `backend/.env` (Created)

```env
DB_HOST=host.docker.internal
DB_PORT=5433
DB_USER=analytics_user
DB_PASSWORD=19sedimat54
DB_NAME=analytics_db
```

### `backend/docker-compose.yml` (Modified)

- PostgreSQL container: **Removed** (commented out)
- API container: **Updated** to use external database
- Redis container: **Preserved** (still running in Docker)

---

## 🐛 Troubleshooting

### Issue: "Connection refused"

**Check if PostgreSQL is running:**
```bash
lsof -i :5433
```

**Check PostgreSQL logs:**
```bash
# If using Homebrew
tail -f /usr/local/var/log/postgresql@14.log
```

### Issue: "Password authentication failed"

**Verify user exists:**
```bash
psql -h localhost -p 5433 -U postgres -d postgres -c "\du"
```

**Reset password:**
```sql
ALTER USER analytics_user WITH PASSWORD '19sedimat54';
```

### Issue: "Database does not exist"

**List databases:**
```bash
psql -h localhost -p 5433 -U postgres -d postgres -c "\l"
```

**Create database:**
```sql
CREATE DATABASE analytics_db OWNER analytics_user;
```

### Issue: "host.docker.internal not resolving"

This is a Docker Desktop feature. If you're on Linux, you may need to:

1. Find your host IP:
   ```bash
   ip addr show docker0 | grep inet
   ```

2. Update `backend/.env`:
   ```env
   DB_HOST=172.17.0.1  # Use your actual host IP
   ```

### Issue: "Permission denied on schema public"

Grant schema permissions:
```sql
GRANT ALL ON SCHEMA public TO analytics_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO analytics_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO analytics_user;
```

---

## 📊 What Changed?

### Before (Docker PostgreSQL):
```
┌─────────────────────────────────────┐
│         Docker Network              │
│                                     │
│  ┌──────────┐      ┌──────────┐   │
│  │   API    │─────▶│ Postgres │   │
│  │Container │      │Container │   │
│  └──────────┘      └──────────┘   │
│                                     │
│  ┌──────────┐                      │
│  │  Redis   │                      │
│  │Container │                      │
│  └──────────┘                      │
└─────────────────────────────────────┘
```

### After (External PostgreSQL):
```
┌─────────────────────────────────────┐
│         Docker Network              │
│                                     │
│  ┌──────────┐                      │
│  │   API    │──┐                   │
│  │Container │  │                   │
│  └──────────┘  │                   │
│                │                   │
│  ┌──────────┐  │                   │
│  │  Redis   │  │                   │
│  │Container │  │                   │
│  └──────────┘  │                   │
└─────────────────┼───────────────────┘
                  │
                  │ host.docker.internal:5433
                  │
                  ▼
         ┌──────────────┐
         │  PostgreSQL  │
         │ (Host:5433)  │
         └──────────────┘
              MacBook
```

---

## ✅ Verification Checklist

- [ ] PostgreSQL is running on port 5433
- [ ] User `analytics_user` exists with password `19sedimat54`
- [ ] Database `analytics_db` exists and is owned by `analytics_user`
- [ ] Database schema is applied (tables exist)
- [ ] Old Docker containers are stopped (`docker-compose down -v`)
- [ ] New API container is running (`docker-compose up -d --build`)
- [ ] API health check returns "healthy" (`curl http://localhost:8080/health`)
- [ ] Frontend can access the API (`http://localhost:5173/`)

---

## 🚀 Quick Command Reference

```bash
# 1. Create user and database (run in psql as superuser)
CREATE USER analytics_user WITH PASSWORD '19sedimat54';
CREATE DATABASE analytics_db OWNER analytics_user;
GRANT ALL PRIVILEGES ON DATABASE analytics_db TO analytics_user;

# 2. Apply schema
cd /Users/manager/Documents/Liberty/seeds-metrics
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U analytics_user -d analytics_db -f backend/migrations/001_initial_schema.sql

# 3. Restart application
cd backend
docker-compose down -v
docker-compose up -d --build

# 4. Test
curl http://localhost:8080/health
bash backend/test-frontend-integration.sh

# 5. Load test data
bash backend/test-fimr-simple.sh
```

---

## 📝 Notes

- **Data Persistence**: Your data is now stored in the external PostgreSQL database and will persist even when Docker containers are removed.
- **Backups**: You can backup the database using standard PostgreSQL tools (`pg_dump`).
- **Access**: You can access the database directly from your Mac using any PostgreSQL client.
- **Performance**: External database may have slightly different performance characteristics compared to containerized database.

---

## 🔄 Reverting to Docker PostgreSQL

If you want to go back to using Docker PostgreSQL:

1. Uncomment the `postgres` service in `docker-compose.yml`
2. Update `backend/.env`:
   ```env
   DB_HOST=postgres
   DB_PORT=5432
   DB_PASSWORD=analytics_password
   ```
3. Restart:
   ```bash
   docker-compose down -v
   docker-compose up -d --build
   ```

