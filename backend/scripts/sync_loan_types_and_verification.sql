-- Sync loan_type and verification_status from Django to SeedsMetrics
-- This script updates the loan_type and verification_status fields in the SeedsMetrics database
-- by fetching the data from the Django database

-- Create foreign data wrapper extension if not exists
CREATE EXTENSION IF NOT EXISTS postgres_fdw;

-- Drop existing server and user mapping if they exist
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_foreign_server WHERE srvname = 'django_server') THEN
        DROP SERVER django_server CASCADE;
    END IF;
END $$;

-- Create foreign server connection to Django database
CREATE SERVER django_server
    FOREIGN DATA WRAPPER postgres_fdw
    OPTIONS (host '164.90.155.2', port '5432', dbname 'savings', sslmode 'require');

-- Create user mapping for the foreign server
CREATE USER MAPPING FOR seedsuser
    SERVER django_server
    OPTIONS (user 'metricsuser', password 'EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg');

-- Import the foreign table schema
IMPORT FOREIGN SCHEMA public
    LIMIT TO (loans_ajoloan)
    FROM SERVER django_server
    INTO public;

-- Update loan_type and verification_status in SeedsMetrics from Django
UPDATE loans l
SET 
    loan_type = d.loan_type,
    verification_status = d.verification_stage,
    updated_at = NOW()
FROM loans_ajoloan d
WHERE l.loan_id = d.id::VARCHAR(50)
    AND d.is_disbursed = TRUE
    AND (
        l.loan_type IS DISTINCT FROM d.loan_type
        OR l.verification_status IS DISTINCT FROM d.verification_stage
    );

-- Get count of updated rows
DO $$
DECLARE
    updated_count INTEGER;
BEGIN
    GET DIAGNOSTICS updated_count = ROW_COUNT;
    RAISE NOTICE 'Updated % loans with loan_type and verification_status', updated_count;
END $$;

-- Clean up: Drop the foreign table and server
DROP FOREIGN TABLE IF EXISTS loans_ajoloan CASCADE;
DROP SERVER IF EXISTS django_server CASCADE;

-- Show sample of updated data
SELECT 
    loan_id,
    loan_type,
    verification_status,
    updated_at
FROM loans
WHERE loan_type IS NOT NULL OR verification_status IS NOT NULL
ORDER BY updated_at DESC
LIMIT 10;

-- Show distinct loan types
SELECT DISTINCT loan_type, COUNT(*) as count
FROM loans
WHERE loan_type IS NOT NULL AND loan_type != ''
GROUP BY loan_type
ORDER BY loan_type;

-- Show distinct verification statuses
SELECT DISTINCT verification_status, COUNT(*) as count
FROM loans
WHERE verification_status IS NOT NULL AND verification_status != ''
GROUP BY verification_status
ORDER BY verification_status;

