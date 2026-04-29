-- name: CrearSolicitudRegistroNuevoCliente :one
INSERT INTO solicitud_registro_nuevo_cliente (
    nombre_comercial,
    razon_social,
    rfc,
    tipo_contribuyente,
    calle,
    numero,
    colonia,
    ciudad,
    estado,
    cp,
    nombre_contacto,
    puesto_contacto,
    correo_contacto,
    telefono_contacto,
    comentarios,
    constancia_sat_url
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
) RETURNING id;

-- name: ListarSolicitudesPendientes :many
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE solicitud_estado = 'Pendiente' AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ObtenerSolicitudByID :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE id = $1 AND deleted_at IS NULL;

-- name: ActualizarSolicitudEstado :exec
UPDATE solicitud_registro_nuevo_cliente
SET 
    solicitud_estado = $2,
    observacion = $3,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $4
WHERE id = $1;

-- name: SoftDeleteSolicitud :exec
UPDATE solicitud_registro_nuevo_cliente
SET 
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1;                                              

-- name: BuscarSolicitudByRFC :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE rfc = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByNombreComercial :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE nombre_comercial = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByNombreContacto :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE nombre_contacto = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByCorreoContacto :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE correo_contacto = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByTelefonoContacto :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE telefono_contacto = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByRazonSocial :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE razon_social = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByCiudad :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE ciudad = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByCP :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE cp = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByColonia :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE colonia = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByCalle :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE calle = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByNumero :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE numero = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByPuestoContacto :one
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE puesto_contacto = $1 AND deleted_at IS NULL;

-- name: BuscarSolicitudByEstado :many
SELECT * FROM solicitud_registro_nuevo_cliente
WHERE solicitud_estado = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;
