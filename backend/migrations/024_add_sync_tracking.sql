-- ============================================================================
-- Migration: 024_add_sync_tracking.sql
-- Description: Add sync tracking table to track incremental syncs
--
-- Purpose: Enable incremental syncing of repayments and other entities
--          by tracking the last sync timestamp for each entity type
--
-- Date: 2025-11-05
-- ============================================================================

-- Create sync_tracking table
CREATE TABLE IF NOT EXISTS sync_tracking (
    sync_id SERIAL PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL UNIQUE,  -- 'repayments', 'loans', 'customers', 'officers'
    last_sync_timestamp TIMESTAMP,             -- NULL means never synced
    last_synced_record_id VARCHAR(50),         -- For reference
    sync_count INTEGER DEFAULT 0,              -- Total number of syncs
    last_sync_duration_ms INTEGER,             -- Duration of last sync in milliseconds
    last_sync_records_count INTEGER,           -- Number of records synced in last sync
    last_sync_errors_count INTEGER,            -- Number of errors in last sync
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for faster lookups
CREATE INDEX idx_sync_tracking_entity ON sync_tracking(entity_type);

-- Initialize sync tracking for repayments (set to NULL to sync all)
INSERT INTO sync_tracking (entity_type, last_sync_timestamp, sync_count)
VALUES ('repayments', NULL, 0)
ON CONFLICT (entity_type) DO NOTHING;

-- Initialize sync tracking for loans
INSERT INTO sync_tracking (entity_type, last_sync_timestamp, sync_count)
VALUES ('loans', NULL, 0)
ON CONFLICT (entity_type) DO NOTHING;

-- Initialize sync tracking for customers
INSERT INTO sync_tracking (entity_type, last_sync_timestamp, sync_count)
VALUES ('customers', NULL, 0)
ON CONFLICT (entity_type) DO NOTHING;

-- Initialize sync tracking for officers
INSERT INTO sync_tracking (entity_type, last_sync_timestamp, sync_count)
VALUES ('officers', NULL, 0)
ON CONFLICT (entity_type) DO NOTHING;

-- Create function to update sync tracking
CREATE OR REPLACE FUNCTION update_sync_tracking(
    p_entity_type VARCHAR(50),
    p_records_count INTEGER,
    p_errors_count INTEGER,
    p_duration_ms INTEGER
)
RETURNS void AS $$
BEGIN
    UPDATE sync_tracking
    SET
        last_sync_timestamp = CURRENT_TIMESTAMP,
        sync_count = sync_count + 1,
        last_sync_records_count = p_records_count,
        last_sync_errors_count = p_errors_count,
        last_sync_duration_ms = p_duration_ms,
        updated_at = CURRENT_TIMESTAMP
    WHERE entity_type = p_entity_type;
END;
$$ LANGUAGE plpgsql;

-- Create function to get last sync timestamp
CREATE OR REPLACE FUNCTION get_last_sync_timestamp(p_entity_type VARCHAR(50))
RETURNS TIMESTAMP AS $$
DECLARE
    v_timestamp TIMESTAMP;
BEGIN
    SELECT last_sync_timestamp INTO v_timestamp
    FROM sync_tracking
    WHERE entity_type = p_entity_type;
    
    RETURN v_timestamp;
END;
$$ LANGUAGE plpgsql;

-- Show initial state
SELECT 
    entity_type,
    last_sync_timestamp,
    sync_count,
    created_at
FROM sync_tracking
ORDER BY entity_type;

