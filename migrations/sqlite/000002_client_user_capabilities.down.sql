-- SQLite cannot DROP COLUMN portably in older versions; leave no-op with note.
-- Downgrade path for desktop SQLite is recreate-from-backup / re-init.
SELECT 1;
