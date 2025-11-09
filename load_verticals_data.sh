#!/bin/bash

# Script to load supervisor and vertical lead data from verticals.tsv into Seeds Metrics database
# Date: 2025-11-09
# Usage: ./load_verticals_data.sh

set -e  # Exit on error

echo "=========================================="
echo "Loading Verticals Data to Officers Table"
echo "=========================================="
echo ""

# File paths
VERTICALS_FILE="verticals.tsv"
MAPPING_FILE="/tmp/officer_mapping.txt"
TEMP_SQL="/tmp/update_officers_verticals.sql"

# Check if files exist
if [ ! -f "$VERTICALS_FILE" ]; then
    echo "❌ Error: $VERTICALS_FILE not found"
    exit 1
fi

if [ ! -f "$MAPPING_FILE" ]; then
    echo "❌ Error: $MAPPING_FILE not found"
    echo "Please run the officer mapping script first"
    exit 1
fi

echo "📁 Files found:"
echo "   - verticals.tsv: $(wc -l < $VERTICALS_FILE) lines"
echo "   - officer_mapping.txt: $(wc -l < $MAPPING_FILE) lines"
echo ""

# Create SQL update statements
echo "🔨 Generating SQL UPDATE statements..."
echo "-- Auto-generated SQL to update officers with supervisor and vertical lead data" > $TEMP_SQL
echo "-- Generated: $(date)" >> $TEMP_SQL
echo "" >> $TEMP_SQL

# Counter for updates
UPDATE_COUNT=0

# Read verticals.tsv (skip header) and generate UPDATE statements
tail -n +2 "$VERTICALS_FILE" | while IFS=$'\t' read -r loan_officer_id loan_officer_email loan_officer_name loan_officer_phone branch_name branch_location branch_sub_location branch_state branch_supervisor_email branch_supervisor_name region_name vertical_lead_email vertical_lead_name; do
    
    # Find the Seeds Metrics officer_id for this email
    SEEDS_OFFICER_ID=$(grep -i "|${loan_officer_email}|" "$MAPPING_FILE" | cut -d'|' -f3 | head -1)
    
    if [ -z "$SEEDS_OFFICER_ID" ]; then
        # Officer not found in mapping - skip
        continue
    fi
    
    # Escape single quotes in names for SQL
    branch_supervisor_name_escaped=$(echo "$branch_supervisor_name" | sed "s/'/''/g")
    vertical_lead_name_escaped=$(echo "$vertical_lead_name" | sed "s/'/''/g")
    
    # Generate UPDATE statement
    cat >> $TEMP_SQL << EOF
UPDATE officers 
SET 
    supervisor_email = '$branch_supervisor_email',
    supervisor_name = '$branch_supervisor_name_escaped',
    vertical_lead_email = '$vertical_lead_email',
    vertical_lead_name = '$vertical_lead_name_escaped',
    updated_at = CURRENT_TIMESTAMP
WHERE officer_id = '$SEEDS_OFFICER_ID';

EOF
    
    UPDATE_COUNT=$((UPDATE_COUNT + 1))
done

echo "✅ Generated $UPDATE_COUNT UPDATE statements"
echo ""

# Show sample of SQL
echo "📄 Sample SQL (first 5 updates):"
head -20 $TEMP_SQL
echo "..."
echo ""

# Ask for confirmation
echo "⚠️  This will update $UPDATE_COUNT officers in the production database."
echo ""
read -p "Do you want to proceed? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "❌ Aborted by user"
    exit 1
fi

echo ""
echo "🚀 Executing SQL updates on production database..."

# Execute SQL on production database
ssh root@143.198.146.44 "cd /home/seeds-metrics-backend/backend && source .env && psql \"host=\$DB_HOST port=\$DB_PORT dbname=\$DB_NAME user=\$DB_USER password=\$DB_PASSWORD sslmode=require\" -f - 2>&1" < $TEMP_SQL

echo ""
echo "✅ SQL execution complete!"
echo ""

# Verify the updates
echo "🔍 Verifying updates..."
VERIFICATION_SQL="SELECT 
    COUNT(*) as total_officers,
    COUNT(supervisor_email) as with_supervisor,
    COUNT(vertical_lead_email) as with_vertical_lead,
    ROUND(100.0 * COUNT(supervisor_email) / COUNT(*), 2) as supervisor_coverage_pct,
    ROUND(100.0 * COUNT(vertical_lead_email) / COUNT(*), 2) as vertical_lead_coverage_pct
FROM officers;"

ssh root@143.198.146.44 "cd /home/seeds-metrics-backend/backend && source .env && psql \"host=\$DB_HOST port=\$DB_PORT dbname=\$DB_NAME user=\$DB_USER password=\$DB_PASSWORD sslmode=require\" -c \"$VERIFICATION_SQL\" 2>&1"

echo ""
echo "📊 Sample updated records:"
SAMPLE_SQL="SELECT 
    officer_id,
    officer_name,
    officer_email,
    supervisor_email,
    supervisor_name,
    vertical_lead_email,
    vertical_lead_name
FROM officers 
WHERE supervisor_email IS NOT NULL 
LIMIT 5;"

ssh root@143.198.146.44 "cd /home/seeds-metrics-backend/backend && source .env && psql \"host=\$DB_HOST port=\$DB_PORT dbname=\$DB_NAME user=\$DB_USER password=\$DB_PASSWORD sslmode=require\" -c \"$SAMPLE_SQL\" 2>&1"

echo ""
echo "=========================================="
echo "✅ Data Loading Complete!"
echo "=========================================="
echo ""
echo "SQL file saved at: $TEMP_SQL"
echo "You can review or re-run it if needed."

