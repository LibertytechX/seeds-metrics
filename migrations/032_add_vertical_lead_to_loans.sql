-- Migration: Add vertical lead columns to loans table
-- Date: 2025-11-09
-- Description: Add vertical_lead_name and vertical_lead_email columns to loans table
--              to link each loan to its officer's vertical leadership structure

BEGIN;

-- Add vertical lead columns
ALTER TABLE loans 
ADD COLUMN IF NOT EXISTS vertical_lead_name VARCHAR(255),
ADD COLUMN IF NOT EXISTS vertical_lead_email VARCHAR(255);

-- Create indexes for query performance
CREATE INDEX IF NOT EXISTS idx_loans_vertical_lead_email 
ON loans(vertical_lead_email) 
WHERE vertical_lead_email IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_loans_vertical_lead_name 
ON loans(vertical_lead_name) 
WHERE vertical_lead_name IS NOT NULL;

-- Add comments for documentation
COMMENT ON COLUMN loans.vertical_lead_name IS 'Name of the vertical lead from the officer who disbursed this loan';
COMMENT ON COLUMN loans.vertical_lead_email IS 'Email of the vertical lead from the officer who disbursed this loan';

COMMIT;

-- Verification queries
SELECT 
    column_name, 
    data_type, 
    is_nullable,
    column_default
FROM information_schema.columns 
WHERE table_name = 'loans' 
AND column_name IN ('vertical_lead_name', 'vertical_lead_email')
ORDER BY column_name;

