DROP INDEX IF EXISTS users_username_lower_idx;

DELETE FROM users WHERE is_guest = true;

ALTER TABLE users DROP COLUMN is_guest;
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
