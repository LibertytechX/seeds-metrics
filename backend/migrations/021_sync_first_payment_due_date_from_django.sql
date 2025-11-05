-- Migration 021: Sync first_payment_due_date from Django database
-- This migration updates the first_payment_due_date field for all loans
-- by pulling the start_date from the Django database using dblink
-- 
-- NOTE: This migration requires:
-- 1. dblink extension to be installed on the SeedsMetrics database
-- 2. Network access to the Django database from the SeedsMetrics database
-- 3. Valid credentials for the Django database
--
-- If dblink is not available, run the sync_from_django.go script instead:
--   cd backend && go run ./scripts/sync_from_django.go
--
-- This script will:
-- 1. Connect to both Django and SeedsMetrics databases
-- 2. Fetch all loans with their start_date from Django
-- 3. Update the first_payment_due_date in SeedsMetrics
-- 4. Report the number of loans updated

-- Check if dblink extension is available
-- If not, you'll need to install it or run the Go sync script

-- Create a temporary table with Django data using dblink
-- This assumes dblink is installed and configured
-- If you get an error, run: CREATE EXTENSION IF NOT EXISTS dblink;

-- For now, this migration is a placeholder
-- The actual sync should be done using the Go script: backend/scripts/sync_from_django.go
-- 
-- To run the sync from a machine that can access both databases:
-- cd backend
-- DJANGO_DB_HOST=164.90.155.2 \
-- DJANGO_DB_USER=seedsuser \
-- DJANGO_DB_PASSWORD=<password> \
-- DJANGO_DB_NAME=seeds_metrics \
-- DJANGO_DB_SSLMODE=disable \
-- DB_HOST=<seedsmetrics_host> \
-- DB_USER=<seedsmetrics_user> \
-- DB_PASSWORD=<seedsmetrics_password> \
-- DB_NAME=seedsmetrics \
-- go run ./scripts/sync_from_django.go

-- After running the sync script, verify the update:
-- SELECT 
--     COUNT(*) as total_loans,
--     COUNT(CASE WHEN (first_payment_due_date - disbursement_date) = 30 THEN 1 END) as loans_with_30_day_gap,
--     COUNT(CASE WHEN (first_payment_due_date - disbursement_date) != 30 THEN 1 END) as loans_with_correct_gap
-- FROM loans
-- WHERE first_payment_due_date IS NOT NULL;

-- This migration is complete when all loans have the correct first_payment_due_date
-- from the Django database instead of the default 30-day calculation.

