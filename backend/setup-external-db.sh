#!/bin/bash

echo "=========================================="
echo "External PostgreSQL Database Setup"
echo "=========================================="
echo ""

# Configuration
DB_HOST="localhost"
DB_PORT="5433"
DB_USER="analytics_user"
DB_PASSWORD="19sedimat54"
DB_NAME="analytics_db"
POSTGRES_SUPERUSER="postgres"
POSTGRES_PASSWORD=""  # Leave empty if no password, or set your postgres user password

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}This script will:${NC}"
echo "  1. Check if PostgreSQL is running on port 5433"
echo "  2. Create database user 'analytics_user' (if needed)"
echo "  3. Create database 'analytics_db' (if needed)"
echo "  4. Apply database schema"
echo "  5. Verify the setup"
echo ""
read -p "Continue? (y/n) " -n 1 -r
echo ""
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Setup cancelled."
    exit 0
fi
echo ""

# Step 1: Check PostgreSQL
echo "Step 1: Checking PostgreSQL..."
if lsof -i :5433 > /dev/null 2>&1; then
    echo -e "${GREEN}✅ PostgreSQL is running on port 5433${NC}"
else
    echo -e "${RED}❌ PostgreSQL is NOT running on port 5433${NC}"
    echo ""
    echo "Please start PostgreSQL on port 5433 first."
    echo ""
    echo "If you're using Homebrew:"
    echo "  brew services start postgresql@14"
    echo ""
    echo "Or check your PostgreSQL configuration to ensure it's listening on port 5433"
    exit 1
fi
echo ""

# Step 2: Create user (if needed)
echo "Step 2: Creating database user..."
export PGPASSWORD=""
USER_EXISTS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname='$DB_USER';" 2>/dev/null)

if [ "$USER_EXISTS" = "1" ]; then
    echo -e "${YELLOW}⚠️  User '$DB_USER' already exists${NC}"
    read -p "Update password? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -c "ALTER USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" 2>&1
        echo -e "${GREEN}✅ Password updated${NC}"
    fi
else
    echo "Creating user '$DB_USER'..."
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ User created successfully${NC}"
    else
        echo -e "${RED}❌ Failed to create user${NC}"
        echo "You may need to run this script with a PostgreSQL superuser"
        exit 1
    fi
fi
echo ""

# Step 3: Create database (if needed)
echo "Step 3: Creating database..."
DB_EXISTS=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DB_NAME';" 2>/dev/null)

if [ "$DB_EXISTS" = "1" ]; then
    echo -e "${YELLOW}⚠️  Database '$DB_NAME' already exists${NC}"
    read -p "Drop and recreate? (y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Dropping database..."
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -c "DROP DATABASE $DB_NAME;" 2>&1
        echo "Creating database..."
        psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>&1
        echo -e "${GREEN}✅ Database recreated${NC}"
    fi
else
    echo "Creating database '$DB_NAME'..."
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Database created successfully${NC}"
    else
        echo -e "${RED}❌ Failed to create database${NC}"
        exit 1
    fi
fi
echo ""

# Grant privileges
echo "Granting privileges..."
psql -h "$DB_HOST" -p "$DB_PORT" -U "$POSTGRES_SUPERUSER" -d postgres -c "GRANT ALL PRIVILEGES ON DATABASE $DB_NAME TO $DB_USER;" 2>&1
echo ""

# Step 4: Apply schema
echo "Step 4: Applying database schema..."
export PGPASSWORD="$DB_PASSWORD"

if [ -f "migrations/001_initial_schema.sql" ]; then
    echo "Applying migrations/001_initial_schema.sql..."
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -f migrations/001_initial_schema.sql 2>&1 | grep -E "(CREATE|ERROR|NOTICE)" | head -20

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Schema applied successfully${NC}"
    else
        echo -e "${RED}❌ Failed to apply schema${NC}"
        exit 1
    fi
else
    echo -e "${RED}❌ Migration file not found: migrations/001_initial_schema.sql${NC}"
    exit 1
fi
echo ""

# Step 5: Verify setup
echo "Step 5: Verifying setup..."
TABLE_COUNT=$(psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -tAc "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public';")
echo -e "${GREEN}✅ Found $TABLE_COUNT tables${NC}"

echo ""
echo "Tables created:"
psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "\dt" | grep public | awk '{print "  - " $3}'
echo ""

# Summary
echo "=========================================="
echo -e "${GREEN}✅ Setup Complete!${NC}"
echo "=========================================="
echo ""
echo "Database Configuration:"
echo "  Host: $DB_HOST"
echo "  Port: $DB_PORT"
echo "  Database: $DB_NAME"
echo "  User: $DB_USER"
echo "  Password: $DB_PASSWORD"
echo ""
echo "Connection String:"
echo "  postgresql://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME"
echo ""
echo "Next Steps:"
echo "  1. Start the API container:"
echo "     cd backend && docker-compose up -d --build"
echo ""
echo "  2. Test the API:"
echo "     curl http://localhost:8080/health"
echo ""
echo "  3. Load test data:"
echo "     bash backend/test-fimr-simple.sh"
echo ""
echo "  4. Access the frontend:"
echo "     http://localhost:5173/"
echo ""

unset PGPASSWORD

