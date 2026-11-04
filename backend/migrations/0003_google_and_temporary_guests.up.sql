ALTER TABLE users ADD COLUMN google_subject TEXT UNIQUE;
ALTER TABLE users ADD COLUMN guest_expires_at TIMESTAMPTZ;
UPDATE users SET guest_expires_at = now() + interval '24 hours' WHERE is_guest;
CREATE INDEX users_guest_expiry_idx ON users (guest_expires_at) WHERE is_guest;
