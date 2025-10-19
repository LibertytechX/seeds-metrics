# 🚀 Quick Start Guide

Get the Analytics Backend running in **5 minutes**!

---

## Option 1: Docker (Recommended) ⭐

### **Step 1: Start Services**

```bash
cd backend
docker-compose up -d
```

This starts:
- ✅ PostgreSQL (with schema auto-created)
- ✅ Redis
- ✅ Analytics API

### **Step 2: Verify**

```bash
# Check health
curl http://localhost:8080/health

# Expected output:
# {"status":"healthy","timestamp":"...","services":{"database":{"status":"healthy",...}}}
```

### **Step 3: Test API**

```bash
# Run test script
./scripts/test-api.sh

# Or manually test
curl -X POST http://localhost:8080/api/v1/etl/loans \
  -H "Content-Type: application/json" \
  -d '{
    "loan_id": "LN2024001",
    "customer_id": "CUST001",
    "customer_name": "Test Customer",
    "officer_id": "OFF001",
    "officer_name": "Test Officer",
    "region": "South West",
    "branch": "Lagos Main",
    "loan_amount": 500000.00,
    "disbursement_date": "2024-10-15",
    "maturity_date": "2025-04-15",
    "loan_term_days": 180,
    "interest_rate": 0.15,
    "fee_amount": 25000.00,
    "channel": "Direct",
    "status": "Active"
  }'
```

### **Step 4: View Logs**

```bash
# All services
docker-compose logs -f

# API only
docker-compose logs -f api
```

### **Step 5: Stop Services**

```bash
docker-compose down

# Stop and delete data
docker-compose down -v
```

---

## Option 2: Local Development

### **Prerequisites**

- Go 1.21+
- PostgreSQL 14+
- Redis 7+

### **Step 1: Install Dependencies**

```bash
cd backend
go mod download
```

### **Step 2: Setup Database**

```bash
# Create database
createdb analytics_db

# Run migrations
psql -U postgres -d analytics_db -f migrations/001_initial_schema.sql
```

### **Step 3: Configure Environment**

```bash
cp .env.example .env

# Edit .env
# Change DB_HOST=localhost, REDIS_HOST=localhost
```

### **Step 4: Run API**

```bash
go run cmd/api/main.go

# Or build and run
make build
./bin/analytics-api
```

---

## 🧪 Testing

### **Using Makefile**

```bash
# Start Docker services
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down

# Reset database
make db-reset
```

### **Using Test Script**

```bash
./scripts/test-api.sh
```

### **Manual Testing**

```bash
# Health check
curl http://localhost:8080/health

# Create loan
curl -X POST http://localhost:8080/api/v1/etl/loans \
  -H "Content-Type: application/json" \
  -d @test-data/loan.json

# Create repayment
curl -X POST http://localhost:8080/api/v1/etl/repayments \
  -H "Content-Type: application/json" \
  -d @test-data/repayment.json
```

---

## 📊 Database Access

```bash
# Using Docker
docker exec -it analytics-postgres psql -U analytics_user -d analytics_db

# Local
psql -U analytics_user -d analytics_db

# Useful queries
SELECT * FROM loans LIMIT 10;
SELECT * FROM repayments LIMIT 10;
SELECT loan_id, total_principal_paid, principal_outstanding FROM loans;
```

---

## 🔧 Common Commands

```bash
# Build
make build

# Run locally
make run

# Start Docker
make docker-up

# Stop Docker
make docker-down

# View logs
make docker-logs

# Reset database
make db-reset

# Format code
make fmt

# Run tests
make test
```

---

## 🐛 Troubleshooting

### **Port 8080 already in use**

```bash
# Change port in docker-compose.yml
ports:
  - "8081:8080"
```

### **Database connection failed**

```bash
# Check if PostgreSQL is running
docker-compose ps postgres

# Restart database
docker-compose restart postgres
```

### **Migrations not applied**

```bash
# Manually run migrations
docker exec -it analytics-postgres psql -U analytics_user -d analytics_db -f /docker-entrypoint-initdb.d/001_initial_schema.sql
```

---

## 📚 Next Steps

1. **Read the full README**: `backend/README.md`
2. **Review API documentation**: `ETL_PAYLOAD_EXAMPLES.md`
3. **Understand the architecture**: `BACKEND_ARCHITECTURE.md`
4. **Connect your frontend**: Update API URL in React app

---

## ✅ Success Checklist

- [ ] Docker services running (`docker-compose ps`)
- [ ] Health check returns 200 (`curl http://localhost:8080/health`)
- [ ] Can create loans (`POST /api/v1/etl/loans`)
- [ ] Can create repayments (`POST /api/v1/etl/repayments`)
- [ ] Database triggers working (check computed fields)
- [ ] Logs show no errors (`docker-compose logs`)

---

**You're ready to go! 🎉**

