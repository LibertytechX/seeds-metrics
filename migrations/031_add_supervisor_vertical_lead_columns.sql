-- Migration: Add supervisor and vertical lead columns to officers table
-- Date: 2025-11-09
-- Description: Add supervisor_email, supervisor_name, vertical_lead_email, vertical_lead_name columns
--              to support organizational hierarchy display in Agent Performance table

-- Add new columns
ALTER TABLE officers 
ADD COLUMN IF NOT EXISTS supervisor_email VARCHAR(255),
ADD COLUMN IF NOT EXISTS supervisor_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS vertical_lead_email VARCHAR(255),
ADD COLUMN IF NOT EXISTS vertical_lead_name VARCHAR(255);

-- Create indexes for performance (filtering and sorting)
CREATE INDEX IF NOT EXISTS idx_officers_supervisor_email ON officers(supervisor_email);
CREATE INDEX IF NOT EXISTS idx_officers_vertical_lead_email ON officers(vertical_lead_email);

-- Add comments for documentation
COMMENT ON COLUMN officers.supervisor_email IS 'Email address of the branch supervisor for this officer';
COMMENT ON COLUMN officers.supervisor_name IS 'Name of the branch supervisor for this officer';
COMMENT ON COLUMN officers.vertical_lead_email IS 'Email address of the vertical lead for this officer';
COMMENT ON COLUMN officers.vertical_lead_name IS 'Name of the vertical lead for this officer';

