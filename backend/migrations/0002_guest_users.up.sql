ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE users ADD COLUMN is_guest BOOLEAN NOT NULL DEFAULT false;

CREATE UNIQUE INDEX users_username_lower_idx ON users (lower(username));
