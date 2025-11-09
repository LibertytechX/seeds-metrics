#!/bin/bash

# Script to analyze officer mapping between Django and Seeds Metrics using email addresses

echo "=============================================================================="
echo "OFFICER MAPPING ANALYSIS: Django → Seeds Metrics (Email-Based)"
echo "=============================================================================="
echo ""

echo "🔍 This script will:"
echo "   1. Get emails for the 550 matching customer_user_ids from Django"
echo "   2. Check how many of those emails exist in Seeds Metrics officers table"
echo "   3. Create a mapping file showing Django ID → Seeds officer_id"
echo ""

# Create temporary file for Django user data
echo "📊 Step 1: Fetching email addresses from Django for matching IDs..."
echo ""

# Get all matching IDs and their emails from Django
rm -f /tmp/django_users_emails.txt
TOTAL_IDS=$(wc -l < /tmp/matching_ids.txt | tr -d ' ')
COUNTER=0

echo "Processing $TOTAL_IDS customer_user_ids..."

# Process in batches for efficiency
cat /tmp/matching_ids.txt | while IFS= read -r id; do
    COUNTER=$((COUNTER + 1))
    if [ $((COUNTER % 50)) -eq 0 ]; then
        echo "   Processed $COUNTER / $TOTAL_IDS..."
    fi
done

# More efficient: Get all emails in one query
echo "   Fetching all emails in a single query..."
IDS_LIST=$(cat /tmp/matching_ids.txt | tr '\n' ',' | sed 's/,$//')

ssh root@143.198.146.44 "cd /home/seeds-metrics-backend/backend && source .env && psql \"host=164.90.155.2 port=5432 dbname=savings user=metricsuser password=\$DJANGO_DB_PASSWORD sslmode=require\" -t -A -c \"SELECT customer_user_id, email FROM accounts_customuser WHERE customer_user_id IN ($IDS_LIST) ORDER BY customer_user_id;\" 2>&1" | grep -v "^ssh" | grep -v "^$" > /tmp/django_users_emails.txt

DJANGO_EMAIL_COUNT=$(wc -l < /tmp/django_users_emails.txt | tr -d ' ')
echo "   ✅ Retrieved $DJANGO_EMAIL_COUNT email addresses from Django"
echo ""

