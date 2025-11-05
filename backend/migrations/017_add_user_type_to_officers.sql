-- Migration 017: Add user_type column to officers table
-- Purpose: Add user_type field to track officer/agent type (MERCHANT, AGENT, LOTTO_AGENT, etc.)
-- Date: 2025-11-04

-- Add user_type column to officers table
ALTER TABLE officers 
ADD COLUMN IF NOT EXISTS user_type VARCHAR(100);

-- Create index for user_type filtering
CREATE INDEX IF NOT EXISTS idx_officers_user_type ON officers(user_type);

-- Add comment to document the column
COMMENT ON COLUMN officers.user_type IS 'Type of user/officer: MERCHANT, AGENT, LOTTO_AGENT, STAFF_AGENT, etc.';

-- Display summary
SELECT 
    'Migration 017 completed' as status,
    COUNT(*) as total_officers,
    COUNT(user_type) as officers_with_user_type
FROM officers;

