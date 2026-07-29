-- name: CrearPedido :one
INSERT INTO pedidos (
    folio, 
    cliente_id, 
    usuario_id, 
    estado, 
    metodo_pago, 
    subtotal, 
    iva, 
    total_orden, 
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: CrearPedidoDetalle :one
INSERT INTO pedido_detalles (
    pedido_id, 
    variante_id, 
    cantidad, 
    precio_unitario_aplicado, 
    created_by
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetPedido :one
SELECT * FROM pedidos
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: GetPedidoByFolio :one
SELECT * FROM pedidos
WHERE folio = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListarPedidosDetalle :many
SELECT d.id, d.pedido_id, d.variante_id, d.cantidad, d.precio_unitario_aplicado, d.shipped_quantity, d.backorder_quantity, d.created_at, d.updated_at, d.deleted_at, d.created_by, d.updated_by, d.deleted_by,
       v.sku AS variante_sku,
       p.descripcion AS padre_descripcion,
       p.descripcion_extendida AS padre_descripcion_extendida,
       p.foto_url AS padre_foto_url
FROM pedido_detalles d
LEFT JOIN productos_variantes v ON d.variante_id = v.id
LEFT JOIN productos_padre p ON v.padre_id = p.id
WHERE d.pedido_id = $1 AND d.deleted_at IS NULL;

-- name: ListarPedidos :many
SELECT * FROM pedidos
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListarPedidosPorRango :many
SELECT * FROM pedidos
WHERE deleted_at IS NULL
  AND created_at >= $1
  AND created_at <= $2
ORDER BY created_at DESC;

-- name: ListarPedidosPorCliente :many
SELECT * FROM pedidos
WHERE cliente_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListarPedidosPorClienteRango :many
SELECT * FROM pedidos
WHERE cliente_id = $1 AND deleted_at IS NULL
  AND created_at >= $2
  AND created_at <= $3
ORDER BY created_at DESC;

-- name: ActualizarEstadoPedido :one
UPDATE pedidos
SET 
    estado = $2,
    guia = $4,
    notas_admin = $5,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CancelarDetallePedido :exec
UPDATE pedido_detalles
SET deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1 AND deleted_at IS NULL;
-- name: PedidoDentroVentanaModificacion :one
SELECT (CURRENT_TIMESTAMP - COALESCE(fecha_pedido, created_at)) < INTERVAL '2 hours' AS dentro_ventana
FROM pedidos WHERE id = $1 AND deleted_at IS NULL;

-- name: GetCommittedStock :one
SELECT COALESCE(SUM(pd.cantidad), 0)::INT as committed_stock
FROM pedido_detalles pd
JOIN pedidos p ON pd.pedido_id = p.id
WHERE pd.variante_id = $1 
  AND p.estado IN ('Pendiente', 'En Proceso') -- Estados que reservan stock
  AND pd.deleted_at IS NULL 
  AND p.deleted_at IS NULL;

-- name: ActualizarEnvioPedidoDetalle :one
UPDATE pedido_detalles
SET
    cantidad = $2,
    shipped_quantity = $2,
    backorder_quantity = $3,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: CrearModificacionPedido :one
INSERT INTO order_modifications (
    order_id,
    user_id,
    item_id,
    original_quantity,
    shipped_quantity,
    backorder_quantity,
    notes,
    backorder_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: ActualizarHasBackorderPedido :one
UPDATE pedidos
SET
    has_backorder = $2,
    estado = $3,
    guia = $5,
    notas_admin = $6,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $4
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ActualizarTotalesPedido :one
UPDATE pedidos
SET
    subtotal = $2,
    iva = $3,
    total_orden = $4,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $5
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: ListarModificacionesPorPedido :many
SELECT * FROM order_modifications
WHERE order_id = $1
ORDER BY created_at DESC;

-- name: ListarModificaciones :many
SELECT
    om.id,
    om.order_id,
    om.user_id,
    om.item_id,
    om.original_quantity,
    om.shipped_quantity,
    om.backorder_quantity,
    om.notes,
    om.created_at,
    om.backorder_id,
    p.folio AS pedido_folio,
    u.email AS usuario_email,
    pv.sku AS variante_sku,
    pp.descripcion AS producto_descripcion,
    COALESCE(b.folio, '') AS backorder_folio,
    COALESCE(b.total_orden, 0) AS backorder_total
FROM order_modifications om
LEFT JOIN pedidos p ON om.order_id = p.id
LEFT JOIN usuarios u ON om.user_id = u.id
LEFT JOIN pedido_detalles pd ON om.item_id = pd.id
LEFT JOIN productos_variantes pv ON pd.variante_id = pv.id
LEFT JOIN productos_padre pp ON pv.padre_id = pp.id
LEFT JOIN backorders b ON om.backorder_id = b.id
WHERE om.backorder_quantity > 0
  AND ($1::timestamp IS NULL OR om.created_at >= $1)
  AND ($2::timestamp IS NULL OR om.created_at <= $2)
ORDER BY om.created_at DESC;