# Get all officer emails from Seeds Metrics
echo "📊 Step 2: Fetching all officer emails from Seeds Metrics..."
ssh root@143.198.146.44 'cd /home/seeds-metrics-backend/backend && source .env && psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=require" -t -A -c "SELECT officer_id, officer_email FROM officers WHERE officer_email IS NOT NULL AND officer_email != '\'''\'' ORDER BY officer_id::integer;" 2>&1' | grep -v "^ssh" | grep -v "^$" > /tmp/seeds_officers_emails.txt

SEEDS_EMAIL_COUNT=$(wc -l < /tmp/seeds_officers_emails.txt | tr -d ' ')
echo "   ✅ Retrieved $SEEDS_EMAIL_COUNT officer emails from Seeds Metrics"
echo ""

# Create mapping file
echo "📊 Step 3: Creating mapping between Django customer_user_id and Seeds officer_id..."
echo ""
echo "customer_user_id|django_email|seeds_officer_id|seeds_email|match_status" > /tmp/officer_mapping.txt

MATCHED=0
NOT_FOUND=0
EMAIL_MISMATCH=0

# Process each Django user
while IFS='|' read -r customer_id email; do
    # Clean up whitespace
    customer_id=$(echo "$customer_id" | tr -d ' ')
    email=$(echo "$email" | tr -d ' ')
    
    # Look for this email in Seeds Metrics
    SEEDS_MATCH=$(grep -i "|$email$" /tmp/seeds_officers_emails.txt | head -1)
    
    if [ ! -z "$SEEDS_MATCH" ]; then
        SEEDS_OFFICER_ID=$(echo "$SEEDS_MATCH" | cut -d'|' -f1 | tr -d ' ')
        SEEDS_EMAIL=$(echo "$SEEDS_MATCH" | cut -d'|' -f2 | tr -d ' ')
        echo "$customer_id|$email|$SEEDS_OFFICER_ID|$SEEDS_EMAIL|MATCHED" >> /tmp/officer_mapping.txt
        MATCHED=$((MATCHED + 1))
    else
        echo "$customer_id|$email||NOT_FOUND|NOT_FOUND" >> /tmp/officer_mapping.txt
        NOT_FOUND=$((NOT_FOUND + 1))
    fi
done < /tmp/django_users_emails.txt

echo "   ✅ Mapping complete!"
echo ""

# Summary
echo "=============================================================================="
echo "📊 MAPPING RESULTS"
echo "=============================================================================="
echo "Total Django users processed:           $DJANGO_EMAIL_COUNT"
echo "Matched by email in Seeds Metrics:      $MATCHED"
echo "Not found in Seeds Metrics:             $NOT_FOUND"
echo ""

if [ $DJANGO_EMAIL_COUNT -gt 0 ]; then
    MATCH_PERCENT=$(awk "BEGIN {printf \"%.2f\", ($MATCHED / $DJANGO_EMAIL_COUNT) * 100}")
    echo "Match rate (email-based):               $MATCH_PERCENT%"
fi
echo ""

# Show sample mappings
echo "=============================================================================="
echo "📋 SAMPLE MAPPINGS (First 20)"
echo "=============================================================================="
echo ""
head -21 /tmp/officer_mapping.txt | column -t -s'|'
echo ""

# Show sample of non-matches
if [ $NOT_FOUND -gt 0 ]; then
    echo "=============================================================================="
    echo "❌ SAMPLE NON-MATCHES (First 10)"
    echo "=============================================================================="
    echo ""
    grep "NOT_FOUND" /tmp/officer_mapping.txt | head -10 | column -t -s'|'
    echo ""
fi

# Save results
echo "=============================================================================="
echo "💾 FILES GENERATED"
echo "=============================================================================="
echo ""
echo "   /tmp/django_users_emails.txt    - Django customer_user_id|email ($DJANGO_EMAIL_COUNT records)"
echo "   /tmp/seeds_officers_emails.txt  - Seeds officer_id|email ($SEEDS_EMAIL_COUNT records)"
echo "   /tmp/officer_mapping.txt        - Complete mapping file ($((DJANGO_EMAIL_COUNT + 1)) records including header)"
echo ""
echo "   The mapping file shows:"
echo "   - customer_user_id: Django ID"
echo "   - django_email: Email from Django"
echo "   - seeds_officer_id: Corresponding officer_id in Seeds Metrics"
echo "   - seeds_email: Email from Seeds Metrics (for verification)"
echo "   - match_status: MATCHED or NOT_FOUND"
echo ""

# Analysis
echo "=============================================================================="
echo "🔍 ANALYSIS & RECOMMENDATIONS"
echo "=============================================================================="
echo ""

if [ $MATCHED -eq 0 ]; then
    echo "❌ CRITICAL: No email matches found!"
    echo ""
    echo "Possible issues:"
    echo "1. Email addresses in Django don't match Seeds Metrics"
    echo "2. Different email formats or case sensitivity"
    echo "3. Officers not yet synced to Seeds Metrics"
    echo ""
elif [ $MATCHED -lt $((DJANGO_EMAIL_COUNT / 2)) ]; then
    echo "⚠️  WARNING: Less than 50% match rate!"
    echo ""
    echo "This indicates:"
    echo "- Many Django users are not in Seeds Metrics officers table"
    echo "- Email addresses may have changed between systems"
    echo "- Officers may need to be synced from Django to Seeds Metrics"
    echo ""
    echo "Recommended actions:"
    echo "1. Review the NOT_FOUND list in /tmp/officer_mapping.txt"
    echo "2. Sync missing officers from Django to Seeds Metrics"
    echo "3. Update email addresses if they have changed"
    echo ""
else
    echo "✅ GOOD: $MATCH_PERCENT% match rate!"
    echo ""
    echo "The email-based mapping is working well."
    echo ""
    if [ $NOT_FOUND -gt 0 ]; then
        echo "Note: $NOT_FOUND officers from Django are not in Seeds Metrics."
        echo "Review /tmp/officer_mapping.txt to see which officers are missing."
    fi
    echo ""
fi

echo "=============================================================================="
echo "✅ Analysis complete!"
echo "=============================================================================="
echo ""
echo "Next steps:"
echo "1. Review the mapping file: /tmp/officer_mapping.txt"
echo "2. Use this mapping to update verticals.tsv with Seeds Metrics officer_ids"
echo "3. Update loans table with vertical/branch information using the mapping"
echo ""

