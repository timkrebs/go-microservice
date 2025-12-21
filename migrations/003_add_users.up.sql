-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    username VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(username);

-- Add user_id to jobs table
ALTER TABLE jobs ADD COLUMN user_id UUID REFERENCES users(id) ON DELETE CASCADE;
CREATE INDEX idx_jobs_user_id ON jobs(user_id);

-- Update existing jobs to have a system user (for backward compatibility)
-- Create a system user for existing anonymous jobs
INSERT INTO users (id, email, password_hash, username)
VALUES ('00000000-0000-0000-0000-000000000000', 'system@internal', '', 'system')
ON CONFLICT (id) DO NOTHING;

-- Assign existing jobs to system user
UPDATE jobs SET user_id = '00000000-0000-0000-0000-000000000000' WHERE user_id IS NULL;

-- Make user_id NOT NULL after backfilling
ALTER TABLE jobs ALTER COLUMN user_id SET NOT NULL;

-- Create updated_at trigger for users
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
