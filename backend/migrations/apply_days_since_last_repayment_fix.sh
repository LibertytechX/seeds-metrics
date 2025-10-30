#!/bin/bash

# Script to apply days_since_last_repayment fix and recalculate existing loans

echo "🔧 Applying days_since_last_repayment trigger update..."

# Apply the migration
PGPASSWORD=@seedsuser2020 psql -U seedsuser \
  -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
  -p 25060 \
  -d seedsmetrics \
  -f backend/migrations/003_update_days_since_last_repayment.sql

if [ $? -eq 0 ]; then
    echo "✅ Trigger updated successfully"
    
    echo ""
    echo "🔄 Recalculating days_since_last_repayment for existing loans..."
    
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
        echo "📊 Checking loan 13 days_since_last_repayment..."
        PGPASSWORD=@seedsuser2020 psql -U seedsuser \
          -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com \
          -p 25060 \
          -d seedsmetrics \
          -c "SELECT loan_id, 
                     (SELECT MAX(payment_date) FROM repayments WHERE loan_id = '13' AND is_reversed = FALSE) as last_payment_date,
                     CURRENT_DATE as today,
                     days_since_last_repayment,
                     CURRENT_DATE - (SELECT MAX(payment_date) FROM repayments WHERE loan_id = '13' AND is_reversed = FALSE) as expected_days
              FROM loans 
              WHERE loan_id = '13';"
    else
        echo "❌ Failed to recalculate loan metrics"
        exit 1
    fi
else
    echo "❌ Failed to update trigger"
    exit 1
fi

