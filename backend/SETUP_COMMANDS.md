# Setup Commands - Copy and Paste

## Step 1: Create Database

Run this command:

```bash
psql -h localhost -p 5433 -U postgres -d postgres -f backend/setup-database.sql
```

**Or** run this command directly in psql:

```bash
psql -h localhost -p 5433 -U postgres -d postgres
```

Then paste:

```sql
CREATE DATABASE seedsmetrics OWNER postgres;
\q
```

---

## Step 2: Apply Database Schema

```bash
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -f backend/migrations/001_initial_schema.sql
```

---

## Step 3: Verify Database Setup

```bash
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -c "\dt"
```

You should see tables like: `loans`, `repayments`, `customers`, `officers`, etc.

---

## Step 4: Restart Application

```bash
cd backend
docker-compose down -v
docker-compose up -d --build
```

---

## Step 5: Test API

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

## Step 6: Load Test Data (Optional)

```bash
bash backend/test-fimr-simple.sh
```

---

## All Commands in One Block

```bash
# 1. Create database (run in psql)
psql -h localhost -p 5433 -U postgres -d postgres -f backend/setup-database.sql

# 2. Apply schema
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -f backend/migrations/001_initial_schema.sql

# 3. Verify
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -c "\dt"

# 4. Restart application
cd backend && docker-compose down -v && docker-compose up -d --build

# 5. Test
curl http://localhost:8080/health

# 6. Load test data
bash backend/test-fimr-simple.sh
```

---

## Troubleshooting

### If postgres user requires password:

Set the password environment variable first:
```bash
export PGPASSWORD=your_postgres_password
psql -h localhost -p 5433 -U postgres -d postgres -f backend/setup-database.sql
unset PGPASSWORD
```

### If postgres user has no password (common on Mac):

The commands above should work as-is.

### Check if PostgreSQL is running:

```bash
lsof -i :5433
```

### Check existing databases:

```bash
psql -h localhost -p 5433 -U postgres -d postgres -c "\l"
```

### Check existing users:

```bash
psql -h localhost -p 5433 -U postgres -d postgres -c "\du"
```

