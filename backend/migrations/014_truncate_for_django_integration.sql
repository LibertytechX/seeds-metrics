-- Migration 014: Truncate SeedsMetrics Database for Django Integration
-- Purpose: Clear all existing data to prepare for Django FDW integration
-- Date: 2025-11-04
-- WARNING: This will DELETE ALL DATA from the SeedsMetrics database
-- IMPORTANT: A full backup should be created before running this migration

-- ============================================================================
-- PHASE 1: VERIFY BACKUP EXISTS
-- ============================================================================

-- Before running this migration, ensure you have created a backup:
-- pg_dump -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 \
--   -U seedsuser -d seedsmetrics --format=custom \
--   -f seedsmetrics_backup_$(date +%Y%m%d_%H%M%S).dump

-- ============================================================================
-- PHASE 2: TRUNCATE ALL DATA TABLES
-- ============================================================================

-- Disable triggers temporarily to avoid cascade issues
SET session_replication_role = 'replica';

-- Truncate core business tables
TRUNCATE TABLE loans CASCADE;
TRUNCATE TABLE repayments CASCADE;
TRUNCATE TABLE officers CASCADE;
TRUNCATE TABLE customers CASCADE;
TRUNCATE TABLE loan_schedule CASCADE;

-- Truncate computed/aggregation tables
TRUNCATE TABLE officer_metrics_daily CASCADE;
TRUNCATE TABLE branch_metrics_daily CASCADE;
TRUNCATE TABLE dpd_transitions CASCADE;
TRUNCATE TABLE par15_snapshots CASCADE;

-- Truncate supporting tables
TRUNCATE TABLE team_members CASCADE;

-- Truncate ETL tracking tables (if they exist)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'etl_sync_log') THEN
        TRUNCATE TABLE etl_sync_log CASCADE;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'etl_errors') THEN
        TRUNCATE TABLE etl_errors CASCADE;
    END IF;
    
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'etl_last_sync') THEN
        TRUNCATE TABLE etl_last_sync CASCADE;
    END IF;
END $$;

-- Re-enable triggers
SET session_replication_role = 'origin';

-- ============================================================================
-- PHASE 3: VERIFY TRUNCATION
-- ============================================================================

-- Create verification view
CREATE OR REPLACE VIEW truncation_verification AS
SELECT
    'loans' as table_name,
    COUNT(*) as row_count
FROM loans
UNION ALL
SELECT 'repayments', COUNT(*) FROM repayments
UNION ALL
SELECT 'officers', COUNT(*) FROM officers
UNION ALL
SELECT 'customers', COUNT(*) FROM customers
UNION ALL
SELECT 'loan_schedule', COUNT(*) FROM loan_schedule
UNION ALL
SELECT 'officer_metrics_daily', COUNT(*) FROM officer_metrics_daily
UNION ALL
SELECT 'branch_metrics_daily', COUNT(*) FROM branch_metrics_daily
UNION ALL
SELECT 'dpd_transitions', COUNT(*) FROM dpd_transitions
UNION ALL
SELECT 'par15_snapshots', COUNT(*) FROM par15_snapshots
UNION ALL
SELECT 'team_members', COUNT(*) FROM team_members
ORDER BY table_name;

-- Display verification results
SELECT * FROM truncation_verification;

-- ============================================================================
-- PHASE 4: RESET SEQUENCES (if needed)
-- ============================================================================

-- Reset auto-increment sequences to start from 1
DO $$
DECLARE
    seq_record RECORD;
BEGIN
    FOR seq_record IN
        SELECT sequence_name
        FROM information_schema.sequences
        WHERE sequence_schema = 'public'
    LOOP
        EXECUTE 'ALTER SEQUENCE ' || seq_record.sequence_name || ' RESTART WITH 1';
    END LOOP;
END $$;

-- ============================================================================
-- PHASE 5: PRESERVE SCHEMA AND TRIGGERS
-- ============================================================================

-- Verify that all triggers are still in place
SELECT
    trigger_name,
    event_object_table,
    action_statement
FROM information_schema.triggers
WHERE trigger_schema = 'public'
ORDER BY event_object_table, trigger_name;

-- Verify that all functions are still in place
SELECT
    routine_name,
    routine_type
FROM information_schema.routines
WHERE routine_schema = 'public'
AND routine_type = 'FUNCTION'
ORDER BY routine_name;

-- ============================================================================
-- NOTES
-- ============================================================================

-- 1. This migration ONLY truncates data, it does NOT drop tables or schema
-- 2. All triggers, functions, and constraints remain intact
-- 3. The database structure is preserved for the Django FDW integration
-- 4. After this migration, run migration 013 to set up Foreign Data Wrapper
-- 5. A full backup should be created BEFORE running this migration

-- ============================================================================
-- ROLLBACK INSTRUCTIONS
-- ============================================================================

-- To restore from backup:
-- pg_restore -h generaldb-do-user-9489371-0.k.db.ondigitalocean.com -p 25060 \
--   -U seedsuser -d seedsmetrics --clean --if-exists \
--   seedsmetrics_backup_YYYYMMDD_HHMMSS.dump

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

