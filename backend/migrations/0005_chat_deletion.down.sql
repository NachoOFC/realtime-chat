DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM conversation_members WHERE cleared_at IS NOT NULL) THEN
        RAISE EXCEPTION 'No se puede revertir mientras existan historiales eliminados: volverían a mostrarse.';
    END IF;
END $$;
ALTER TABLE conversation_members DROP COLUMN is_hidden;
ALTER TABLE conversation_members DROP COLUMN cleared_at;
