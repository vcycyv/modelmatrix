-- ModelMatrix Database Initialization Script
-- Run this script as a PostgreSQL superuser to create the database

-- Create the database if it doesn't exist
SELECT 'CREATE DATABASE modelmatrix'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'modelmatrix')\gexec

-- Connect to the modelmatrix database
\c modelmatrix

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Grant privileges (adjust user as needed)
GRANT ALL PRIVILEGES ON DATABASE modelmatrix TO postgres;

-- Note: Tables will be created by GORM migrations
-- This script only ensures the database exists

