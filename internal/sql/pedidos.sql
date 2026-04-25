-- internal/sql/pedidos.sql

-- name: CrearPedido :one
-- Inicia el flujo de venta y reserva de crédito (cite: 234, 274)
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
    fecha_pedido,
    created_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: GetPedido :one
-- Obtiene el encabezado con los datos de auditoría
SELECT * FROM pedidos 
WHERE id = $1 AND deleted_at IS NULL LIMIT 1;

-- name: ListarPedidosPorCliente :many
-- Para el historial y re-order del cliente (cite: 215, 217)
SELECT * FROM pedidos 
WHERE cliente_id = $1 AND deleted_at IS NULL 
ORDER BY fecha_pedido DESC;

-- name: ActualizarEstadoPedido :exec
-- Controla el workflow: Pendiente -> Picking -> Tránsito -> Entregado (cite: 248)
UPDATE pedidos
SET 
    estado = $2,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $3
WHERE id = $1;

-- name: SoftDeletePedido :exec
-- Cancelación lógica del pedido
UPDATE pedidos
SET 
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2,
    estado = 'Cancelado'
WHERE id = $1;

-- name: RegistrarPartidaPedido :one
-- Agrega cada producto al detalle del pedido (cite: 259)
INSERT INTO pedido_detalles (
    pedido_id, 
    variante_id, 
    cantidad, 
    precio_unitario_aplicado,
    created_by
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ActualizarPartidaPedido :exec
UPDATE pedido_detalles
SET 
    cantidad = $2,
    precio_unitario_aplicado = $3,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = $4
WHERE id = $1;

-- name: SoftDeletePedidoDetalle :exec
-- Borrado lógico del detalle del pedido
UPDATE pedido_detalles
SET 
    deleted_at = CURRENT_TIMESTAMP,
    deleted_by = $2
WHERE id = $1;
