-- Migration 013: Setup Foreign Data Wrapper to Django Database
-- Purpose: Enable direct read access to Django database for real-time data
-- Date: 2025-11-04
-- Related: DJANGO_INTEGRATION_ARCHITECTURE.md

-- ============================================================================
-- PHASE 1: INSTALL AND CONFIGURE FOREIGN DATA WRAPPER
-- ============================================================================

-- Step 1: Install postgres_fdw extension
CREATE EXTENSION IF NOT EXISTS postgres_fdw;

-- Step 2: Create foreign server connection to Django database
-- This establishes the connection to the main Django database
DROP SERVER IF EXISTS django_db CASCADE;

CREATE SERVER django_db
FOREIGN DATA WRAPPER postgres_fdw
OPTIONS (
    host '164.90.155.2',
    port '5432',
    dbname 'savings',
    fetch_size '10000',           -- Optimize bulk fetches
    use_remote_estimate 'true'    -- Use remote query planner estimates
);

-- Step 3: Create user mapping with read-only credentials
-- This maps the local seedsuser to the Django database read-only user
CREATE USER MAPPING IF NOT EXISTS FOR seedsuser
SERVER django_db
OPTIONS (
    user 'metricsuser',
    password 'EiRXo6IfeHQuM3wcbZ67$LzwmVKCXhpUhWg'
);

-- Step 4: Create schema to hold foreign tables
CREATE SCHEMA IF NOT EXISTS django_db;

-- Step 5: Import foreign tables from Django database
-- Import only the core business tables we need
IMPORT FOREIGN SCHEMA public
LIMIT TO (
    accounts_customuser,
    ajo_ajouser,
    loans_ajoloan,
    loans_ajoloanrepayment,
    loans_ajoloanschedule
)
FROM SERVER django_db
INTO django_db;

-- ============================================================================
-- PHASE 2: CREATE SCHEMA MAPPING VIEWS
-- ============================================================================

-- View 1: officers_live
-- Maps Django's accounts_customuser to SeedsMetrics officers schema
-- This provides real-time officer/agent data
DROP VIEW IF EXISTS officers_live CASCADE;

CREATE VIEW officers_live AS
SELECT
    u.id::VARCHAR(50) as officer_id,
    COALESCE(u.username, u.email) as officer_name,
    u.user_phone as officer_phone,
    u.email as officer_email,
    u.user_branch as branch,
    -- Derive region from branch (default to 'Nigeria' for now)
    CASE
        WHEN u.user_branch LIKE '%Lagos%' THEN 'Lagos'
        WHEN u.user_branch LIKE '%Abuja%' THEN 'FCT'
        WHEN u.user_branch LIKE '%Kano%' THEN 'Kano'
        WHEN u.user_branch LIKE '%Ibadan%' THEN 'Oyo'
        WHEN u.user_branch LIKE '%Port Harcourt%' THEN 'Rivers'
        ELSE 'Nigeria'
    END as region,
    -- Map performance_status to employment_status
    CASE
        WHEN u.performance_status = 'Active' THEN 'Active'
        WHEN u.performance_status = 'Inactive' THEN 'Inactive'
        WHEN u.performance_status = 'Suspended' THEN 'Inactive'
        ELSE 'Active'
    END as employment_status,
    u.date_joined::DATE as hire_date,
    NULL::DATE as termination_date,
    -- Derive primary_channel from user_type
    CASE
        WHEN u.user_type IN ('PROSPER_AGENT', 'DMO_AGENT') THEN 'Partner'
        ELSE 'Direct'
    END as primary_channel,
    u.date_joined as created_at,
    CURRENT_TIMESTAMP as updated_at
FROM django_db.accounts_customuser u
WHERE u.user_type IN (
    'AGENT',
    'STAFF_AGENT',
    'PROSPER_AGENT',
    'DMO_AGENT',
    'AJO_AGENT',
    'RECOVERY_AGENT'
)
AND u.is_active = TRUE;

-- Add comment for documentation
COMMENT ON VIEW officers_live IS 'Real-time view of officers/agents from Django database via Foreign Data Wrapper';

-- View 2: customers_live
-- Maps Django's ajo_ajouser to SeedsMetrics customers schema
-- This provides real-time customer data
DROP VIEW IF EXISTS customers_live CASCADE;

