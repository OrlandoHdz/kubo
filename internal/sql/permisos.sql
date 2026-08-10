-- name: ListarPermisos :many
-- Lista todos los módulos/permisos del menú del Panel de Control
SELECT id, clave, nombre, grupo, orden
FROM permisos
ORDER BY orden ASC, id ASC;

-- name: ListarPermisosDeUsuario :many
-- Lista los módulos con el estado (activo/inactivo) configurado para un usuario
SELECT p.id, p.clave, p.nombre, p.grupo, p.orden, COALESCE(up.activo, FALSE) AS activo
FROM permisos p
LEFT JOIN usuario_permisos up ON up.permiso_id = p.id AND up.usuario_id = $1
ORDER BY p.orden ASC, p.id ASC;

-- name: ListarPermisosActivosDeUsuario :many
-- Claves de los módulos activos de un usuario (para el login/perfil)
SELECT p.clave
FROM permisos p
JOIN usuario_permisos up ON up.permiso_id = p.id AND up.usuario_id = $1
WHERE up.activo = TRUE
ORDER BY p.orden ASC, p.id ASC;

-- name: ContarPermisosConfiguradosDeUsuario :one
-- Cuenta cuántos permisos han sido configurados explícitamente para un usuario
SELECT COUNT(*) FROM usuario_permisos WHERE usuario_id = $1;

-- name: ActualizarPermisoUsuario :exec
-- Activa/desactiva un módulo para un usuario (upsert)
INSERT INTO usuario_permisos (usuario_id, permiso_id, activo, updated_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (usuario_id, permiso_id)
DO UPDATE SET
    activo = EXCLUDED.activo,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = EXCLUDED.updated_by;
