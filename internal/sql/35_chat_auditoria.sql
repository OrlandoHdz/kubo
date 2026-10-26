-- Migración 35: tablas de auditoría del chat.
-- Son append-only; el rol chat_audit solo recibe INSERT.

CREATE TABLE IF NOT EXISTS chat_conversations (
    id TEXT PRIMARY KEY,
    usuario_id INTEGER,
    cliente_id INTEGER,
    canal TEXT NOT NULL DEFAULT 'web',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    usuario_id INTEGER,
    role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'tool', 'system')),
    content TEXT NOT NULL DEFAULT '',
    tool_calls JSONB,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation
    ON chat_messages (conversation_id, created_at);

CREATE TABLE IF NOT EXISTS chat_escalations (
    id BIGSERIAL PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    usuario_id INTEGER,
    motivo TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT 'pendiente'
        CHECK (estado IN ('pendiente', 'atendida', 'cerrada')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_chat_escalations_estado
    ON chat_escalations (estado, created_at);

COMMENT ON TABLE chat_conversations
    IS 'Conversaciones del chat; append-only para el rol de auditoría.';
COMMENT ON TABLE chat_messages
    IS 'Mensajes y tool calls del chat; tokens de entrada/salida por turno.';
COMMENT ON TABLE chat_escalations
    IS 'Solicitudes de atención humana; sustituye un UPDATE sobre la conversación.';
