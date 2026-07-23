DROP INDEX IF EXISTS idx_users_is_system_admin;
ALTER TABLE users DROP COLUMN is_system_admin;
