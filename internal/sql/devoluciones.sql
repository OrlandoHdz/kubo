-- name: CrearDevolucion :one
INSERT INTO devoluciones_garantias (
    folio,
    cliente_id,
    pedido_folio,
    tipo,
    numeros_parte,
    cantidades,
    nota_cliente,
    evidencias,
    estatus,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, 'Pendiente', $9
) RETURNING *;

-- name: GetDevolucion :one
SELECT * FROM devoluciones_garantias
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListDevolucionesByCliente :many
SELECT * FROM devoluciones_garantias
WHERE cliente_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListDevolucionesByClienteRango :many
SELECT * FROM devoluciones_garantias
WHERE cliente_id = $1 AND deleted_at IS NULL
  AND created_at >= $2
  AND created_at <= $3
ORDER BY created_at DESC;

-- name: ListDevoluciones :many
SELECT * FROM devoluciones_garantias
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListDevolucionesPorRango :many
SELECT * FROM devoluciones_garantias
WHERE deleted_at IS NULL
  AND created_at >= $1
  AND created_at <= $2
ORDER BY created_at DESC;

-- name: ActualizarEstatusDevolucion :one
UPDATE devoluciones_garantias
SET
    estatus = $2,
    nota_administrador = $3,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;
