-- Remove user_id from jobs
ALTER TABLE jobs DROP COLUMN user_id;

-- Drop indexes
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;

-- Drop trigger
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop users table
DROP TABLE IF EXISTS users;
