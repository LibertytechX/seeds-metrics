#!/bin/bash

# Loan Metrics Validation Test Runner
# This script runs the loan metrics validation test suite

set -e

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Loan Metrics Validation Test Runner${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if Python is installed
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}ERROR: Python 3 is not installed${NC}"
    exit 1
fi

# Check if psycopg2 is installed
if ! python3 -c "import psycopg2" 2>/dev/null; then
    echo -e "${YELLOW}WARNING: psycopg2 is not installed${NC}"
    echo -e "${YELLOW}Installing psycopg2-binary...${NC}"
    pip3 install psycopg2-binary
fi

# Set default environment variables if not already set
export DJANGO_DB_HOST="${DJANGO_DB_HOST:-164.90.155.2}"
export DJANGO_DB_PORT="${DJANGO_DB_PORT:-5432}"
export DJANGO_DB_NAME="${DJANGO_DB_NAME:-savings}"
export DJANGO_DB_USER="${DJANGO_DB_USER:-metricsuser}"
export DJANGO_DB_PASSWORD="${DJANGO_DB_PASSWORD:-EiRXo6IfeHQuM3wcbZ67\$LzwmVKCXhpUhWg}"

export SEEDSMETRICS_DB_HOST="${SEEDSMETRICS_DB_HOST:-generaldb-do-user-9489371-0.k.db.ondigitalocean.com}"
export SEEDSMETRICS_DB_PORT="${SEEDSMETRICS_DB_PORT:-25060}"
export SEEDSMETRICS_DB_NAME="${SEEDSMETRICS_DB_NAME:-seedsmetrics}"
export SEEDSMETRICS_DB_USER="${SEEDSMETRICS_DB_USER:-seedsuser}"
export SEEDSMETRICS_DB_PASSWORD="${SEEDSMETRICS_DB_PASSWORD:-@seedsuser2020}"

# Get the directory of this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Run the test suite
echo -e "${BLUE}Running test suite...${NC}"
echo ""

python3 "${SCRIPT_DIR}/test_loan_metrics_validation.py"

# Capture exit code
EXIT_CODE=$?

echo ""
if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo -e "${GREEN}========================================${NC}"
else
    echo -e "${RED}========================================${NC}"
    echo -e "${RED}✗ Some tests failed${NC}"
    echo -e "${RED}========================================${NC}"
fi

exit $EXIT_CODE

