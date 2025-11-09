#!/bin/bash

# Script to update officer regions from verticals.tsv
# Uses the officer mapping file to match TSV data to Seeds Metrics officers

set -e

echo "=========================================="
echo "Updating Officer Regions from verticals.tsv"
echo "=========================================="
echo ""

# Check if required files exist
if [ ! -f "verticals.tsv" ]; then
    echo "❌ Error: verticals.tsv not found"
    exit 1
fi

if [ ! -f "/tmp/officer_mapping.txt" ]; then
    echo "❌ Error: /tmp/officer_mapping.txt not found"
    exit 1
fi

# Create temporary SQL file
SQL_FILE="/tmp/update_regions.sql"
echo "-- Update officer regions from verticals.tsv" > "$SQL_FILE"
echo "BEGIN;" >> "$SQL_FILE"

# Counter for statistics
total_updates=0
skipped_no_region=0
skipped_no_mapping=0

echo "📊 Processing verticals.tsv..."

# Read verticals.tsv (skip header) and generate UPDATE statements
tail -n +2 verticals.tsv | while IFS=$'\t' read -r loan_officer_id loan_officer_email loan_officer_name loan_officer_phone branch_name branch_location branch_sub_location branch_state branch_supervisor_email branch_supervisor_name region_name vertical_lead_email vertical_lead_name; do
    
    # Skip if region is empty
    if [ -z "$region_name" ]; then
        ((skipped_no_region++)) || true
        continue
    fi
    
    # Find the Seeds officer_id from the mapping file
    seeds_officer_id=$(grep -F "|$loan_officer_email|" /tmp/officer_mapping.txt | cut -d'|' -f3 | head -1)
    
    # Skip if no mapping found
    if [ -z "$seeds_officer_id" ]; then
        ((skipped_no_mapping++)) || true
        continue
    fi
    
    # Escape single quotes in region name
    escaped_region=$(echo "$region_name" | sed "s/'/''/g")
    
    # Generate UPDATE statement
    echo "UPDATE officers SET region = '$escaped_region' WHERE officer_id = '$seeds_officer_id';" >> "$SQL_FILE"
    
    ((total_updates++)) || true
    
    # Progress indicator
    if [ $((total_updates % 50)) -eq 0 ]; then
        echo "  Processed $total_updates officers..."
    fi
done

echo "COMMIT;" >> "$SQL_FILE"

echo ""
echo "📈 Statistics:"
echo "  - Total UPDATE statements generated: $total_updates"
echo "  - Skipped (no region in TSV): $skipped_no_region"
echo "  - Skipped (no mapping found): $skipped_no_mapping"
echo ""

# Show sample of SQL file
echo "📄 Sample UPDATE statements:"
head -20 "$SQL_FILE"
echo "..."
echo ""

# Ask for confirmation
read -p "Do you want to execute these updates on the production database? (yes/no): " confirm

if [ "$confirm" != "yes" ]; then
    echo "❌ Update cancelled"
    exit 0
fi

echo ""
echo "🚀 Executing updates on production database..."

# Execute SQL on production database
ssh root@143.198.146.44 "cd /home/seeds-metrics-backend/backend && source .env && psql \"host=\$DB_HOST port=\$DB_PORT dbname=\$DB_NAME user=\$DB_USER password=\$DB_PASSWORD sslmode=require\" -f -" < "$SQL_FILE"

echo ""
echo "✅ Region updates completed!"
echo ""

# Verify the updates
echo "🔍 Verifying updates..."
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -c "SELECT region, COUNT(*) as count FROM officers GROUP BY region ORDER BY count DESC LIMIT 20;" 2>&1'

echo ""
echo "✅ Done!"

