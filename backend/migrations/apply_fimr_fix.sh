#!/bin/bash

# Script to apply FIMR trigger fix and recalculate existing loans

echo "🔧 Applying FIMR trigger update..."

# Apply the migration
PGPASSWORD=@seedsuser2020 psql -U seedsuser \
  -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 \
  -d seedsmetrics \
  -f backend/migrations/002_update_fimr_trigger.sql

if [ $? -eq 0 ]; then
    echo "✅ Trigger updated successfully"
    
    echo ""
    echo "🔄 Recalculating FIMR tags for existing loans..."
    
    # Manually trigger recalculation by updating a dummy field on each repayment
    # This will fire the trigger and recalculate all loan metrics
    PGPASSWORD=@seedsuser2020 psql -U seedsuser \
      -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
      -p 25060 \
      -d seedsmetrics \
      -c "UPDATE repayments SET updated_at = CURRENT_TIMESTAMP WHERE is_reversed = FALSE;"
    
    if [ $? -eq 0 ]; then
        echo "✅ Loan metrics recalculated"
        
        echo ""
        echo "📊 Checking loan 13 FIMR status..."
        PGPASSWORD=@seedsuser2020 psql -U seedsuser \
          -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
          -p 25060 \
          -d seedsmetrics \
          -c "SELECT loan_id, disbursement_date, first_payment_due_date, first_payment_received_date, fimr_tagged FROM loans WHERE loan_id = '13';"
    else
        echo "❌ Failed to recalculate loan metrics"
        exit 1
    fi
else
    echo "❌ Failed to update trigger"
    exit 1
fi