CREATE VIEW customers_live AS
SELECT
    u.id::VARCHAR(50) as customer_id,
    COALESCE(
        TRIM(u.first_name || ' ' || u.last_name),
        u.phone_number
    ) as customer_name,
    u.phone_number as customer_phone,
    NULL as customer_email,  -- Not tracked in Django schema
    u.dob as date_of_birth,
    u.gender,
    u.state,
    u.lga,
    u.address,
    -- Derive KYC status from verification flags
    CASE
        WHEN u.bvn_verified = TRUE AND u.onboarding_verified = TRUE THEN 'Verified'
        WHEN u.bvn_verified = TRUE AND u.onboarding_verified = FALSE THEN 'Partial'
        ELSE 'Pending'
    END as kyc_status,
    NULL::DATE as kyc_verified_date,  -- Not tracked in Django
    u.date_created as created_at,
    u.date_modified as updated_at
FROM django_db.ajo_ajouser u
WHERE u.onboarding_complete = TRUE;

-- Add comment for documentation
COMMENT ON VIEW customers_live IS 'Real-time view of customers from Django database via Foreign Data Wrapper';

-- View 3: loans_base_live
-- Maps Django's loans_ajoloan base fields to SeedsMetrics loans schema
-- This provides real-time loan base data (NOT computed fields)
DROP VIEW IF EXISTS loans_base_live CASCADE;

CREATE VIEW loans_base_live AS
SELECT
    l.id::VARCHAR(50) as loan_id,
    l.borrower_id::VARCHAR(50) as customer_id,
    l.borrower_full_name as customer_name,
    l.borrower_phone_number as customer_phone,
    l.agent_id::VARCHAR(50) as officer_id,
    -- Join to get officer details
    o.officer_name,
    o.officer_phone,
    o.region,
    o.branch,
    -- Join to get customer state
    c.state,
    -- Loan financial details
    l.amount_disbursed as loan_amount,
    l.repayment_amount,
    l.date_disbursed::DATE as disbursement_date,
    l.expected_end_date as maturity_date,
    l.tenor_in_days as loan_term_days,
    -- Convert interest rate from percentage to decimal
    (l.interest_rate / 100.0)::DECIMAL(5,4) as interest_rate,
    -- Sum all fees
    (COALESCE(l.processing_fee, 0) + COALESCE(l.nem_fee, 0))::DECIMAL(15,2) as fee_amount,
    -- Derive channel from loan_type
    CASE
        WHEN l.loan_type = 'BNPL' THEN 'Partner'
        WHEN l.loan_type = 'PROSPER' THEN 'Partner'
        WHEN l.loan_type = 'DMO' THEN 'Partner'
        ELSE 'Direct'
    END as channel,
    NULL as channel_partner,  -- Not in Django schema
    -- Loan status
    l.status,
    l.date_completed::DATE as closed_date,
    -- Default wave (not in Django schema)
    'Wave 2' as wave,
    -- Timestamps
    l.date_created as created_at,
    l.date_modified as updated_at
FROM django_db.loans_ajoloan l
LEFT JOIN officers_live o ON l.agent_id::VARCHAR(50) = o.officer_id
LEFT JOIN customers_live c ON l.borrower_id::VARCHAR(50) = c.customer_id
WHERE l.is_disbursed = TRUE;

-- Add comment for documentation
COMMENT ON VIEW loans_base_live IS 'Real-time view of loan base fields from Django database via Foreign Data Wrapper. Does NOT include computed fields (DPD, outstanding balances, etc.) - those remain in local loans table.';

-- ============================================================================
-- PHASE 3: CREATE HELPER VIEWS FOR DATA VALIDATION
-- ============================================================================

-- View to compare officer counts between Django and local
CREATE OR REPLACE VIEW data_validation_officers AS
SELECT
    'Django (Live)' as source,
    COUNT(*) as officer_count
FROM officers_live
UNION ALL
SELECT
    'SeedsMetrics (Local)' as source,
    COUNT(*) as officer_count
FROM officers;

-- View to compare customer counts
CREATE OR REPLACE VIEW data_validation_customers AS
SELECT
    'Django (Live)' as source,
    COUNT(*) as customer_count
FROM customers_live
UNION ALL
SELECT
    'SeedsMetrics (Local)' as source,
    COUNT(*) as customer_count
FROM customers;

-- View to compare loan counts
CREATE OR REPLACE VIEW data_validation_loans AS
SELECT
    'Django (Live)' as source,
    COUNT(*) as loan_count
FROM loans_base_live
UNION ALL
SELECT
    'SeedsMetrics (Local)' as source,
    COUNT(*) as loan_count
