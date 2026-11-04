DROP INDEX IF EXISTS users_guest_expiry_idx;
ALTER TABLE users DROP COLUMN guest_expires_at;
ALTER TABLE users DROP COLUMN google_subject;
