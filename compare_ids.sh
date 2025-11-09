#!/bin/bash

# Script to compare IDs from verticals.tsv with accounts_customuser table in Django database

echo "=========================================="
echo "ID COMPARISON: verticals.tsv vs Django DB"
echo "=========================================="
echo ""

# Extract IDs from TSV file (first column, skip header)
echo "📄 Extracting IDs from verticals.tsv..."
tail -n +2 verticals.tsv | cut -f1 | sort | uniq > /tmp/tsv_ids.txt
TSV_COUNT=$(wc -l < /tmp/tsv_ids.txt | tr -d ' ')
echo "   Found $TSV_COUNT unique IDs in TSV file"
echo ""

# Query Django database for all customer_user_id values
echo "🗄️  Querying Django database for customer_user_id values..."
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=164.90.155.2 port=5432 dbname=savings user=metricsuser password=$DJANGO_DB_PASSWORD sslmode=require" -t -A -c "SELECT customer_user_id FROM accounts_customuser WHERE customer_user_id IS NOT NULL ORDER BY customer_user_id;"' 2>/dev/null | grep -v '^$' | tr -d ' ' | sort | uniq > /tmp/db_ids.txt

DB_COUNT=$(wc -l < /tmp/db_ids.txt | tr -d ' ')
echo "   Found $DB_COUNT unique IDs in database"
echo ""

# Find IDs in TSV but not in DB
echo "🔍 Finding IDs in TSV but NOT in database..."
comm -23 /tmp/tsv_ids.txt /tmp/db_ids.txt > /tmp/tsv_only.txt
TSV_ONLY_COUNT=$(wc -l < /tmp/tsv_only.txt | tr -d ' ')
echo "   Found $TSV_ONLY_COUNT IDs"
if [ $TSV_ONLY_COUNT -gt 0 ]; then
    echo "   First 20 IDs in TSV but not in DB:"
    head -20 /tmp/tsv_only.txt | sed 's/^/      /'
    if [ $TSV_ONLY_COUNT -gt 20 ]; then
        echo "      ... and $((TSV_ONLY_COUNT - 20)) more"
    fi
fi
echo ""

# Find IDs in DB but not in TSV
echo "🔍 Finding IDs in database but NOT in TSV..."
comm -13 /tmp/tsv_ids.txt /tmp/db_ids.txt > /tmp/db_only.txt
DB_ONLY_COUNT=$(wc -l < /tmp/db_only.txt | tr -d ' ')
echo "   Found $DB_ONLY_COUNT IDs"
if [ $DB_ONLY_COUNT -gt 0 ]; then
    echo "   First 20 IDs in DB but not in TSV:"
    head -20 /tmp/db_only.txt | sed 's/^/      /'
    if [ $DB_ONLY_COUNT -gt 20 ]; then
        echo "      ... and $((DB_ONLY_COUNT - 20)) more"
    fi
fi
echo ""

# Find matching IDs
echo "✅ Finding matching IDs (in both TSV and database)..."
comm -12 /tmp/tsv_ids.txt /tmp/db_ids.txt > /tmp/matching_ids.txt
MATCHING_COUNT=$(wc -l < /tmp/matching_ids.txt | tr -d ' ')
echo "   Found $MATCHING_COUNT matching IDs"
echo ""

# Summary
echo "=========================================="
echo "📊 SUMMARY"
echo "=========================================="
echo "Total IDs in TSV file:              $TSV_COUNT"
echo "Total IDs in database:              $DB_COUNT"
echo "Matching IDs (in both):             $MATCHING_COUNT"
echo "IDs only in TSV (missing from DB):  $TSV_ONLY_COUNT"
echo "IDs only in DB (missing from TSV):  $DB_ONLY_COUNT"
echo ""

# Calculate percentages
if [ $TSV_COUNT -gt 0 ]; then
    MATCH_PERCENT=$(awk "BEGIN {printf \"%.2f\", ($MATCHING_COUNT / $TSV_COUNT) * 100}")
    echo "Match rate (TSV perspective):       $MATCH_PERCENT%"
fi
if [ $DB_COUNT -gt 0 ]; then
    MATCH_PERCENT_DB=$(awk "BEGIN {printf \"%.2f\", ($MATCHING_COUNT / $DB_COUNT) * 100}")
    echo "Match rate (DB perspective):        $MATCH_PERCENT_DB%"
fi
echo ""

# Save detailed results
echo "💾 Saving detailed results..."
echo "   /tmp/tsv_ids.txt        - All IDs from TSV file"
echo "   /tmp/db_ids.txt         - All IDs from database"
echo "   /tmp/matching_ids.txt   - IDs in both TSV and DB"
echo "   /tmp/tsv_only.txt       - IDs only in TSV (missing from DB)"
echo "   /tmp/db_only.txt        - IDs only in DB (missing from TSV)"
echo ""

# Check for potential data issues
echo "=========================================="
echo "⚠️  POTENTIAL ISSUES"
echo "=========================================="
if [ $TSV_ONLY_COUNT -gt 0 ]; then
    echo "❌ WARNING: $TSV_ONLY_COUNT IDs in TSV file do not exist in the database!"
    echo "   These users may have been deleted or the IDs are incorrect."
    echo "   See /tmp/tsv_only.txt for the full list."
    echo ""
fi

if [ $DB_ONLY_COUNT -gt 0 ]; then
    echo "ℹ️  INFO: $DB_ONLY_COUNT IDs in database are not in the TSV file."
    echo "   This is expected if the TSV file only contains a subset of users."
    echo "   See /tmp/db_only.txt for the full list."
    echo ""
fi

if [ $TSV_ONLY_COUNT -eq 0 ] && [ $MATCHING_COUNT -eq $TSV_COUNT ]; then
    echo "✅ EXCELLENT: All IDs in TSV file exist in the database!"
    echo ""
fi

echo "=========================================="
echo "Comparison complete!"
echo "=========================================="