FROM loans;

-- ============================================================================
-- PHASE 4: CREATE PERFORMANCE TESTING QUERIES
-- ============================================================================

-- Function to test query performance
CREATE OR REPLACE FUNCTION test_fdw_performance()
RETURNS TABLE(
    test_name TEXT,
    execution_time_ms NUMERIC,
    row_count BIGINT
) AS $$
DECLARE
    start_time TIMESTAMP;
    end_time TIMESTAMP;
BEGIN
    -- Test 1: Count officers
    start_time := clock_timestamp();
    SELECT COUNT(*) INTO row_count FROM officers_live;
    end_time := clock_timestamp();
    test_name := 'Count Officers (FDW)';
    execution_time_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time));
    RETURN NEXT;

    -- Test 2: Count customers
    start_time := clock_timestamp();
    SELECT COUNT(*) INTO row_count FROM customers_live;
    end_time := clock_timestamp();
    test_name := 'Count Customers (FDW)';
    execution_time_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time));
    RETURN NEXT;

    -- Test 3: Count loans
    start_time := clock_timestamp();
    SELECT COUNT(*) INTO row_count FROM loans_base_live;
    end_time := clock_timestamp();
    test_name := 'Count Loans (FDW)';
    execution_time_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time));
    RETURN NEXT;

    -- Test 4: Get single officer by ID
    start_time := clock_timestamp();
    SELECT COUNT(*) INTO row_count FROM officers_live WHERE officer_id = '1';
    end_time := clock_timestamp();
    test_name := 'Get Officer by ID (FDW)';
    execution_time_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time));
    RETURN NEXT;

    -- Test 5: Get loans for specific officer
    start_time := clock_timestamp();
    SELECT COUNT(*) INTO row_count FROM loans_base_live WHERE officer_id = '1';
    end_time := clock_timestamp();
    test_name := 'Get Loans by Officer (FDW)';
    execution_time_ms := EXTRACT(MILLISECONDS FROM (end_time - start_time));
    RETURN NEXT;

    RETURN;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- VERIFICATION QUERIES
-- ============================================================================

-- Run these queries to verify the setup:

-- 1. Check that foreign server is created
-- SELECT * FROM pg_foreign_server WHERE srvname = 'django_db';

-- 2. Check that user mapping exists
-- SELECT * FROM pg_user_mappings WHERE srvname = 'django_db';

-- 3. Check that foreign tables are imported
-- SELECT foreign_table_schema, foreign_table_name
-- FROM information_schema.foreign_tables
-- WHERE foreign_table_schema = 'django_db';

-- 4. Test data access
-- SELECT COUNT(*) FROM officers_live;
-- SELECT COUNT(*) FROM customers_live;
-- SELECT COUNT(*) FROM loans_base_live;

-- 5. Compare data counts
-- SELECT * FROM data_validation_officers;
-- SELECT * FROM data_validation_customers;
-- SELECT * FROM data_validation_loans;

-- 6. Test performance
-- SELECT * FROM test_fdw_performance();

-- ============================================================================
-- ROLLBACK INSTRUCTIONS
-- ============================================================================

-- To rollback this migration:
-- DROP VIEW IF EXISTS data_validation_loans CASCADE;
-- DROP VIEW IF EXISTS data_validation_customers CASCADE;
-- DROP VIEW IF EXISTS data_validation_officers CASCADE;
-- DROP FUNCTION IF EXISTS test_fdw_performance();
-- DROP VIEW IF EXISTS loans_base_live CASCADE;
-- DROP VIEW IF EXISTS customers_live CASCADE;
-- DROP VIEW IF EXISTS officers_live CASCADE;
-- DROP SCHEMA IF EXISTS django_db CASCADE;
-- DROP USER MAPPING IF EXISTS FOR metricsuser SERVER django_db;
-- DROP SERVER IF EXISTS django_db CASCADE;
-- DROP EXTENSION IF EXISTS postgres_fdw CASCADE;

-- ============================================================================
-- NOTES
-- ============================================================================

-- 1. This migration sets up READ-ONLY access to Django database
-- 2. No data is modified in Django database
-- 3. Views provide real-time data from Django
-- 4. Computed fields (DPD, outstanding balances, etc.) remain in local loans table
-- 5. ETL process should continue running until Phase 3 (Backend Refactoring) is complete
-- 6. Performance should be monitored after deployment

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

