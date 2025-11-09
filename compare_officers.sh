#!/bin/bash

# Script to compare matching customer_user_ids with Seeds Metrics officers table

echo "=========================================================================="
echo "OFFICER MAPPING: Django customer_user_ids → Seeds Metrics officers table"
echo "=========================================================================="
echo ""

# Get all officer_ids from Seeds Metrics database
echo "📊 Querying Seeds Metrics database for all officer IDs..."
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -t -A -c "SELECT officer_id FROM officers ORDER BY officer_id::integer;"' 2>/dev/null | grep -v '^$' | sort > /tmp/seeds_officer_ids.txt

SEEDS_COUNT=$(wc -l < /tmp/seeds_officer_ids.txt | tr -d ' ')
echo "   Found $SEEDS_COUNT officer IDs in Seeds Metrics database"
echo ""

# Load matching IDs from previous comparison
MATCHING_COUNT=$(wc -l < /tmp/matching_ids.txt | tr -d ' ')
echo "📄 Loaded $MATCHING_COUNT matching customer_user_ids from Django comparison"
echo ""

# Find which matching IDs exist in Seeds Metrics officers table
echo "🔍 Finding which matching IDs exist in Seeds Metrics officers table..."
comm -12 /tmp/matching_ids.txt /tmp/seeds_officer_ids.txt > /tmp/in_both_systems.txt
IN_BOTH_COUNT=$(wc -l < /tmp/in_both_systems.txt | tr -d ' ')
echo "   Found $IN_BOTH_COUNT IDs that exist in BOTH Django and Seeds Metrics"
echo ""

# Find matching IDs that are NOT in Seeds Metrics
echo "🔍 Finding matching IDs that are NOT in Seeds Metrics officers table..."
comm -23 /tmp/matching_ids.txt /tmp/seeds_officer_ids.txt > /tmp/missing_from_seeds.txt
MISSING_COUNT=$(wc -l < /tmp/missing_from_seeds.txt | tr -d ' ')
echo "   Found $MISSING_COUNT IDs that are in Django but NOT in Seeds Metrics"
if [ $MISSING_COUNT -gt 0 ]; then
    echo "   First 20 missing IDs:"
    head -20 /tmp/missing_from_seeds.txt | sed 's/^/      /'
    if [ $MISSING_COUNT -gt 20 ]; then
        echo "      ... and $((MISSING_COUNT - 20)) more"
    fi
fi
echo ""

# Find Seeds Metrics officers that are NOT in the matching list
echo "🔍 Finding Seeds Metrics officers NOT in the matching list..."
comm -13 /tmp/matching_ids.txt /tmp/seeds_officer_ids.txt > /tmp/seeds_only.txt
SEEDS_ONLY_COUNT=$(wc -l < /tmp/seeds_only.txt | tr -d ' ')
echo "   Found $SEEDS_ONLY_COUNT officer IDs in Seeds Metrics but not in matching list"
echo ""

# Summary
echo "=========================================================================="
echo "📊 SUMMARY"
echo "=========================================================================="
echo "Total matching customer_user_ids (Django):     $MATCHING_COUNT"
echo "Total officer_ids in Seeds Metrics:            $SEEDS_COUNT"
echo "IDs in BOTH systems:                           $IN_BOTH_COUNT"
echo "Matching IDs missing from Seeds Metrics:       $MISSING_COUNT"
echo "Seeds Metrics officers not in matching list:   $SEEDS_ONLY_COUNT"
echo ""

# Calculate percentages
if [ $MATCHING_COUNT -gt 0 ]; then
    COVERAGE_PERCENT=$(awk "BEGIN {printf \"%.2f\", ($IN_BOTH_COUNT / $MATCHING_COUNT) * 100}")
    echo "Coverage rate (matching IDs in Seeds):         $COVERAGE_PERCENT%"
fi
if [ $SEEDS_COUNT -gt 0 ]; then
    MATCH_PERCENT=$(awk "BEGIN {printf \"%.2f\", ($IN_BOTH_COUNT / $SEEDS_COUNT) * 100}")
    echo "Match rate (Seeds officers in matching list):  $MATCH_PERCENT%"
fi
echo ""

