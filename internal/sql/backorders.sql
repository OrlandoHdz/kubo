-- name: CrearBackorder :one
INSERT INTO backorders (
    folio,
    cliente_id,
    usuario_id,
    estado_backorder,
    metodo_pago,
    subtotal,
    iva,
    total_orden,
    guia_backorder,
    notas_admin_backorder,
    pedido_origen_id,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: CrearBackorderDetalle :one
INSERT INTO backorder_detalles (
    backorder_id,
    variante_id,
    cantidad,
    precio_unitario_aplicado,
    disponible,
    created_by
) VALUES (
    $1, $2, $3, $4, FALSE, $5
) RETURNING *;

-- name: GetBackorder :one
SELECT b.*, COALESCE(p.folio, '') AS pedido_origen_folio
FROM backorders b
LEFT JOIN pedidos p ON b.pedido_origen_id = p.id
WHERE b.id = $1 AND b.deleted_at IS NULL LIMIT 1;

-- name: GetBackorderByFolio :one
SELECT b.*, COALESCE(p.folio, '') AS pedido_origen_folio
FROM backorders b
LEFT JOIN pedidos p ON b.pedido_origen_id = p.id
WHERE b.folio = $1 AND b.deleted_at IS NULL LIMIT 1;

-- name: GetBackorderDetalles :many
SELECT d.id, d.backorder_id, d.variante_id, d.cantidad, d.precio_unitario_aplicado, d.disponible,
       d.created_at, d.updated_at, d.deleted_at, d.created_by, d.updated_by, d.deleted_by,
       v.sku AS variante_sku,
       p.descripcion AS padre_descripcion,
       p.descripcion_extendida AS padre_descripcion_extendida,
       p.foto_url AS padre_foto_url
FROM backorder_detalles d
LEFT JOIN productos_variantes v ON d.variante_id = v.id
LEFT JOIN productos_padre p ON v.padre_id = p.id
WHERE d.backorder_id = $1 AND d.deleted_at IS NULL;

-- name: ListarBackorders :many
SELECT b.*, COALESCE(p.folio, '') AS pedido_origen_folio
FROM backorders b
LEFT JOIN pedidos p ON b.pedido_origen_id = p.id
WHERE b.deleted_at IS NULL
ORDER BY b.created_at DESC;

-- name: ListarBackordersPorRango :many
SELECT b.*, COALESCE(p.folio, '') AS pedido_origen_folio
FROM backorders b
LEFT JOIN pedidos p ON b.pedido_origen_id = p.id
WHERE b.deleted_at IS NULL
  AND b.created_at >= $1
  AND b.created_at <= $2
ORDER BY b.created_at DESC;

-- name: ListarBackordersPorCliente :many
SELECT b.*, COALESCE(p.folio, '') AS pedido_origen_folio
FROM backorders b
LEFT JOIN pedidos p ON b.pedido_origen_id = p.id
WHERE b.cliente_id = $1 AND b.deleted_at IS NULL
ORDER BY b.created_at DESC;

-- name: ListarBackordersPorClienteRango :many
SELECT b.*, COALESCE(p.folio, '') AS pedido_origen_folio
FROM backorders b
LEFT JOIN pedidos p ON b.pedido_origen_id = p.id
WHERE b.cliente_id = $1 AND b.deleted_at IS NULL
  AND b.created_at >= $2
  AND b.created_at <= $3
ORDER BY b.created_at DESC;

-- name: ActualizarEstadoBackorder :one
UPDATE backorders
SET
    estado_backorder = $2,
    guia_backorder = $4,
    notas_admin_backorder = $5,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: MarcarDisponibleBackorderDetalle :one
UPDATE backorder_detalles
SET
    disponible = TRUE,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1 AND backorder_id = $2 AND deleted_at IS NULL
RETURNING *;

-- name: SetPedidoOrigenBackorder :one
UPDATE backorders
SET
    pedido_origen_id = $2,
    estado_backorder = 'Convertido',
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;
