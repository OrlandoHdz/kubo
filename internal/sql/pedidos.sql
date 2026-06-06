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
    es_backorder, 
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
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

-- name: ListarPedidosDetalle :many
SELECT * FROM pedido_detalles
WHERE pedido_id = $1 AND deleted_at IS NULL;

-- name: ListarPedidos :many
SELECT * FROM pedidos
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListarPedidosPorCliente :many
SELECT * FROM pedidos
WHERE cliente_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ActualizarEstadoPedido :one
UPDATE pedidos
SET 
    estado = $2,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1 AND deleted_at IS NULL
RETURNING *;
-- name: CancelarDetallePedido :exec
UPDATE pedido_detalles
SET deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1 AND deleted_at IS NULL;
-- name: GetCommittedStock :one
SELECT COALESCE(SUM(pd.cantidad), 0)::INT as committed_stock
FROM pedido_detalles pd
JOIN pedidos p ON pd.pedido_id = p.id
WHERE pd.variante_id = $1 
  AND p.estado IN ('Pendiente', 'En Proceso') -- Estados que reservan stock
  AND pd.deleted_at IS NULL 
  AND p.deleted_at IS NULL;
