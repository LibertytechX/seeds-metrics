#!/bin/bash

echo "=========================================="
echo "Quick External Database Setup"
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
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}This will set up the external PostgreSQL database for the analytics application.${NC}"
echo ""

# Step 1: Create user and database
echo "Step 1: Creating database user and database..."
echo ""

psql -h "$DB_HOST" -p "$DB_PORT" -U postgres -d postgres << EOF
-- Create user if not exists
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '$DB_USER') THEN
    CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
    RAISE NOTICE 'User $DB_USER created';
  ELSE
    ALTER USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
    RAISE NOTICE 'User $DB_USER already exists, password updated';
  END IF;
END
\$\$;

-- Create database if not exists
SELECT 'CREATE DATABASE $DB_NAME OWNER $DB_USER'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$DB_NAME')\gexec

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;

\echo ''
\echo '✅ Database user and database ready!'
\echo ''
EOF

if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Failed to create user/database${NC}"
    echo ""
    echo "Please ensure:"
    echo "  1. PostgreSQL is running on port 5433"
    echo "  2. You can connect as user 'postgres'"
    echo ""
    exit 1
fi

echo ""

# Step 2: Apply schema
echo "Step 2: Applying database schema..."
echo ""

export PGPASSWORD="$DB_PASSWORD"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f migrations/001_initial_schema.sql 2>&1 | grep -E "(CREATE|ERROR|NOTICE|Table)" | head -30

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ Schema applied successfully${NC}"
else
    echo ""
    echo -e "${RED}❌ Failed to apply schema${NC}"
    exit 1
fi

echo ""

# Step 3: Verify
echo "Step 3: Verifying setup..."
echo ""

TABLE_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")

echo -e "${GREEN}✅ Found $TABLE_COUNT tables${NC}"
echo ""
echo "Tables created:"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt" 2>/dev/null | grep public | awk '{print "  ✓ " $3}'

echo ""
echo "=========================================="
echo -e "${GREEN}✅ Database Setup Complete!${NC}"
echo "=========================================="
echo ""
echo "Database Details:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo ""
echo "Next Steps:"
echo "  1. Start the application:"
echo "     ${BLUE}cd backend && docker-compose down -v && docker-compose up -d --build${NC}"
echo ""
echo "  2. Test the API:"
echo "     ${BLUE}curl http://localhost:8080/health${NC}"
echo ""
echo "  3. Load test data (optional):"
echo "     ${BLUE}bash backend/test-fimr-simple.sh${NC}"
echo ""

unset PGPASSWORD

