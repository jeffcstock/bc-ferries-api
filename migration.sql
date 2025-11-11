-- Migration to add date column to capacity_routes and non_capacity_routes tables
-- Run this script after deploying the code changes

-- Add date column to capacity_routes table
ALTER TABLE capacity_routes
ADD COLUMN date DATE NOT NULL DEFAULT CURRENT_DATE;

-- Add date column to non_capacity_routes table
ALTER TABLE non_capacity_routes
ADD COLUMN date DATE NOT NULL DEFAULT CURRENT_DATE;

-- Verify the changes
SELECT column_name, data_type, is_nullable, column_default
FROM information_schema.columns
WHERE table_name IN ('capacity_routes', 'non_capacity_routes')
ORDER BY table_name, ordinal_position;
