ALTER TABLE conversations ADD COLUMN kind TEXT NOT NULL DEFAULT 'direct';
ALTER TABLE conversations ADD COLUMN name TEXT;
ALTER TABLE conversations ALTER COLUMN direct_key DROP NOT NULL;
ALTER TABLE conversations ADD CONSTRAINT conversations_kind_check CHECK (kind IN ('direct', 'group'));
ALTER TABLE conversations ADD CONSTRAINT conversations_details_check CHECK (
    (kind = 'direct' AND direct_key IS NOT NULL AND name IS NULL)
    OR (kind = 'group' AND direct_key IS NULL AND name IS NOT NULL AND char_length(btrim(name)) BETWEEN 1 AND 60)
);