# Save detailed results
echo "💾 Saving detailed results..."
echo "   /tmp/seeds_officer_ids.txt    - All officer IDs from Seeds Metrics ($SEEDS_COUNT IDs)"
echo "   /tmp/matching_ids.txt          - Matching customer_user_ids from Django ($MATCHING_COUNT IDs)"
echo "   /tmp/in_both_systems.txt       - IDs in both Django and Seeds Metrics ($IN_BOTH_COUNT IDs)"
echo "   /tmp/missing_from_seeds.txt    - Matching IDs missing from Seeds Metrics ($MISSING_COUNT IDs)"
echo "   /tmp/seeds_only.txt            - Seeds officers not in matching list ($SEEDS_ONLY_COUNT IDs)"
echo ""

# Get sample records from Seeds Metrics for IDs that exist in both systems
if [ $IN_BOTH_COUNT -gt 0 ]; then
    echo "=========================================================================="
    echo "📋 SAMPLE RECORDS: Officers in Both Systems"
    echo "=========================================================================="
    
    # Get first 10 IDs from in_both_systems.txt
    SAMPLE_IDS=$(head -10 /tmp/in_both_systems.txt | tr '\n' ',' | sed 's/,$//')
    
    if [ ! -z "$SAMPLE_IDS" ]; then
        # Build SQL IN clause
        SQL_IN=$(echo "$SAMPLE_IDS" | sed "s/,/','/g" | sed "s/^/'/" | sed "s/$/'/")
        
        echo ""
        echo "Sample of officers that exist in both Django and Seeds Metrics:"
        echo ""
        ssh root@143.198.146.44 "cd /home/seeds-metrics-backend/backend && source .env && psql \"host=\$DB_HOST port=\$DB_PORT dbname=\$DB_NAME user=\$DB_USER password=\$DB_PASSWORD sslmode=require\" -c \"SELECT officer_id, officer_name, officer_email, region, branch, user_type FROM officers WHERE officer_id IN ($SQL_IN) ORDER BY officer_id::integer LIMIT 10;\" 2>&1" | grep -v "^ssh"
        echo ""
    fi
fi

# Analysis and recommendations
echo "=========================================================================="
echo "⚠️  ANALYSIS"
echo "=========================================================================="

if [ $IN_BOTH_COUNT -eq 0 ]; then
    echo "❌ CRITICAL: NO matching IDs found in Seeds Metrics officers table!"
    echo ""
    echo "This suggests that:"
    echo "1. The officer_id field in Seeds Metrics uses a DIFFERENT ID system"
    echo "2. The mapping between Django customer_user_id and Seeds officer_id is not 1:1"
    echo "3. There may be a separate mapping table or field needed"
    echo ""
    echo "Recommended next steps:"
    echo "1. Check if officers table has an email field that matches Django users"
    echo "2. Look for a customer_user_id or django_id field in the officers table"
    echo "3. Check if there's a separate mapping table in Seeds Metrics database"
    echo ""
elif [ $IN_BOTH_COUNT -lt $((MATCHING_COUNT / 2)) ]; then
    echo "⚠️  WARNING: Less than 50% of matching IDs exist in Seeds Metrics!"
    echo ""
    echo "Coverage: $COVERAGE_PERCENT%"
    echo ""
    echo "This indicates:"
    echo "- Many loan officers from Django are not yet synced to Seeds Metrics"
    echo "- OR the ID mapping is not consistent between systems"
    echo ""
    echo "Recommended actions:"
    echo "1. Sync missing officers from Django to Seeds Metrics"
    echo "2. Verify the ID mapping logic between systems"
    echo ""
else
    echo "✅ GOOD: $COVERAGE_PERCENT% of matching IDs exist in Seeds Metrics"
    echo ""
    echo "This indicates reasonable data synchronization between systems."
    echo ""
    if [ $MISSING_COUNT -gt 0 ]; then
        echo "Note: $MISSING_COUNT officers from Django are not in Seeds Metrics yet."
        echo "Consider syncing these officers if they are active loan officers."
    fi
    echo ""
fi

echo "=========================================================================="
echo "Comparison complete!"
echo "=========================================================================="

