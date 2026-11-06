ALTER TABLE conversation_members ADD COLUMN cleared_at TIMESTAMPTZ;
ALTER TABLE conversation_members ADD COLUMN is_hidden BOOLEAN NOT NULL DEFAULT false;
