-- Add is_system_admin to users. The GORM model (types.User.IsSystemAdmin)
-- carries this column but the monolithic 000000_init.up.sql predates the
-- SystemAdmin feature, so fresh SQLite databases lack it and INSERTs that
-- include the column fail ("table users has no column named is_system_admin"),
-- breaking auto-setup / registration on the personal (SQLite) edition.
ALTER TABLE users ADD COLUMN is_system_admin BOOLEAN NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_is_system_admin ON users(is_system_admin);
