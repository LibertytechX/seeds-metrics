-- Update first_payment_due_date from Django database
-- This script syncs the start_date from Django's loans_ajoloan table to SeedsMetrics loans table

-- Create a temporary table with Django data
CREATE TEMP TABLE temp_django_loans AS
SELECT
    l.id::VARCHAR(50) as loan_id,
    l.start_date as first_payment_due_date
FROM dblink(
    'host=164.90.155.2 port=5432 dbname=savings user=metricsuser password=EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg sslmode=require',
    'SELECT id, start_date FROM loans_ajoloan WHERE is_disbursed = TRUE'
) AS l(id INTEGER, start_date DATE);

-- Update loans table with first_payment_due_date from Django
UPDATE loans
SET
    first_payment_due_date = t.first_payment_due_date,
    updated_at = CURRENT_TIMESTAMP
FROM temp_django_loans t
WHERE loans.loan_id = t.loan_id
  AND (loans.first_payment_due_date IS NULL OR loans.first_payment_due_date != t.first_payment_due_date);

-- Show summary
SELECT
    COUNT(*) as total_loans_updated,
    COUNT(CASE WHEN first_payment_due_date IS NOT NULL THEN 1 END) as loans_with_first_payment_due_date,
    COUNT(CASE WHEN first_payment_due_date IS NULL THEN 1 END) as loans_without_first_payment_due_date
FROM loans;

