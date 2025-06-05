-- Health Tracker Database Initialization Script
-- This script runs when PostgreSQL container starts for the first time

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search optimization

-- Create application user (if different from default)
-- Uncomment and modify if you need a separate application user
-- DO $$ 
-- BEGIN
--     IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'health_app') THEN
--         CREATE ROLE health_app WITH LOGIN PASSWORD 'app_password';
--         GRANT CONNECT ON DATABASE health_tracker TO health_app;
--         GRANT USAGE ON SCHEMA public TO health_app;
--         GRANT CREATE ON SCHEMA public TO health_app;
--     END IF;
-- END $$;

-- Set timezone
SET timezone = 'Asia/Tokyo';

-- Create basic tables structure (example)
-- Uncomment and modify according to your application needs
/*
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS health_records (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    record_date DATE NOT NULL,
    weight DECIMAL(5,2),
    height DECIMAL(5,2),
    blood_pressure_systolic INTEGER,
    blood_pressure_diastolic INTEGER,
    heart_rate INTEGER,
    notes TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_health_records_user_id ON health_records(user_id);
CREATE INDEX IF NOT EXISTS idx_health_records_date ON health_records(record_date);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
*/

-- Log successful initialization
\echo 'Health Tracker database initialized successfully'
\echo 'Extensions enabled: uuid-ossp, pg_stat_statements, pg_trgm'
\echo 'Timezone set to: Asia/Tokyo'
