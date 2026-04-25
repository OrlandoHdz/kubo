-- name: CrearUsuario :one
-- Registra un nuevo usuario (Staff o Cliente). 
-- Si cliente_id es NULL, es personal de Alialloys.
INSERT INTO usuarios (
    cliente_id, 
    email, 
    password_hash, 
    rol, 
    is_active, 
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetUsuarioByID :one
SELECT * FROM usuarios 
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetUsuarioByEmail :one
-- Fundamental para el flujo de Login
SELECT * FROM usuarios 
WHERE email = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListarUsuarios :many
-- Lista todos los usuarios no borrados para el panel de control
SELECT * FROM usuarios 
WHERE deleted_at IS NULL 
ORDER BY created_at DESC;

-- name: ListarUsuariosPorCliente :many
-- Obtiene todos los contactos/compradores de una empresa específica
SELECT * FROM usuarios 
WHERE cliente_id = $1 AND deleted_at IS NULL;

-- name: ActualizarUsuario :one
-- Permite cambiar rol, email o estado de activación
UPDATE usuarios
SET 
    email = $2,
    rol = $3,
    is_active = $4,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $5
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActualizarPassword :exec
-- Query específica para cambio de contraseña por seguridad
UPDATE usuarios
SET 
    password_hash = $2,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $1
WHERE id = $1;

-- name: SoftDeleteUsuario :exec
-- Borrado lógico registrando quién ejecutó la baja
UPDATE usuarios
SET 
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2,
    is_active = FALSE
WHERE id = $1;