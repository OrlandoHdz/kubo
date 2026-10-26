-- Consultas del panel de administración para la bandeja de escalamientos.
-- Usan el pool principal de la aplicación (rol con permisos de lectura/escritura).

-- name: ListarEscalamientosChat :many
SELECT
    e.id,
    e.conversation_id,
    e.usuario_id,
    e.motivo,
    e.estado,
    to_char(e.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') AS created_at,
    to_char(e.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') AS updated_at,
    e.contacto_nombre,
    e.contacto_email,
    e.contacto_telefono,
    e.nota_admin,
    e.atendida_por,
    coalesce(ua.email, '') AS atendida_por_email,
    coalesce(c.cliente_id, 0)::int AS cliente_id,
    coalesce(cl.nombre_comercial, '') AS cliente_nombre,
    coalesce(u.email, '') AS usuario_email
FROM chat_escalations e
LEFT JOIN chat_conversations c ON c.id = e.conversation_id
LEFT JOIN clientes cl ON cl.id = c.cliente_id
LEFT JOIN usuarios u ON u.id = e.usuario_id
LEFT JOIN usuarios ua ON ua.id = e.atendida_por
WHERE (sqlc.narg(estado)::text IS NULL OR e.estado = sqlc.narg(estado)::text)
ORDER BY e.created_at DESC
LIMIT LEAST(GREATEST(sqlc.arg(limite)::int, 1), 100);

-- name: ObtenerEscalamientoChat :one
SELECT
    e.id,
    e.conversation_id,
    e.usuario_id,
    e.motivo,
    e.estado,
    to_char(e.created_at, 'YYYY-MM-DD"T"HH24:MI:SS') AS created_at,
    to_char(e.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') AS updated_at,
    e.contacto_nombre,
    e.contacto_email,
    e.contacto_telefono,
    e.nota_admin,
    e.atendida_por,
    coalesce(ua.email, '') AS atendida_por_email,
    coalesce(c.cliente_id, 0)::int AS cliente_id,
    coalesce(cl.nombre_comercial, '') AS cliente_nombre,
    coalesce(u.email, '') AS usuario_email
FROM chat_escalations e
LEFT JOIN chat_conversations c ON c.id = e.conversation_id
LEFT JOIN clientes cl ON cl.id = c.cliente_id
LEFT JOIN usuarios u ON u.id = e.usuario_id
LEFT JOIN usuarios ua ON ua.id = e.atendida_por
WHERE e.id = sqlc.arg(id)::bigint
LIMIT 1;

-- name: ListarMensajesConversacionChat :many
SELECT
    role,
    content,
    to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS') AS created_at
FROM chat_messages
WHERE conversation_id = sqlc.arg(conversation_id)::text
ORDER BY created_at ASC, id ASC
LIMIT 200;

-- name: ActualizarEstadoEscalamientoChat :one
UPDATE chat_escalations
SET
    estado = sqlc.arg(estado)::text,
    nota_admin = sqlc.arg(nota_admin)::text,
    atendida_por = sqlc.narg(atendida_por)::int,
    updated_at = CURRENT_TIMESTAMP
WHERE id = sqlc.arg(id)::bigint
RETURNING
    id,
    conversation_id,
    estado,
    nota_admin,
    atendida_por,
    to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS') AS updated_at;
