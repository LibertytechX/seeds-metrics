-- Migration 026: Create holiday table
-- Purpose: Sync holiday data from Django database to Seeds Metrics
-- Source: loans_holiday table from Django (savings database)
-- Records: 1015 holidays

BEGIN;

-- Create the holiday table
CREATE TABLE IF NOT EXISTS holiday (
    id BIGINT PRIMARY KEY,
    date DATE,
    name VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    agent_id BIGINT,
    branch_id BIGINT,
    created_by_id BIGINT,
    type VARCHAR(255) NOT NULL,
    salary_waver BOOLEAN NOT NULL DEFAULT FALSE
);

-- Create indexes for common queries
CREATE INDEX IF NOT EXISTS idx_holiday_date ON holiday(date);
CREATE INDEX IF NOT EXISTS idx_holiday_type ON holiday(type);
CREATE INDEX IF NOT EXISTS idx_holiday_agent_id ON holiday(agent_id);
CREATE INDEX IF NOT EXISTS idx_holiday_branch_id ON holiday(branch_id);
CREATE INDEX IF NOT EXISTS idx_holiday_created_at ON holiday(created_at);

-- Add comment to table
COMMENT ON TABLE holiday IS 'Holiday calendar synced from Django loans_holiday table. Contains company-wide and agent-specific holidays.';
COMMENT ON COLUMN holiday.id IS 'Primary key from Django';
COMMENT ON COLUMN holiday.date IS 'Holiday date';
COMMENT ON COLUMN holiday.name IS 'Holiday name (e.g., Good Friday, Easter Monday)';
COMMENT ON COLUMN holiday.type IS 'Holiday type: company or agent-specific';
COMMENT ON COLUMN holiday.salary_waver IS 'Whether salary is waived on this holiday';
COMMENT ON COLUMN holiday.agent_id IS 'Agent ID if this is an agent-specific holiday';
COMMENT ON COLUMN holiday.branch_id IS 'Branch ID if this is a branch-specific holiday';
COMMENT ON COLUMN holiday.created_by_id IS 'User ID who created this holiday record';

COMMIT;

