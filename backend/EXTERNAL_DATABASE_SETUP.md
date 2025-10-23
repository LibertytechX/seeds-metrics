# External PostgreSQL Database Setup

## Overview

The backend application has been reconfigured to connect to an **external PostgreSQL database** running on your MacBook instead of using a Docker containerized database.

---

## Configuration Changes

### 1. **Database Connection Details**

**External PostgreSQL Database:**
- **Host:** `localhost` (from host machine) / `host.docker.internal` (from Docker container)
- **Port:** `5433`
- **Database:** `analytics_db`
- **User:** `analytics_user`
- **Password:** `19sedimat54`

### 2. **Files Modified**

#### **`backend/.env`** (NEW)
Created environment configuration file with external database settings:
```env
DB_HOST=host.docker.internal
DB_PORT=5433
DB_USER=analytics_user
DB_PASSWORD=19sedimat54
DB_NAME=analytics_db
```

#### **`backend/docker-compose.yml`** (MODIFIED)
- **Removed:** PostgreSQL container (commented out)
- **Removed:** `postgres_data` volume
- **Updated:** API service to use `.env` file
- **Added:** `extra_hosts` configuration for `host.docker.internal`
- **Removed:** Dependency on PostgreSQL container health check

---

## How It Works

### **Docker Container → Host Machine Database**

When the Go API runs inside a Docker container, it **cannot** use `localhost` to access services on the host machine. Instead, it uses:

```
host.docker.internal:5433
```

This special DNS name resolves to the host machine's IP address from within the Docker container.

### **Local Process → Host Machine Database**

If you run the Go API directly on your Mac (not in Docker), it would use:

```
localhost:5433
```

---

## Prerequisites

### 1. **Ensure PostgreSQL is Running on Port 5433**

Check if PostgreSQL is running:
```bash
psql -h localhost -p 5433 -U postgres -c "SELECT version();"
```

### 2. **Create Database and User**

Connect to your PostgreSQL instance and run:
```sql
-- Create user
CREATE USER analytics_user WITH PASSWORD '19sedimat54';

-- Create database
CREATE DATABASE analytics_db OWNER analytics_user;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE analytics_db TO analytics_user;
```

### 3. **Apply Database Schema**

Run the migration script to create tables:
```bash
psql -h localhost -p 5433 -U analytics_user -d analytics_db -f backend/migrations/001_initial_schema.sql
```

Or if you need to provide the password:
```bash
PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U analytics_user -d analytics_db -f backend/migrations/001_initial_schema.sql
```

---

## Starting the Application

### **Option 1: Using Docker Compose (Recommended)**

```bash
cd backend
docker-compose down -v  # Stop and remove old containers
docker-compose up -d --build
```

The API container will connect to your external PostgreSQL database on `host.docker.internal:5433`.

### **Option 2: Running Locally (Without Docker)**

If you want to run the Go API directly on your Mac:

1. Update `.env` to use `localhost` instead of `host.docker.internal`:
   ```env
   DB_HOST=localhost
   ```

2. Build and run:
   ```bash
   cd backend
   go build -o bin/analytics-api cmd/api/main.go
   ./bin/analytics-api
   ```

---

## Testing the Connection

### **1. Test Database Connection from Host**

```bash
psql -h localhost -p 5433 -U analytics_user -d analytics_db -c "SELECT COUNT(*) FROM loans;"
```

### **2. Test API Health Endpoint**

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

### **3. Run Integration Tests**

```bash
bash backend/test-frontend-integration.sh
```

---

## Troubleshooting

### **Issue: "Connection refused" or "Cannot connect to database"**

**Possible Causes:**
1. PostgreSQL is not running on port 5433
2. PostgreSQL is not accepting connections from Docker
3. Firewall blocking connections

**Solutions:**

1. **Check PostgreSQL is running:**
   ```bash
   lsof -i :5433
   ```

2. **Check PostgreSQL configuration** (`postgresql.conf`):
   ```
   listen_addresses = '*'  # or 'localhost'
   port = 5433
   ```

3. **Check PostgreSQL host-based authentication** (`pg_hba.conf`):
   ```
   # Allow connections from Docker
   host    all             all             172.16.0.0/12           md5
   host    all             all             127.0.0.1/32            md5
   ```

4. **Restart PostgreSQL:**
   ```bash
   brew services restart postgresql@14  # or your PostgreSQL version
   ```

### **Issue: "Database does not exist"**

Create the database:
```bash
psql -h localhost -p 5433 -U postgres -c "CREATE DATABASE analytics_db OWNER analytics_user;"
```

### **Issue: "User does not exist"**

Create the user:
```bash
psql -h localhost -p 5433 -U postgres -c "CREATE USER analytics_user WITH PASSWORD '19sedimat54';"
```

### **Issue: "host.docker.internal not resolving"**

On Linux, you may need to use the host's IP address instead:
```bash
# Find your host IP
ip addr show docker0 | grep inet

# Update .env
DB_HOST=172.17.0.1  # or your host IP
```

---

## Data Persistence

### **Advantages of External Database:**
- ✅ Data persists even when Docker containers are removed
- ✅ Can access data directly from host machine
- ✅ Easier to backup and restore
- ✅ Can use existing PostgreSQL tools and GUIs
- ✅ Shared database across multiple applications

### **Backup Database:**
```bash
pg_dump -h localhost -p 5433 -U analytics_user -d analytics_db > backup.sql
```

### **Restore Database:**
```bash
psql -h localhost -p 5433 -U analytics_user -d analytics_db < backup.sql
```

---

## Reverting to Docker PostgreSQL

If you want to go back to using the Docker containerized database:

1. Uncomment the `postgres` service in `docker-compose.yml`
2. Update `.env`:
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

---

## Summary

✅ **PostgreSQL container removed** - Using external database on host machine  
✅ **Connection configured** - API connects to `host.docker.internal:5433`  
✅ **Redis container preserved** - Still running in Docker  
✅ **Data persistence** - Data survives container restarts  
✅ **Environment variables** - Configured in `backend/.env`  

**Next Steps:**
1. Ensure PostgreSQL is running on port 5433
2. Create database and user if they don't exist
3. Apply schema migrations
4. Start the application with `docker-compose up -d --build`
5. Test the connection with health check endpoint

