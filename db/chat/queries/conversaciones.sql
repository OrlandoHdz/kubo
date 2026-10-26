-- Consultas de escritura para auditoría. Se ejecutan con el pool chat_audit.
-- No hay SELECT/UPDATE/DELETE para ese rol.

-- name: CrearConversacion :exec
INSERT INTO chat_conversations (
    id,
    usuario_id,
    cliente_id,
    canal
) VALUES (
    sqlc.arg(id)::text,
    sqlc.narg(usuario_id)::int,
    sqlc.narg(cliente_id)::int,
    'web'
);

-- name: CrearMensaje :exec
INSERT INTO chat_messages (
    conversation_id,
    usuario_id,
    role,
    content,
    tool_calls,
    input_tokens,
    output_tokens
) VALUES (
    sqlc.arg(conversation_id)::text,
    sqlc.narg(usuario_id)::int,
    sqlc.arg(role)::text,
    sqlc.arg(content)::text,
    sqlc.narg(tool_calls)::jsonb,
    sqlc.arg(input_tokens)::int,
    sqlc.arg(output_tokens)::int
);

-- name: CrearEscalamiento :exec
INSERT INTO chat_escalations (
    conversation_id,
    usuario_id,
    motivo,
    contacto_nombre,
    contacto_email,
    contacto_telefono
) VALUES (
    sqlc.arg(conversation_id)::text,
    sqlc.narg(usuario_id)::int,
    sqlc.arg(motivo)::text,
    sqlc.arg(contacto_nombre)::text,
    sqlc.arg(contacto_email)::text,
    sqlc.arg(contacto_telefono)::text
);
