-- Setup script for external PostgreSQL database
-- Run this as the postgres superuser
-- Note: Using existing 'postgres' user, only creating the database

-- Create database if not exists
SELECT 'CREATE DATABASE seedsmetrics OWNER postgres'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'seedsmetrics')\gexec

-- Grant privileges (postgres user already has all privileges as owner)
GRANT ALL PRIVILEGES ON DATABASE seedsmetrics TO postgres;

-- Display success message
\echo ''
\echo '✅ Database created successfully!'
\echo ''
\echo 'Database: seedsmetrics'
\echo 'User: postgres'
\echo 'Password: 19sedimat54'
\echo ''
\echo 'Next step: Apply the schema with:'
\echo 'PGPASSWORD=19sedimat54 psql -h localhost -p 5433 -U postgres -d seedsmetrics -f backend/migrations/001_initial_schema.sql'
\echo ''

