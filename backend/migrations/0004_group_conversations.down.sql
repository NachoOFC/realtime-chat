DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM conversations WHERE kind = 'group') THEN
        RAISE EXCEPTION 'No se puede revertir mientras existan grupos. Conserva sus datos antes de continuar.';
    END IF;
END $$;
ALTER TABLE conversations DROP CONSTRAINT conversations_details_check;
ALTER TABLE conversations DROP CONSTRAINT conversations_kind_check;
ALTER TABLE conversations ALTER COLUMN direct_key SET NOT NULL;
ALTER TABLE conversations DROP COLUMN name;
ALTER TABLE conversations DROP COLUMN kind;
