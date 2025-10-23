#!/bin/bash

echo "=========================================="
echo "External PostgreSQL Database Test"
echo "=========================================="
echo ""

# Configuration
DB_HOST="localhost"
DB_PORT="5433"
DB_USER="analytics_user"
DB_PASSWORD="19sedimat54"
DB_NAME="analytics_db"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test 1: Check if PostgreSQL is running on port 5433
echo "1. Checking if PostgreSQL is running on port 5433..."
if lsof -i :5433 > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PostgreSQL is running on port 5433${NC}"
else
    echo -e "${RED}❌ PostgreSQL is NOT running on port 5433${NC}"
    echo "   Please start PostgreSQL on port 5433"
    exit 1
fi
echo ""

# Test 2: Test connection to PostgreSQL
echo "2. Testing connection to PostgreSQL..."
export PGPASSWORD="$DB_PASSWORD"
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "SELECT version();" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Successfully connected to PostgreSQL${NC}"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "SELECT version();" | head -3
else
    echo -e "${RED}❌ Failed to connect to PostgreSQL${NC}"
    echo "   Check username, password, and PostgreSQL configuration"
    exit 1
fi
echo ""

# Test 3: Check if database exists
echo "3. Checking if database '$DB_NAME' exists..."
DB_EXISTS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME';")
if [ "$DB_EXISTS" = "1" ]; then
    echo -e "${GREEN}✅ Database '$DB_NAME' exists${NC}"
else
    echo -e "${YELLOW}⚠️  Database '$DB_NAME' does not exist${NC}"
    echo "   Creating database..."
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Database created successfully${NC}"
    else
        echo -e "${RED}❌ Failed to create database${NC}"
        echo "   You may need to create it manually as a superuser"
    fi
fi
echo ""

# Test 4: Check if user has access to database
echo "4. Testing access to database '$DB_NAME'..."
if psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT current_database();" > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Successfully connected to database '$DB_NAME'${NC}"
else
    echo -e "${RED}❌ Failed to connect to database '$DB_NAME'${NC}"
    exit 1
fi
echo ""

# Test 5: Check if schema is applied
echo "5. Checking if database schema is applied..."
TABLE_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
if [ "$TABLE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ Database schema is applied ($TABLE_COUNT tables found)${NC}"
    echo "   Tables:"
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt" | grep public | awk '{print "   - " $3}'
else
    echo -e "${YELLOW}⚠️  Database schema is NOT applied (no tables found)${NC}"
    echo "   You need to apply the migration script:"
    echo "   PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f migrations/001_initial_schema.sql"
fi
echo ""

# Test 6: Check data in database
echo "6. Checking data in database..."
if [ "$TABLE_COUNT" -gt 0 ]; then
    LOAN_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM loans;" 2>/dev/null || echo "0")
    OFFICER_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM officers;" 2>/dev/null || echo "0")
    CUSTOMER_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM customers;" 2>/dev/null || echo "0")
    
    echo -e "${GREEN}✅ Data summary:${NC}"
    echo "   - Loans: $LOAN_COUNT"
    echo "   - Officers: $OFFICER_COUNT"
    echo "   - Customers: $CUSTOMER_COUNT"
else
    echo -e "${YELLOW}⚠️  Skipping data check (no tables)${NC}"
fi
echo ""

# Test 7: Test connection from Docker perspective
echo "7. Testing Docker container access to host database..."
echo "   Using host.docker.internal:5433..."

# Create a temporary test script
cat > /tmp/test-docker-db.sh << 'EOF'
#!/bin/bash
export PGPASSWORD="19sedimat54"
psql -h host.docker.internal -p 5433 -U analytics_user -d analytics_db -c "SELECT 'Docker can connect!' as status;" 2>&1
EOF
chmod +x /tmp/test-docker-db.sh

# Check if Docker is running
if docker info > /dev/null 2>&1; then
    # Try to run test in a PostgreSQL client container
    if docker run --rm --add-host=host.docker.internal:host-gateway -e PGPASSWORD="$DB_PASSWORD" postgres:14-alpine psql -h host.docker.internal -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "SELECT 'Docker can connect!' as status;" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Docker containers can access host database${NC}"
    else
        echo -e "${YELLOW}⚠️  Docker containers may have issues accessing host database${NC}"
        echo "   This is expected if the database doesn't exist yet"
        echo "   The API container will use 'extra_hosts' configuration to access the host"
    fi
else
    echo -e "${YELLOW}⚠️  Docker is not running, skipping Docker test${NC}"
fi
echo ""

# Test 8: Check if API container is running
echo "8. Checking if API container is running..."
if docker ps | grep -q analytics-api; then
    echo -e "${GREEN}✅ API container is running${NC}"
    
    # Test API health endpoint
    echo "   Testing API health endpoint..."
    sleep 2
    HEALTH_RESPONSE=$(curl -s http://localhost:8080/health)
    if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
        echo -e "${GREEN}✅ API is healthy and connected to database${NC}"
        echo "$HEALTH_RESPONSE" | jq . 2>/dev/null || echo "$HEALTH_RESPONSE"
    else
        echo -e "${RED}❌ API health check failed${NC}"
        echo "$HEALTH_RESPONSE"
    fi
else
    echo -e "${YELLOW}⚠️  API container is not running${NC}"
    echo "   Start it with: cd backend && docker-compose up -d --build"
fi
echo ""

# Summary
echo "=========================================="
echo "Summary"
echo "=========================================="
echo ""
echo "Database Configuration:"
echo "  Host: $DB_HOST (from host) / host.docker.internal (from Docker)"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo ""

if [ "$TABLE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ External database is configured and ready!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Start the API: cd backend && docker-compose up -d --build"
    echo "  2. Test the API: curl http://localhost:8080/health"
    echo "  3. Load test data: bash backend/test-fimr-simple.sh"
else
    echo -e "${YELLOW}⚠️  Database exists but schema is not applied${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Apply schema: PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -f backend/migrations/001_initial_schema.sql"
    echo "  2. Start the API: cd backend && docker-compose up -d --build"
    echo "  3. Test the API: curl http://localhost:8080/health"
    echo "  4. Load test data: bash backend/test-fimr-simple.sh"
fi
echo ""

unset PGPASSWORD

