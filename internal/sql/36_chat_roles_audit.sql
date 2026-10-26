-- Migración 36: rol de escritura exclusivo para auditoría.
-- Solo INSERT sobre las tablas de auditoría y USAGE de secuencias.
-- No tiene SELECT, UPDATE ni DELETE.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'chat_audit') THEN
        CREATE ROLE chat_audit LOGIN;
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO chat_audit;

DO $$
BEGIN
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO chat_audit', current_database());
EXCEPTION
    WHEN insufficient_privilege THEN
        RAISE NOTICE 'No se pudo otorgar CONNECT a chat_audit; el administrador debe hacerlo manualmente.';
END $$;

GRANT INSERT ON
    chat_conversations,
    chat_messages,
    chat_escalations
TO chat_audit;

GRANT USAGE ON SEQUENCE
    chat_messages_id_seq,
    chat_escalations_id_seq
TO chat_audit;

ALTER ROLE chat_audit SET statement_timeout = '5s';
